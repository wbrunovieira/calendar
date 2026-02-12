from __future__ import annotations

import json
import logging
from typing import Any

import anthropic

from app.agents.crm.prompts import (
    INVESTIGATOR_SYSTEM,
    INVESTIGATOR_USER,
    STRUCTURE_SYSTEM,
    STRUCTURE_USER,
    SUPERVISOR_SYSTEM,
    SUPERVISOR_USER,
)
from app.agents.crm.state import CRMLeadState
from app.agents.finances.nodes import langfuse
from app.clients import crm
from app.config import settings

logger = logging.getLogger("agents")

# ── Langfuse prompt helpers ─────────────────────────────────

_prompt_cache: dict[str, Any] = {}


def _get_langfuse_prompt(name: str) -> Any | None:
    if name in _prompt_cache:
        return _prompt_cache[name]
    if not langfuse.enabled:
        return None
    try:
        prompt = langfuse.get_prompt(name, type="chat", label="production")
        _prompt_cache[name] = prompt
        logger.info("Loaded prompt '%s' v%s from Langfuse", name, prompt.version)
        return prompt
    except Exception:
        logger.warning("Failed to fetch prompt '%s' from Langfuse, using local fallback", name)
        return None


def _build_investigator_messages(
    icp_context: str, query: str, count: int,
) -> tuple[str, str, Any | None]:
    prompt = _get_langfuse_prompt("crm-lead-investigator")
    if prompt is not None:
        compiled = prompt.compile(icp_context=icp_context, query=query, count=str(count))
        return compiled[0]["content"], compiled[1]["content"], prompt
    system = INVESTIGATOR_SYSTEM
    user = INVESTIGATOR_USER.format(icp_context=icp_context, query=query, count=count)
    return system, user, None


def _build_structure_messages(dossier: str) -> tuple[str, str, Any | None]:
    prompt = _get_langfuse_prompt("crm-lead-structurer")
    if prompt is not None:
        compiled = prompt.compile(dossier=dossier)
        return compiled[0]["content"], compiled[1]["content"], prompt
    system = STRUCTURE_SYSTEM
    user = STRUCTURE_USER.format(dossier=dossier)
    return system, user, None


def _build_supervisor_messages(
    icp_context: str, lead_json: str, contacts_json: str,
) -> tuple[str, str, Any | None]:
    prompt = _get_langfuse_prompt("crm-lead-supervisor")
    if prompt is not None:
        compiled = prompt.compile(icp_context=icp_context, lead_json=lead_json, contacts_json=contacts_json)
        return compiled[0]["content"], compiled[1]["content"], prompt
    system = SUPERVISOR_SYSTEM
    user = SUPERVISOR_USER.format(icp_context=icp_context, lead_json=lead_json, contacts_json=contacts_json)
    return system, user, None


# ── Claude helpers ──────────────────────────────────────────


def _get_client() -> anthropic.AsyncAnthropic:
    return anthropic.AsyncAnthropic(api_key=settings.anthropic_api_key)


async def _call_claude(
    system: str,
    user: str,
    model: str | None = None,
    tools: list | None = None,
    max_tokens: int = 4096,
) -> anthropic.types.Message:
    client = _get_client()
    kwargs: dict = {
        "model": model or settings.anthropic_model,
        "max_tokens": max_tokens,
        "system": system,
        "messages": [{"role": "user", "content": user}],
    }
    if tools:
        kwargs["tools"] = tools
    return await client.messages.create(**kwargs)


async def _call_claude_with_tool_loop(
    system: str,
    user: str,
    model: str | None = None,
    tools: list | None = None,
    max_tokens: int = 16384,
    max_rounds: int = 20,
) -> str:
    """Call Claude in a tool-use loop until it produces a final text response."""
    client = _get_client()
    messages: list[dict] = [{"role": "user", "content": user}]

    for _ in range(max_rounds):
        kwargs: dict = {
            "model": model or settings.anthropic_model,
            "max_tokens": max_tokens,
            "system": system,
            "messages": messages,
        }
        if tools:
            kwargs["tools"] = tools

        response = await client.messages.create(**kwargs)

        if response.stop_reason == "end_turn":
            text_parts = [b.text for b in response.content if b.type == "text"]
            return "\n".join(text_parts)

        if response.stop_reason == "tool_use":
            messages.append({"role": "assistant", "content": response.content})
            tool_results = []
            for block in response.content:
                if block.type == "tool_use":
                    tool_results.append({
                        "type": "tool_result",
                        "tool_use_id": block.id,
                        "content": "Search completed. Continue with the results above.",
                    })
            messages.append({"role": "user", "content": tool_results})
        else:
            text_parts = [b.text for b in response.content if b.type == "text"]
            return "\n".join(text_parts)

    text_parts = [b.text for b in response.content if b.type == "text"]
    return "\n".join(text_parts)


# ── Graph nodes ─────────────────────────────────────────────


async def load_icp_context(state: CRMLeadState) -> dict:
    """Fetch ICP from CRM, or use pre-loaded ICP if already in state."""
    trace = state.get("_trace")
    icp_id = state["icp_id"]

    # Skip API call if ICP was already provided in the request
    if state.get("icp_context"):
        if trace:
            span = trace.span(name="load_icp_context", input={"icp_id": icp_id})
            span.end(output={"icp_name": state["icp_context"].get("name", ""), "source": "pre-loaded"})
        return {}

    span = trace.span(name="load_icp_context", input={"icp_id": icp_id}) if trace else None

    try:
        icp = await crm.get_icp(icp_id)
    except LookupError:
        if span:
            span.end(output={"error": "icp_not_found"})
        return {
            "error": "icp_not_found",
            "reply": f"ICP '{icp_id}' not found",
        }
    except Exception as exc:
        logger.exception("Failed to load ICP %s", icp_id)
        if span:
            span.end(output={"error": str(exc)})
        return {
            "error": "icp_load_failed",
            "reply": f"Error loading ICP: {exc}",
        }

    if span:
        span.end(output={"icp_name": icp.get("name", "")})

    return {"icp_context": icp}


async def investigate_leads(state: CRMLeadState) -> dict:
    """Use Claude + web_search to research leads."""
    if state.get("error"):
        return {}

    trace = state.get("_trace")
    icp = state["icp_context"]
    query = state["query"]
    count = state.get("count", 1)

    icp_text = f"Name: {icp.get('name', '')}\n{icp.get('content', '')}"

    system, user, lf_prompt = _build_investigator_messages(icp_text, query, count)

    gen_kwargs: dict = {
        "name": "investigate_leads",
        "model": settings.anthropic_model,
        "input": [{"role": "system", "content": system}, {"role": "user", "content": user}],
        "metadata": {"prompt_source": "langfuse" if lf_prompt else "local"},
    }
    if lf_prompt is not None:
        gen_kwargs["prompt"] = lf_prompt

    generation = trace.generation(**gen_kwargs) if trace else None

    tools = [{"type": "web_search_20250305", "name": "web_search", "max_uses": 10}]

    try:
        raw_text = await _call_claude_with_tool_loop(system, user, tools=tools)
    except Exception as exc:
        logger.exception("Investigation failed")
        if generation:
            generation.end(output={"error": str(exc)})
        return {
            "error": "investigation_failed",
            "reply": f"Investigation failed: {exc}",
        }

    if generation:
        generation.end(output=raw_text[:2000])

    # Split dossiers by company separator
    dossiers = []
    if "=== COMPANY:" in raw_text:
        parts = raw_text.split("=== COMPANY:")
        for part in parts[1:]:
            dossiers.append("=== COMPANY:" + part.strip())
    else:
        # If no separator, treat entire response as a single dossier
        dossiers = [raw_text]

    return {"raw_dossiers": dossiers}


async def structure_leads_data(state: CRMLeadState) -> dict:
    """Convert each dossier into CRM-compatible JSON."""
    if state.get("error"):
        return {}

    trace = state.get("_trace")
    dossiers = state.get("raw_dossiers", [])

    leads_data: list[dict] = []
    contacts_data: list[list[dict]] = []

    for i, dossier in enumerate(dossiers):
        system, user, lf_prompt = _build_structure_messages(dossier)

        gen_kwargs: dict = {
            "name": f"structure_lead_{i}",
            "model": settings.anthropic_model,
            "input": [{"role": "system", "content": system}, {"role": "user", "content": user}],
        }
        if lf_prompt is not None:
            gen_kwargs["prompt"] = lf_prompt

        generation = trace.generation(**gen_kwargs) if trace else None

        try:
            response = await _call_claude(system, user)
            raw = "".join(b.text for b in response.content if b.type == "text").strip()
            parsed = json.loads(raw)
        except Exception:
            logger.exception("Failed to structure dossier %d", i)
            if generation:
                generation.end(output={"error": "parse_failed"})
            continue

        if generation:
            generation.end(output=parsed)

        if "error" in parsed:
            logger.warning("Dossier %d has error: %s", i, parsed["error"])
            continue

        lead = parsed.get("lead", {})
        if not lead.get("businessName"):
            logger.warning("Dossier %d missing businessName, skipping", i)
            continue

        leads_data.append(lead)
        contacts_data.append(parsed.get("contacts", []))

    return {"leads_data": leads_data, "contacts_data": contacts_data}


async def supervisor_review(state: CRMLeadState) -> dict:
    """Review each lead for quality. Uses a cheaper model."""
    if state.get("error"):
        return {}

    trace = state.get("_trace")
    leads = state.get("leads_data", [])
    contacts = state.get("contacts_data", [])
    icp = state.get("icp_context", {})

    icp_text = f"Name: {icp.get('name', '')}\n{icp.get('content', '')}"

    approved_indices: list[int] = []
    rejected: list[dict] = []

    for i, lead in enumerate(leads):
        lead_contacts = contacts[i] if i < len(contacts) else []

        system, user, lf_prompt = _build_supervisor_messages(
            icp_context=icp_text,
            lead_json=json.dumps(lead, ensure_ascii=False, indent=2),
            contacts_json=json.dumps(lead_contacts, ensure_ascii=False, indent=2),
        )

        gen_kwargs: dict = {
            "name": f"supervisor_review_{i}",
            "model": settings.anthropic_supervisor_model,
            "input": [{"role": "system", "content": system}, {"role": "user", "content": user}],
        }
        if lf_prompt is not None:
            gen_kwargs["prompt"] = lf_prompt

        generation = trace.generation(**gen_kwargs) if trace else None

        try:
            response = await _call_claude(system, user, model=settings.anthropic_supervisor_model)
            raw = "".join(b.text for b in response.content if b.type == "text").strip()
            review = json.loads(raw)
        except Exception:
            logger.exception("Supervisor review failed for lead %d", i)
            if generation:
                generation.end(output={"error": "review_failed"})
            rejected.append({
                "query_term": lead.get("businessName", "Unknown"),
                "reason": "Supervisor review failed",
            })
            continue

        if generation:
            generation.end(output=review)

        if review.get("approved"):
            approved_indices.append(i)
        else:
            rejected.append({
                "query_term": lead.get("businessName", "Unknown"),
                "reason": review.get("notes", "Rejected by supervisor"),
            })

    return {"approved_indices": approved_indices, "rejected": rejected}


async def save_to_crm(state: CRMLeadState) -> dict:
    """Save approved leads to CRM."""
    if state.get("error"):
        return {}

    trace = state.get("_trace")
    leads = state.get("leads_data", [])
    contacts = state.get("contacts_data", [])
    approved = state.get("approved_indices", [])

    span = trace.span(name="save_to_crm", input={"approved_count": len(approved)}) if trace else None

    created_leads: list[dict] = []

    for idx in approved:
        lead_data = leads[idx]
        lead_contacts = contacts[idx] if idx < len(contacts) else []

        try:
            lead_result = await crm.create_lead(lead_data)
        except Exception as exc:
            logger.exception("Failed to create lead %s", lead_data.get("businessName"))
            continue

        saved_contacts = []
        lead_id = lead_result.get("id", "")
        for contact in lead_contacts:
            try:
                contact_result = await crm.create_lead_contact(lead_id, contact)
                saved_contacts.append(contact_result)
            except Exception:
                logger.exception("Failed to create contact for lead %s", lead_id)

        created_leads.append({
            "lead": lead_result,
            "contacts": saved_contacts,
            "supervisor_notes": "",
        })

    if span:
        span.end(output={"created": len(created_leads)})

    return {"created_leads": created_leads}


async def format_reply(state: CRMLeadState) -> dict:
    """Build the final reply message."""
    created = state.get("created_leads", [])
    rejected = state.get("rejected", [])
    error = state.get("error")

    if error:
        return {}

    total_created = len(created)
    total_rejected = len(rejected)

    if total_created == 0 and total_rejected > 0:
        names = ", ".join(r.get("query_term", "?") for r in rejected)
        return {
            "reply": f"All {total_rejected} leads rejected by supervisor: {names}",
            "error": "all_rejected",
        }

    parts = []
    if total_created > 0:
        names = ", ".join(c["lead"].get("businessName", "?") for c in created)
        parts.append(f"{total_created} lead(s) created: {names}")
    if total_rejected > 0:
        parts.append(f"{total_rejected} rejected")

    return {"reply": ". ".join(parts)}
