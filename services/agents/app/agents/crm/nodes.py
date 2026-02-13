from __future__ import annotations

import json
import logging
import re
from typing import Any

import anthropic
import httpx

from app.agents.crm.prompts import (
    CONTACT_ENRICHER_SYSTEM,
    CONTACT_ENRICHER_USER,
    INVESTIGATOR_SYSTEM,
    INVESTIGATOR_USER,
    INVESTIGATOR_USER_TAVILY,
    QUERY_PLANNER_SYSTEM,
    QUERY_PLANNER_USER,
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


async def _plan_search_queries(
    query: str, icp_text: str, country: str,
    existing_leads: list[str], trace: Any = None,
    previous_queries: list[str] | None = None,
) -> list[str]:
    """Use Haiku to generate smart search queries based on ICP + context."""
    existing_str = ", ".join(existing_leads) if existing_leads else "(none)"

    if query:
        query_section = f'"{query}" in {country}'
    else:
        query_section = f"No specific search term provided. Generate queries based on the ICP above, targeting {country}."

    system = QUERY_PLANNER_SYSTEM
    user = QUERY_PLANNER_USER.format(
        icp_context=icp_text, query_section=query_section, country=country,
        existing_leads=existing_str,
    )

    # On retry rounds, tell the planner to avoid previous queries
    if previous_queries:
        prev_str = "\n".join(f"- {q}" for q in previous_queries)
        user += (
            f"\n\n## Previous search queries (DO NOT repeat these — use DIFFERENT terms)\n"
            f"{prev_str}\n"
        )

    generation = trace.generation(
        name="plan_search_queries",
        model=settings.anthropic_supervisor_model,
        input=[{"role": "system", "content": system}, {"role": "user", "content": user}],
    ) if trace else None

    try:
        response = await _call_claude(system, user, model=settings.anthropic_supervisor_model, max_tokens=512)
        usage = _extract_usage(response)
        raw = "".join(b.text for b in response.content if b.type == "text").strip()
        # Strip markdown code fences if present
        if raw.startswith("```"):
            raw = raw.split("\n", 1)[1] if "\n" in raw else raw[3:]
        if raw.endswith("```"):
            raw = raw[:-3].rstrip()
        queries = json.loads(raw)
        if generation:
            generation.end(output=queries, usage=usage)
        if isinstance(queries, list) and all(isinstance(q, str) for q in queries):
            return queries[:5]
    except Exception:
        logger.warning("Query planner failed, using fallback queries")
        if generation:
            generation.end(output={"error": "fallback"})

    # Fallback: programmatic queries
    base = query if query else "empresas"
    return [
        f"{base} {country}",
        f"{base} {country} CNPJ site oficial telefone email",
        f"{base} {country} CEO fundador diretor LinkedIn",
    ]


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


_DECISION_MAKER_ROLES = {
    "ceo", "cto", "cfo", "coo", "cmo", "cpo", "cio", "cro",
    "founder", "co-founder", "cofounder", "fundador", "cofundador",
    "owner", "sócio", "socio", "proprietário", "proprietario",
    "president", "presidente",
    "director", "diretor", "diretora",
    "vp", "vice-president", "vice-presidente",
    "head", "chief",
    "partner", "managing",
    "gerente", "manager",
    "coordenador", "coordenadora", "coordinator",
    "supervisor", "superintendente",
    "lead", "líder", "lider",
}


def _is_decision_maker(contact: dict) -> bool:
    """Check if a contact is a real decision-maker based on role/title."""
    name = contact.get("name", "")
    role = contact.get("role", "")
    # Must have a real name (first + last)
    if not name or len(name.split()) < 2:
        return False
    # Must have a recognizable decision-maker role
    if not role:
        return False
    role_lower = role.lower()
    return any(kw in role_lower for kw in _DECISION_MAKER_ROLES)


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
    country = state.get("country", "Brasil")
    existing_leads = state.get("existing_leads", [])

    # On retry rounds, use remaining_count instead of original count
    remaining = state.get("remaining_count", 0)
    retry_round = state.get("retry_round", 0)
    count = remaining if remaining > 0 else state.get("count", 1)

    icp_text = f"Name: {icp.get('name', '')}\n{icp.get('content', '')}"

    previous_queries = state.get("previous_queries", [])
    logger.info("Investigate round %d: count=%d, existing=%d, prev_queries=%d", retry_round + 1, count, len(existing_leads), len(previous_queries))

    # ── Tavily mode (primary) ────────────────────────────────
    if settings.tavily_api_key:
        result = await _investigate_with_tavily(
            trace, icp, icp_text, query, count, existing_leads, country,
            previous_queries=previous_queries if previous_queries else None,
        )
        # Track retry round
        result["retry_round"] = retry_round + 1
        return result

    # ── Sonnet + web_search fallback ─────────────────────────
    result = await _investigate_with_web_search(
        trace, icp_text, query, count, existing_leads,
    )
    result["retry_round"] = retry_round + 1
    return result


async def _investigate_with_tavily(
    trace: Any, icp: dict, icp_text: str, query: str, count: int,
    existing_leads: list[str], country: str = "Brasil",
    previous_queries: list[str] | None = None,
) -> dict:
    """Tavily search + Haiku to build dossiers, with smart query planning."""
    all_tavily_data: list[dict] = []
    all_queries_used: list[str] = list(previous_queries or [])

    # Up to 2 rounds: initial search + optional retry with new queries
    for round_num in range(2):
        # 1. Plan search queries using Haiku
        queries = await _plan_search_queries(
            query, icp_text, country, existing_leads, trace,
            previous_queries=all_queries_used if all_queries_used else None,
        )
        logger.info("Search round %d queries: %s", round_num + 1, queries)

        # 2. Run Tavily search
        search_span = trace.span(
            name=f"tavily_search_round_{round_num + 1}",
            input={"queries": queries, "round": round_num + 1},
        ) if trace else None

        try:
            tavily_data = await _tavily_search(queries)
        except Exception as exc:
            logger.exception("Tavily search failed (round %d)", round_num + 1)
            if search_span:
                search_span.end(output={"error": str(exc)})
            if all_tavily_data:
                break  # Use results from previous round
            return {
                "error": "investigation_failed",
                "reply": "Web search failed. Please try again.",
            }

        total_results = sum(len(g.get("results", [])) for g in tavily_data)
        if search_span:
            search_span.end(output={"total_results": total_results, "queries_run": len(tavily_data)})

        all_tavily_data.extend(tavily_data)
        all_queries_used.extend(queries)

        if total_results == 0 and round_num == 0:
            # No results at all on first round — retry
            logger.info("No results in round 1, retrying with new queries")
            continue

        # Check if results contain any NEW companies (not already in CRM)
        if round_num == 0 and existing_leads:
            result_text = _format_tavily_results(tavily_data).lower()
            existing_lower = [name.lower() for name in existing_leads]
            has_new = any(
                name not in result_text
                for name in existing_lower
            )
            # If ALL existing companies appear in results and we have few unique results,
            # the search is probably too generic — retry
            all_existing_found = sum(1 for name in existing_lower if name in result_text)
            if all_existing_found >= min(3, len(existing_lower)) and not has_new:
                logger.info(
                    "Round 1 results overlap heavily with existing leads (%d/%d), retrying",
                    all_existing_found, len(existing_lower),
                )
                continue

        break  # Good results, no need to retry

    # 3. Format all results and call Haiku to write dossier
    search_context = _format_tavily_results(all_tavily_data)

    if not search_context.strip():
        return {
            "error": "investigation_failed",
            "reply": "No search results found. Try a different query.",
        }

    existing_section = ""
    if existing_leads:
        names = ", ".join(existing_leads)
        existing_section = (
            f"\n**IMPORTANT — These companies are already in the CRM. DO NOT include them:**\n"
            f"{names}\n"
        )

    query_hint = f" for the query: **{query}**" if query else ""

    system = INVESTIGATOR_SYSTEM
    user = INVESTIGATOR_USER_TAVILY.format(
        icp_context=icp_text, query_hint=query_hint, count=count,
        existing_leads_section=existing_section,
        search_results=search_context,
    )

    generation = trace.generation(
        name="investigate_leads",
        model=settings.anthropic_supervisor_model,
        input=[{"role": "system", "content": system}, {"role": "user", "content": user}],
        metadata={"search_provider": "tavily", "total_results": len(all_tavily_data)},
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

    # Programmatic dedup: remove dossiers for companies already in CRM
    if existing_leads:
        filtered = []
        for dossier in result.get("raw_dossiers", []):
            company_name = _extract_company_name(dossier)
            matched = _is_duplicate(company_name, existing_leads)
            if matched:
                logger.info("Dedup: filtered out '%s' (matches existing '%s')", company_name, matched)
                continue
            filtered.append(dossier)
        if len(filtered) < len(result["raw_dossiers"]):
            logger.info("Dedup: %d/%d dossiers kept", len(filtered), len(result["raw_dossiers"]))
        result["raw_dossiers"] = filtered

    # Enforce requested count — LLM may produce more dossiers than asked
    if len(result.get("raw_dossiers", [])) > count:
        result["raw_dossiers"] = result["raw_dossiers"][:count]
    result["tavily_data"] = all_tavily_data
    result["previous_queries"] = all_queries_used
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


def _extract_company_name(dossier: str) -> str:
    """Extract company name from dossier header '=== COMPANY: Name ==='."""
    if "=== COMPANY:" not in dossier:
        return ""
    after = dossier.split("=== COMPANY:", 1)[1]
    # Name ends at '===' or newline
    for delimiter in ["===", "\n"]:
        if delimiter in after:
            return after.split(delimiter, 1)[0].strip()
    return after.strip()


_COMPANY_SUFFIXES = re.compile(
    r"\b(ltda|s\.?a\.?|s/a|me|eireli|epp|"
    r"participações|participacoes|"
    r"group|grupo|editora|instituto)\b",
    re.IGNORECASE,
)


def _normalize_company_name(name: str) -> str:
    """Normalize company name by stripping suffixes and extra whitespace."""
    normalized = _COMPANY_SUFFIXES.sub("", name.lower())
    normalized = re.sub(r"[.\-/,()]+", " ", normalized)
    return re.sub(r"\s+", " ", normalized).strip()


def _is_duplicate(candidate: str, existing_leads: list[str]) -> str | None:
    """Check if candidate name matches any existing lead. Returns matched name or None."""
    if not candidate:
        return None
    norm_candidate = _normalize_company_name(candidate)
    if not norm_candidate:
        return None
    for existing in existing_leads:
        norm_existing = _normalize_company_name(existing)
        if not norm_existing:
            continue
        # Exact match after normalization
        if norm_candidate == norm_existing:
            return existing
        # Substring match (bidirectional) — require the shorter string to be
        # at least 5 chars to avoid false positives ("portal", "med", etc.)
        shorter = min(len(norm_candidate), len(norm_existing))
        if shorter >= 5:
            if norm_candidate in norm_existing or norm_existing in norm_candidate:
                return existing
    return None


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
    enrichment_indices: list[int] = []
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
            review = _parse_json_lenient(raw)
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

        verdict = review.get("verdict", "")
        # Backwards compat: old prompt used "approved": true/false
        if not verdict:
            verdict = "approved" if review.get("approved") else "rejected"

        if verdict == "approved":
            # Double-check: filter out generic contacts and verify real people remain
            real_contacts = [c for c in lead_contacts if _is_decision_maker(c)]
            if real_contacts:
                contacts[i] = real_contacts
                approved_indices.append(i)
            else:
                # LLM approved but contacts are not real people — send to enrichment
                enrichment_indices.append(i)
        elif verdict == "needs_enrichment":
            enrichment_indices.append(i)
        else:
            rejected.append({
                "query_term": lead.get("businessName", "Unknown"),
                "reason": review.get("notes", "Rejected by supervisor"),
            })

    return {
        "approved_indices": approved_indices,
        "enrichment_indices": enrichment_indices,
        "rejected": rejected,
    }


_LEAD_STRING_FIELDS = [
    "businessName", "contactName", "email", "phone", "website",
    "address", "cnpj", "description", "source", "status",
]

_EMAIL_RE = re.compile(r"^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$")


def _is_valid_email(value: str) -> bool:
    """Check if a string is a valid email address."""
    return bool(value and _EMAIL_RE.match(value.strip()))


def _sanitize_lead_data(lead: dict) -> dict:
    """Ensure all string fields are strings (not None/null) for CRM API."""
    sanitized = dict(lead)
    for field in _LEAD_STRING_FIELDS:
        if field in sanitized and sanitized[field] is None:
            sanitized[field] = ""
    # Validate email — CRM rejects invalid formats
    if sanitized.get("email") and not _is_valid_email(sanitized["email"]):
        logger.warning("Invalid lead email stripped: %s", sanitized["email"])
        sanitized["email"] = ""
    return sanitized


def _sanitize_contact_data(contact: dict) -> dict:
    """Sanitize contact data before sending to CRM API."""
    sanitized = dict(contact)
    for field in ("name", "role", "email", "phone", "linkedin"):
        if field in sanitized and sanitized[field] is None:
            sanitized[field] = ""
    if sanitized.get("email") and not _is_valid_email(sanitized["email"]):
        logger.warning("Invalid contact email stripped: %s (contact: %s)", sanitized["email"], sanitized.get("name", "?"))
        sanitized["email"] = ""
    return sanitized


async def save_to_crm(state: CRMLeadState) -> dict:
    """Save approved leads to CRM."""
    if state.get("error"):
        return {}

    trace = state.get("_trace")
    leads = state.get("leads_data", [])
    contacts = state.get("contacts_data", [])
    approved = state.get("approved_indices", [])
    icp_id = state.get("icp_id", "")
    original_count = state.get("count", 1)

    # Accumulate from previous rounds
    previously_created = state.get("created_leads", [])
    existing_leads = state.get("existing_leads", [])

    span = trace.span(name="save_to_crm", input={"approved_count": len(approved)}) if trace else None

    created_leads: list[dict] = list(previously_created)
    new_names: list[str] = []

    for idx in approved:
        lead_data = _sanitize_lead_data(leads[idx])
        lead_contacts = contacts[idx] if idx < len(contacts) else []

        try:
            lead_result = await crm.create_lead(lead_data)
        except Exception:
            logger.exception("Failed to create lead %s", lead_data.get("businessName"))
            continue

        lead_id = lead_result.get("id", "")
        new_names.append(lead_data.get("businessName", ""))

        # Link lead to ICP
        if lead_id and icp_id:
            try:
                await crm.link_lead_to_icp(lead_id, icp_id)
            except Exception:
                logger.warning("Failed to link lead %s to ICP %s", lead_id, icp_id)

        saved_contacts = []
        for contact in lead_contacts:
            try:
                sanitized_contact = _sanitize_contact_data(contact)
                contact_result = await crm.create_lead_contact(lead_id, sanitized_contact)
                saved_contacts.append(contact_result)
            except Exception:
                logger.exception("Failed to create contact for lead %s", lead_id)

        created_leads.append({
            "lead": lead_result,
            "contacts": saved_contacts,
            "supervisor_notes": "",
        })

    if span:
        span.end(output={"created_this_round": len(new_names), "total_created": len(created_leads)})

    # Calculate remaining and update existing_leads for retry
    remaining = original_count - len(created_leads)
    # Add rejected names to existing_leads so they're not searched again
    rejected_names = [r.get("query_term", "") for r in state.get("rejected", []) if r.get("query_term")]
    updated_existing = list(set(existing_leads + new_names + rejected_names))

    logger.info(
        "Save complete: created=%d/%d, remaining=%d",
        len(created_leads), original_count, max(remaining, 0),
    )

    return {
        "created_leads": created_leads,
        "remaining_count": max(remaining, 0),
        "existing_leads": updated_existing,
    }


async def enrich_contacts(state: CRMLeadState) -> dict:
    """Enrich leads that have incomplete contact data via targeted web searches."""
    if state.get("error"):
        return {}

    trace = state.get("_trace")
    leads = state.get("leads_data", [])
    contacts = state.get("contacts_data", [])
    enrichment_indices = state.get("enrichment_indices", [])

    if not enrichment_indices:
        return {}

    span = trace.span(
        name="enrich_contacts",
        input={"leads_to_enrich": len(enrichment_indices)},
    ) if trace else None

    newly_approved: list[int] = []
    newly_rejected: list[dict] = []

    for idx in enrichment_indices:
        lead = leads[idx]
        lead_contacts = contacts[idx] if idx < len(contacts) else []
        company_name = lead.get("businessName", "")
        website = lead.get("website", "")

        # ── Phase 1: Discover who the decision-makers are ──────
        phase1_queries = [
            f"{company_name} fundador CEO diretor email contato",
            f"{company_name} equipe diretoria liderança",
        ]

        try:
            tavily_data = await _tavily_search(phase1_queries, max_results=5)
        except Exception:
            logger.warning("Contact enrichment search failed for %s", company_name)
            newly_rejected.append({
                "query_term": company_name,
                "reason": "Contact enrichment search failed",
            })
            continue

        # Fetch company website pages for contact info
        if website:
            base = website.rstrip("/")
            for page_path in ["/sobre", "/contato", "/equipe", "/about", "/contact", "/team"]:
                try:
                    async with httpx.AsyncClient(timeout=10) as client:
                        resp = await client.get(f"{base}{page_path}", follow_redirects=True)
                        if resp.status_code == 200 and len(resp.text) < 50000:
                            tavily_data.append({
                                "query": f"website {page_path}",
                                "answer": "",
                                "results": [{"title": f"{company_name} {page_path}", "url": f"{base}{page_path}", "content": resp.text[:3000]}],
                            })
                except Exception:
                    pass

        search_context = _format_tavily_results(tavily_data)

        if not search_context.strip():
            logger.info("No enrichment data found for %s", company_name)
            newly_rejected.append({
                "query_term": company_name,
                "reason": "No contact data found after enrichment search",
            })
            continue

        # ── Phase 2: Find LinkedIn for each person by name ────
        # Extract names from phase 1 results + existing contacts
        # Search "{name}" {company} LinkedIn for each
        phase2_names = [c.get("name", "") for c in lead_contacts if c.get("name")]
        # Also extract names mentioned in phase 1 search context (Haiku will do this)
        # For now, do targeted LinkedIn searches for known contacts
        for contact_name in phase2_names:
            if len(contact_name.split()) >= 2:
                try:
                    linkedin_results = await _tavily_search(
                        [f'"{contact_name}" {company_name} LinkedIn'], max_results=3,
                    )
                    tavily_data.extend(linkedin_results)
                except Exception:
                    pass

        # Rebuild context with LinkedIn results included
        search_context = _format_tavily_results(tavily_data)

        # 3. Call Haiku to extract contacts from phase 1 + known LinkedIn results
        existing_contacts_str = json.dumps(lead_contacts, ensure_ascii=False, indent=2)
        system = CONTACT_ENRICHER_SYSTEM
        user = CONTACT_ENRICHER_USER.format(
            company_name=company_name,
            company_website=website,
            existing_contacts=existing_contacts_str,
            search_results=search_context,
        )

        generation = trace.generation(
            name=f"enrich_contacts_{idx}",
            model=settings.anthropic_supervisor_model,
            input=[{"role": "system", "content": system}, {"role": "user", "content": user}],
        ) if trace else None

        try:
            response = await _call_claude(system, user, model=settings.anthropic_supervisor_model, max_tokens=2048)
            usage = _extract_usage(response)
            raw = "".join(b.text for b in response.content if b.type == "text").strip()
            if raw.startswith("```"):
                raw = raw.split("\n", 1)[1] if "\n" in raw else raw[3:]
            if raw.endswith("```"):
                raw = raw[:-3].rstrip()
            enriched = _parse_json_lenient(raw)
        except Exception:
            logger.exception("Contact enrichment LLM failed for %s", company_name)
            if generation:
                generation.end(output={"error": "enrichment_failed"})
            newly_rejected.append({
                "query_term": company_name,
                "reason": "Contact enrichment failed",
            })
            continue

        if generation:
            generation.end(output=enriched, usage=usage)

        # ── Phase 3: Search LinkedIn for contacts still missing it ──
        extracted_contacts = enriched.get("contacts", [])
        missing_linkedin = [
            c for c in extracted_contacts
            if c.get("name") and len(c["name"].split()) >= 2 and not c.get("linkedin")
        ]
        if missing_linkedin:
            linkedin_queries = [
                f'"{c["name"]}" {company_name} LinkedIn'
                for c in missing_linkedin
            ]
            try:
                linkedin_data = await _tavily_search(linkedin_queries, max_results=3)
                linkedin_context = _format_tavily_results(linkedin_data)
                if linkedin_context.strip():
                    # Quick Haiku call to match LinkedIn URLs to contacts
                    linkedin_system = (
                        "Match LinkedIn profile URLs to the people listed below. "
                        "Return ONLY valid JSON: {\"matches\": [{\"name\": \"...\", \"linkedin\": \"https://linkedin.com/in/...\"}]}\n"
                        "RULES:\n"
                        "- Only match if the LinkedIn profile clearly belongs to that person at that company.\n"
                        "- If unsure, omit the match. Do NOT guess."
                    )
                    contacts_names = json.dumps(
                        [{"name": c["name"], "role": c.get("role", "")} for c in missing_linkedin],
                        ensure_ascii=False,
                    )
                    linkedin_user = f"## People to find\n{contacts_names}\n\n## Company\n{company_name}\n\n## Search results\n{linkedin_context}"

                    li_gen = trace.generation(
                        name=f"enrich_linkedin_{idx}",
                        model=settings.anthropic_supervisor_model,
                        input=[{"role": "system", "content": linkedin_system}, {"role": "user", "content": linkedin_user}],
                    ) if trace else None

                    li_resp = await _call_claude(linkedin_system, linkedin_user, model=settings.anthropic_supervisor_model, max_tokens=512)
                    li_usage = _extract_usage(li_resp)
                    li_raw = "".join(b.text for b in li_resp.content if b.type == "text").strip()
                    if li_raw.startswith("```"):
                        li_raw = li_raw.split("\n", 1)[1] if "\n" in li_raw else li_raw[3:]
                    if li_raw.endswith("```"):
                        li_raw = li_raw[:-3].rstrip()
                    li_parsed = _parse_json_lenient(li_raw)
                    if li_gen:
                        li_gen.end(output=li_parsed, usage=li_usage)

                    # Apply LinkedIn matches to extracted contacts
                    matches = {m["name"].lower(): m["linkedin"] for m in li_parsed.get("matches", [])}
                    for c in extracted_contacts:
                        if not c.get("linkedin") and c.get("name", "").lower() in matches:
                            c["linkedin"] = matches[c["name"].lower()]
                            logger.info("LinkedIn found for %s: %s", c["name"], c["linkedin"])
            except Exception:
                logger.warning("LinkedIn enrichment search failed for %s", company_name)

        enriched["contacts"] = extracted_contacts

        # 5. Update contacts and lead with enriched data
        new_contacts = enriched.get("contacts", lead_contacts)
        if enriched.get("company_email") and not lead.get("email"):
            leads[idx]["email"] = enriched["company_email"]
        if enriched.get("company_phone") and not lead.get("phone"):
            leads[idx]["phone"] = enriched["company_phone"]
        contacts[idx] = new_contacts

        # Filter out generic/team contacts (not real people)
        new_contacts = [c for c in new_contacts if _is_decision_maker(c)]
        contacts[idx] = new_contacts

        # Check if enrichment succeeded (at least 1 real person with name + email or linkedin)
        has_good_contact = any(
            c.get("name") and (c.get("email") or c.get("linkedin"))
            for c in new_contacts
        )
        if has_good_contact:
            newly_approved.append(idx)
            logger.info("Enrichment succeeded for %s: %d contacts", company_name, len(new_contacts))
        else:
            newly_rejected.append({
                "query_term": company_name,
                "reason": "Contact data still incomplete after enrichment",
            })

    # Merge enriched approvals with existing approvals
    current_approved = state.get("approved_indices", [])
    current_rejected = state.get("rejected", [])

    if span:
        span.end(output={"enriched": len(newly_approved), "failed": len(newly_rejected)})

    return {
        "approved_indices": current_approved + newly_approved,
        "enrichment_indices": [],  # Clear — enrichment is done
        "rejected": current_rejected + newly_rejected,
        "leads_data": leads,
        "contacts_data": contacts,
    }


async def format_reply(state: CRMLeadState) -> dict:
    """Build the final reply message."""
    created = state.get("created_leads", [])
    rejected = state.get("rejected", [])
    error = state.get("error")

    # If error but we have results from previous rounds, treat as partial success
    if error and not created:
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

    # Clear error if we have results (partial success)
    return {"reply": ". ".join(parts), "error": None}


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
    query_hint = f" for the query: **{query}**" if query else ""
    system = INVESTIGATOR_SYSTEM
    user = INVESTIGATOR_USER_TAVILY.format(
        icp_context=icp_text, query_hint=query_hint, count=count,
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
