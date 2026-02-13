from __future__ import annotations

import json
import logging
from typing import Any

import anthropic
import httpx

from app.agents.crm.prompts import (
    INVESTIGATOR_SYSTEM,
    INVESTIGATOR_USER,
    INVESTIGATOR_USER_TAVILY,
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
    icp_context: str, query: str, count: int, existing_leads: list[str] | None = None,
) -> tuple[str, str, Any | None]:
    existing_section = ""
    if existing_leads:
        names = ", ".join(existing_leads)
        existing_section = (
            f"\n**IMPORTANT — These companies are already in the CRM. DO NOT research them again:**\n"
            f"{names}\n"
        )
    prompt = _get_langfuse_prompt("crm-lead-investigator")
    if prompt is not None:
        compiled = prompt.compile(
            icp_context=icp_context, query=query, count=str(count),
            existing_leads_section=existing_section,
        )
        return compiled[0]["content"], compiled[1]["content"], prompt
    system = INVESTIGATOR_SYSTEM
    user = INVESTIGATOR_USER.format(
        icp_context=icp_context, query=query, count=count,
        existing_leads_section=existing_section,
    )
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
    lead_json: str, contacts_json: str,
) -> tuple[str, str, Any | None]:
    prompt = _get_langfuse_prompt("crm-lead-supervisor")
    if prompt is not None:
        compiled = prompt.compile(lead_json=lead_json, contacts_json=contacts_json)
        return compiled[0]["content"], compiled[1]["content"], prompt
    system = SUPERVISOR_SYSTEM
    user = SUPERVISOR_USER.format(lead_json=lead_json, contacts_json=contacts_json)
    return system, user, None


# ── Tavily search ─────────────────────────────────────────


async def _tavily_search(queries: list[str], max_results: int = 5) -> list[dict]:
    """Run Tavily searches and return combined results."""
    all_results: list[dict] = []
    async with httpx.AsyncClient(timeout=30) as client:
        for query in queries:
            try:
                resp = await client.post(
                    "https://api.tavily.com/search",
                    json={
                        "api_key": settings.tavily_api_key,
                        "query": query,
                        "max_results": max_results,
                        "search_depth": "advanced",
                        "include_answer": True,
                    },
                )
                resp.raise_for_status()
                data = resp.json()
                all_results.append({
                    "query": query,
                    "answer": data.get("answer", ""),
                    "results": data.get("results", []),
                })
            except Exception:
                logger.warning("Tavily search failed for query: %s", query)
    return all_results


def _format_tavily_results(tavily_data: list[dict]) -> str:
    """Format Tavily search results as text context for LLM."""
    parts: list[str] = []
    seen_urls: set[str] = set()
    for group in tavily_data:
        if group.get("answer"):
            parts.append(f"### AI Summary for '{group['query']}'\n{group['answer']}\n")
        for r in group.get("results", []):
            url = r.get("url", "")
            if url in seen_urls:
                continue
            seen_urls.add(url)
            title = r.get("title", "No title")
            content = r.get("content", "")
            parts.append(f"**{title}** ({url})\n{content}\n")
    return "\n".join(parts)


def _build_tavily_queries(query: str, icp_context: dict, count: int, country: str = "Brasil") -> list[str]:
    """Generate search queries for Tavily from ICP + user query + country."""
    queries = [
        f"{query} {country}",
        f"{query} {country} CNPJ site oficial endereço telefone email",
        f"{query} {country} CEO fundador diretor LinkedIn email",
    ]
    if count > 1:
        segment = icp_context.get("name", "")
        if segment:
            queries.append(f"{segment} empresas {country} {query}")
    return queries


# ── Claude helpers ──────────────────────────────────────────


class AgentError(Exception):
    """User-friendly error from agent operations."""

    def __init__(self, message: str, code: str = "agent_error"):
        super().__init__(message)
        self.code = code


def _get_client() -> anthropic.AsyncAnthropic:
    return anthropic.AsyncAnthropic(api_key=settings.anthropic_api_key)


def _handle_anthropic_error(exc: Exception) -> AgentError:
    """Convert Anthropic SDK exceptions to user-friendly AgentError."""
    msg = str(exc)
    if isinstance(exc, anthropic.AuthenticationError):
        return AgentError("Anthropic API key is invalid or expired. Contact the administrator.", "auth_error")
    if isinstance(exc, anthropic.BadRequestError):
        if "credit balance" in msg.lower() or "billing" in msg.lower():
            return AgentError("Anthropic API credits exhausted. Please add credits at console.anthropic.com.", "billing_error")
        return AgentError(f"Invalid request to AI provider: {msg}", "bad_request")
    if isinstance(exc, anthropic.RateLimitError):
        return AgentError("AI provider rate limit reached. Please try again in a few minutes.", "rate_limit")
    if isinstance(exc, anthropic.APIStatusError):
        return AgentError(f"AI provider error (HTTP {exc.status_code}). Please try again later.", "api_error")
    if isinstance(exc, TypeError) and "authentication" in msg.lower():
        return AgentError("Anthropic API key is not configured. Contact the administrator.", "auth_error")
    return AgentError(f"Unexpected AI error: {msg}", "agent_error")


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
    try:
        return await client.messages.create(**kwargs)
    except Exception as exc:
        raise _handle_anthropic_error(exc) from exc


def _parse_json_lenient(raw: str) -> dict:
    """Parse JSON leniently — handles extra data after the first valid object."""
    try:
        return json.loads(raw)
    except json.JSONDecodeError as exc:
        if "Extra data" in str(exc):
            decoder = json.JSONDecoder()
            obj, _ = decoder.raw_decode(raw)
            return obj
        raise


def _extract_usage(response: anthropic.types.Message) -> dict:
    """Extract token usage from an Anthropic response."""
    usage = response.usage
    return {
        "input": usage.input_tokens,
        "output": usage.output_tokens,
        "total": usage.input_tokens + usage.output_tokens,
        "unit": "TOKENS",
    }


async def _call_claude_with_tool_loop(
    system: str,
    user: str,
    model: str | None = None,
    tools: list | None = None,
    max_tokens: int = 16384,
    max_rounds: int = 20,
) -> tuple[str, dict]:
    """Call Claude in a tool-use loop. Returns (text, accumulated_usage)."""
    client = _get_client()
    messages: list[dict] = [{"role": "user", "content": user}]
    total_input = 0
    total_output = 0

    for _ in range(max_rounds):
        kwargs: dict = {
            "model": model or settings.anthropic_model,
            "max_tokens": max_tokens,
            "system": system,
            "messages": messages,
        }
        if tools:
            kwargs["tools"] = tools

        try:
            response = await client.messages.create(**kwargs)
        except Exception as exc:
            raise _handle_anthropic_error(exc) from exc

        total_input += response.usage.input_tokens
        total_output += response.usage.output_tokens

        accumulated = {"input": total_input, "output": total_output, "total": total_input + total_output, "unit": "TOKENS"}

        if response.stop_reason == "end_turn":
            text_parts = [b.text for b in response.content if b.type == "text"]
            return "\n".join(text_parts), accumulated

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
            return "\n".join(text_parts), accumulated

    text_parts = [b.text for b in response.content if b.type == "text"]
    return "\n".join(text_parts), accumulated


# ── Graph nodes ─────────────────────────────────────────────


async def load_icp_context(state: CRMLeadState) -> dict:
    """Fetch ICP from CRM, or use pre-loaded ICP if already in state."""
    trace = state.get("_trace")
    icp_id = state["icp_id"]

    span = trace.span(name="load_icp_context", input={"icp_id": icp_id}) if trace else None

    # Fetch existing leads for dedup (regardless of ICP source)
    existing_leads: list[str] = []
    try:
        leads_list = await crm.get_leads_by_icp(icp_id)
        existing_leads = [
            lead.get("businessName", "") for lead in leads_list
            if lead.get("businessName")
        ]
        if existing_leads:
            logger.info("Found %d existing leads for ICP %s: %s", len(existing_leads), icp_id, existing_leads)
    except Exception:
        logger.warning("Failed to fetch existing leads for ICP %s, continuing without dedup", icp_id)

    # Skip ICP API call if ICP was already provided in the request
    if state.get("icp_context"):
        if span:
            span.end(output={
                "icp_name": state["icp_context"].get("name", ""),
                "source": "pre-loaded",
                "existing_leads": len(existing_leads),
            })
        return {"existing_leads": existing_leads}

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
        span.end(output={"icp_name": icp.get("name", ""), "existing_leads": len(existing_leads)})

    return {"icp_context": icp, "existing_leads": existing_leads}


async def investigate_leads(state: CRMLeadState) -> dict:
    """Research leads using Tavily search + Haiku, with Sonnet+web_search fallback."""
    if state.get("error"):
        return {}

    trace = state.get("_trace")
    icp = state["icp_context"]
    query = state["query"]
    count = state.get("count", 1)
    country = state.get("country", "Brasil")

    icp_text = f"Name: {icp.get('name', '')}\n{icp.get('content', '')}"
    existing_leads = state.get("existing_leads", [])

    # ── Tavily mode (primary) ────────────────────────────────
    if settings.tavily_api_key:
        return await _investigate_with_tavily(
            trace, icp, icp_text, query, count, existing_leads, country,
        )

    # ── Sonnet + web_search fallback ─────────────────────────
    return await _investigate_with_web_search(
        trace, icp_text, query, count, existing_leads,
    )


async def _investigate_with_tavily(
    trace: Any, icp: dict, icp_text: str, query: str, count: int,
    existing_leads: list[str], country: str = "Brasil",
) -> dict:
    """Tavily search + Haiku to build dossiers."""
    # 1. Generate and run search queries
    queries = _build_tavily_queries(query, icp, count, country)

    search_span = trace.span(name="tavily_search", input={"queries": queries}) if trace else None

    try:
        tavily_data = await _tavily_search(queries)
    except Exception as exc:
        logger.exception("Tavily search failed")
        if search_span:
            search_span.end(output={"error": str(exc)})
        return {
            "error": "investigation_failed",
            "reply": "Web search failed. Please try again.",
        }

    total_results = sum(len(g.get("results", [])) for g in tavily_data)
    if search_span:
        search_span.end(output={"total_results": total_results, "queries_run": len(tavily_data)})

    if total_results == 0:
        return {
            "error": "investigation_failed",
            "reply": "No search results found. Try a different query.",
        }

    # 2. Format results and call Haiku to write dossier
    search_context = _format_tavily_results(tavily_data)

    existing_section = ""
    if existing_leads:
        names = ", ".join(existing_leads)
        existing_section = (
            f"\n**IMPORTANT — These companies are already in the CRM. DO NOT include them:**\n"
            f"{names}\n"
        )

    system = INVESTIGATOR_SYSTEM
    user = INVESTIGATOR_USER_TAVILY.format(
        icp_context=icp_text, query=query, count=count,
        existing_leads_section=existing_section,
        search_results=search_context,
    )

    generation = trace.generation(
        name="investigate_leads",
        model=settings.anthropic_supervisor_model,
        input=[{"role": "system", "content": system}, {"role": "user", "content": user}],
        metadata={"search_provider": "tavily", "search_results": total_results},
    ) if trace else None

    try:
        response = await _call_claude(system, user, model=settings.anthropic_supervisor_model, max_tokens=8192)
        usage = _extract_usage(response)
        raw_text = "".join(b.text for b in response.content if b.type == "text")
    except AgentError as exc:
        logger.error("Investigation (Tavily+Haiku) failed: %s [%s]", exc, exc.code)
        if generation:
            generation.end(output={"error": str(exc)})
        return {"error": exc.code, "reply": str(exc)}
    except Exception as exc:
        logger.exception("Investigation (Tavily+Haiku) failed")
        if generation:
            generation.end(output={"error": str(exc)})
        return {
            "error": "investigation_failed",
            "reply": "Investigation failed due to an unexpected error. Please try again.",
        }

    if generation:
        generation.end(output=raw_text[:2000], usage=usage)

    result = _split_dossiers(raw_text)
    result["tavily_data"] = tavily_data
    return result


async def _investigate_with_web_search(
    trace: Any, icp_text: str, query: str, count: int,
    existing_leads: list[str],
) -> dict:
    """Fallback: Sonnet + Claude web_search tool loop."""
    system, user, lf_prompt = _build_investigator_messages(icp_text, query, count, existing_leads)

    gen_kwargs: dict = {
        "name": "investigate_leads",
        "model": settings.anthropic_model,
        "input": [{"role": "system", "content": system}, {"role": "user", "content": user}],
        "metadata": {"search_provider": "claude_web_search"},
    }
    if lf_prompt is not None:
        gen_kwargs["prompt"] = lf_prompt

    generation = trace.generation(**gen_kwargs) if trace else None

    tools = [{"type": "web_search_20250305", "name": "web_search", "max_uses": 5}]

    try:
        raw_text, usage = await _call_claude_with_tool_loop(system, user, tools=tools)
    except AgentError as exc:
        logger.error("Investigation failed: %s [%s]", exc, exc.code)
        if generation:
            generation.end(output={"error": str(exc)})
        return {"error": exc.code, "reply": str(exc)}
    except Exception:
        logger.exception("Investigation failed")
        if generation:
            generation.end(output={"error": "unexpected"})
        return {
            "error": "investigation_failed",
            "reply": "Investigation failed due to an unexpected error. Please try again.",
        }

    if generation:
        generation.end(output=raw_text[:2000], usage=usage)

    return _split_dossiers(raw_text)


def _split_dossiers(raw_text: str) -> dict:
    """Split raw investigation text into individual dossiers."""
    dossiers: list[str] = []
    if "=== COMPANY:" in raw_text:
        parts = raw_text.split("=== COMPANY:")
        for part in parts[1:]:
            dossiers.append("=== COMPANY:" + part.strip())
    else:
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
            "model": settings.anthropic_supervisor_model,
            "input": [{"role": "system", "content": system}, {"role": "user", "content": user}],
        }
        if lf_prompt is not None:
            gen_kwargs["prompt"] = lf_prompt

        generation = trace.generation(**gen_kwargs) if trace else None

        try:
            response = await _call_claude(system, user, model=settings.anthropic_supervisor_model)
            usage = _extract_usage(response)
            raw = "".join(b.text for b in response.content if b.type == "text").strip()
            # Strip markdown code fences if present
            if raw.startswith("```"):
                raw = raw.split("\n", 1)[1] if "\n" in raw else raw[3:]
            if raw.endswith("```"):
                raw = raw[:-3].rstrip()
            parsed = _parse_json_lenient(raw)
        except AgentError as exc:
            logger.error("Failed to structure dossier %d: %s [%s]", i, exc, exc.code)
            if generation:
                generation.end(output={"error": str(exc)})
            return {"error": exc.code, "reply": str(exc)}
        except Exception:
            logger.exception("Failed to structure dossier %d", i)
            if generation:
                generation.end(output={"error": "parse_failed"})
            continue

        if generation:
            generation.end(output=parsed, usage=usage)

        lead = parsed.get("lead", {})
        needs_retry = "error" in parsed or not lead.get("businessName")

        if needs_retry:
            logger.info("Dossier %d missing businessName, retrying with correction prompt", i)
            retry_user = (
                "The previous attempt failed to extract a businessName. "
                "Here is the original dossier. Please try again — extract the company/brand name "
                "even if it's not perfectly formatted. Use the most prominent organization mentioned.\n\n"
                f"{dossier}"
            )
            retry_gen = trace.generation(
                name=f"structure_lead_{i}_retry",
                model=settings.anthropic_supervisor_model,
                input=[{"role": "system", "content": system}, {"role": "user", "content": retry_user}],
            ) if trace else None

            try:
                retry_resp = await _call_claude(system, retry_user, model=settings.anthropic_supervisor_model)
                retry_usage = _extract_usage(retry_resp)
                retry_raw = "".join(b.text for b in retry_resp.content if b.type == "text").strip()
                if retry_raw.startswith("```"):
                    retry_raw = retry_raw.split("\n", 1)[1] if "\n" in retry_raw else retry_raw[3:]
                if retry_raw.endswith("```"):
                    retry_raw = retry_raw[:-3].rstrip()
                parsed = _parse_json_lenient(retry_raw)
                lead = parsed.get("lead", {})
                if retry_gen:
                    retry_gen.end(output=parsed, usage=retry_usage)
            except Exception:
                logger.exception("Retry structuring dossier %d also failed", i)
                if retry_gen:
                    retry_gen.end(output={"error": "retry_failed"})

            if not lead.get("businessName"):
                logger.warning("Dossier %d still missing businessName after retry, skipping", i)
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

    approved_indices: list[int] = []
    rejected: list[dict] = []

    for i, lead in enumerate(leads):
        lead_contacts = contacts[i] if i < len(contacts) else []

        system, user, lf_prompt = _build_supervisor_messages(
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
            usage = _extract_usage(response)
            raw = "".join(b.text for b in response.content if b.type == "text").strip()
            # Strip markdown code fences if present
            if raw.startswith("```"):
                raw = raw.split("\n", 1)[1] if "\n" in raw else raw[3:]
            if raw.endswith("```"):
                raw = raw[:-3].rstrip()
            review = json.loads(raw)
        except AgentError as exc:
            logger.error("Supervisor review failed for lead %d: %s [%s]", i, exc, exc.code)
            if generation:
                generation.end(output={"error": str(exc)})
            return {"error": exc.code, "reply": str(exc)}
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
            generation.end(output=review, usage=usage)

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
    icp_id = state.get("icp_id", "")

    span = trace.span(name="save_to_crm", input={"approved_count": len(approved)}) if trace else None

    created_leads: list[dict] = []

    for idx in approved:
        lead_data = leads[idx]
        lead_contacts = contacts[idx] if idx < len(contacts) else []

        try:
            lead_result = await crm.create_lead(lead_data)
        except Exception:
            logger.exception("Failed to create lead %s", lead_data.get("businessName"))
            continue

        lead_id = lead_result.get("id", "")

        # Link lead to ICP
        if lead_id and icp_id:
            try:
                await crm.link_lead_to_icp(lead_id, icp_id)
            except Exception:
                logger.warning("Failed to link lead %s to ICP %s", lead_id, icp_id)

        saved_contacts = []
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

    if total_created == 0 and total_rejected == 0:
        return {
            "reply": "No leads could be processed. The data could not be structured. Please try again.",
            "error": "no_leads_processed",
        }

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


# ── Shadow experiment (A/B test) ──────────────────────────


async def run_shadow_investigation(
    trace: Any,
    icp_context: dict,
    query: str,
    count: int,
    existing_leads: list[str],
    tavily_data: list[dict] | None = None,
) -> None:
    """Run a shadow investigation with DeepSeek using the same Tavily results.

    Reuses Tavily search results from the main flow (no extra search cost).
    Results are logged to Langfuse only (not saved to CRM).
    """
    if not settings.deepseek_api_key or not tavily_data:
        return

    model = settings.deepseek_model
    icp_text = f"Name: {icp_context.get('name', '')}\n{icp_context.get('content', '')}"

    existing_section = ""
    if existing_leads:
        names = ", ".join(existing_leads)
        existing_section = (
            f"\n**IMPORTANT — These companies are already in the CRM. DO NOT include them:**\n"
            f"{names}\n"
        )

    search_context = _format_tavily_results(tavily_data)
    system = INVESTIGATOR_SYSTEM
    user = INVESTIGATOR_USER_TAVILY.format(
        icp_context=icp_text, query=query, count=count,
        existing_leads_section=existing_section,
        search_results=search_context,
    )

    generation = None
    if trace:
        generation = trace.generation(
            name="investigate_leads_experiment",
            model=model,
            input=[{"role": "system", "content": system}, {"role": "user", "content": user}],
            metadata={"provider": "deepseek", "experiment": True},
        )

    try:
        async with httpx.AsyncClient(timeout=120) as client:
            resp = await client.post(
                f"{settings.deepseek_base_url}/chat/completions",
                headers={
                    "Authorization": f"Bearer {settings.deepseek_api_key}",
                    "Content-Type": "application/json",
                },
                json={
                    "model": model,
                    "messages": [
                        {"role": "system", "content": system},
                        {"role": "user", "content": user},
                    ],
                    "max_tokens": 8192,
                },
            )
            resp.raise_for_status()
            data = resp.json()

        text = data["choices"][0]["message"]["content"]
        usage_data = data.get("usage", {})
        usage = {
            "input": usage_data.get("prompt_tokens", 0),
            "output": usage_data.get("completion_tokens", 0),
            "total": usage_data.get("total_tokens", 0),
            "unit": "TOKENS",
        }

        if generation:
            generation.end(output=text[:3000], usage=usage)

        logger.info(
            "Shadow investigation (%s) completed: %d chars, %d tokens",
            model, len(text), usage["total"],
        )

    except Exception as exc:
        logger.warning("Shadow investigation (%s) failed: %s", model, exc)
        if generation:
            generation.end(output={"error": str(exc)})

    if langfuse.enabled:
        langfuse.flush()
