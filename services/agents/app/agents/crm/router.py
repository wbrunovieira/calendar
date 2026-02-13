from __future__ import annotations

import asyncio
import logging
import uuid

from pydantic import BaseModel, Field
from fastapi import APIRouter, BackgroundTasks

from app.agents.crm.graph import build_crm_graph
# from app.agents.crm.nodes import run_shadow_investigation
from app.clients import crm
from app.config import settings
from app.agents.finances.nodes import langfuse

logger = logging.getLogger("agents")

router = APIRouter(prefix="/agents", tags=["agents"])

_graph = None


def _get_graph():
    global _graph
    if _graph is None:
        _graph = build_crm_graph()
    return _graph


# ── Request models ──────────────────────────────────────────

class ICPPayload(BaseModel):
    id: str
    name: str = ""
    content: str = ""


class SearchParams(BaseModel):
    searchTerm: str
    country: str = "Brasil"
    quantity: int = Field(default=1, ge=1, le=10)
    quality: str = "cold"


class LeadResearchRequest(BaseModel):
    # CRM format (primary)
    icp: ICPPayload | None = None
    searchParams: SearchParams | None = None

    # Direct format (alternative)
    query: str | None = None
    icp_id: str | None = None
    count: int | None = Field(default=None, ge=1, le=10)
    country: str = "Brasil"

    # Max ICP content chars to send to the agent (ICP definition + checklist only, no sales copy)
    _ICP_CONTENT_MAX_CHARS = 2500

    def resolve(self) -> tuple[str, str, int, str, dict | None]:
        """Returns (query, icp_id, count, country, icp_context_or_none)."""
        if self.icp and self.searchParams:
            # CRM format: ICP object + searchParams — truncate content to save tokens
            content = self.icp.content[:self._ICP_CONTENT_MAX_CHARS] if self.icp.content else ""
            icp_context = {"id": self.icp.id, "name": self.icp.name, "content": content}
            return (
                self.searchParams.searchTerm,
                self.icp.id,
                self.searchParams.quantity,
                self.searchParams.country,
                icp_context,
            )
        # Direct format
        return (
            self.query or "",
            self.icp_id or "",
            self.count or 1,
            self.country,
            None,
        )


# ── Response models ─────────────────────────────────────────

class LeadResult(BaseModel):
    lead: dict
    contacts: list[dict]
    supervisor_notes: str


class RejectedLead(BaseModel):
    query_term: str
    reason: str


class LeadResearchResponse(BaseModel):
    status: str
    jobId: str | None = None
    reply: str
    leads: list[LeadResult] = []
    rejected: list[RejectedLead] = []
    error: str | None = None


# ── Background worker ──────────────────────────────────────

async def _run_research(
    job_id: str, query: str, icp_id: str, count: int,
    country: str, icp_context: dict | None, trace_id: str | None,
):
    """Run lead research in background and notify CRM via webhook."""
    trace = None
    if trace_id and langfuse.enabled:
        trace = langfuse.trace(
            name="crm-lead-research",
            id=trace_id,
            input={"query": query, "icp_id": icp_id, "count": count, "country": country},
        )

    graph = _get_graph()
    state_input: dict = {
        "query": query,
        "icp_id": icp_id,
        "count": count,
        "country": country,
        "_trace": trace,
    }
    if icp_context:
        state_input["icp_context"] = icp_context

    try:
        result = await graph.ainvoke(state_input)
    except Exception:
        logger.exception("Background lead research failed")
        if trace:
            trace.update(output={"error": "background_task_failed"})
            langfuse.flush()
        await crm.send_webhook({
            "jobId": job_id,
            "status": "error",
            "createdLeads": [],
            "rejected": [],
            "summary": "Background task failed unexpectedly",
            "error": "background_task_failed",
        })
        return

    # Shadow A/B experiment — uncomment to test another model
    # if settings.deepseek_api_key:
    #     icp_ctx = icp_context or result.get("icp_context") or {}
    #     existing = result.get("existing_leads", [])
    #     tavily_data = result.get("tavily_data")
    #     asyncio.create_task(
    #         run_shadow_investigation(trace, icp_ctx, query, count, existing, tavily_data)
    #     )

    error = result.get("error")
    reply = result.get("reply", "Unknown error")
    status = "error" if error else "completed"

    if trace:
        trace.update(output={"reply": reply, "status": status, "error": error})
        langfuse.flush()

    created = result.get("created_leads", [])
    rejected = result.get("rejected", [])
    logger.info(
        "Lead research completed: query=%s, created=%d, rejected=%d, error=%s",
        query, len(created), len(rejected), error,
    )

    # Notify CRM via webhook
    await crm.send_webhook({
        "jobId": job_id,
        "status": status,
        "createdLeads": created,
        "rejected": [
            {"queryTerm": r.get("query_term", "?"), "reason": r.get("reason", "")}
            for r in rejected
        ],
        "summary": reply,
        "error": error,
    })


# ── Endpoint ───────────────────────────────────────────────

@router.post("/crm/lead-research", response_model=LeadResearchResponse)
async def handle_lead_research(req: LeadResearchRequest, background_tasks: BackgroundTasks):
    query, icp_id, count, country, icp_context = req.resolve()

    if not query:
        return LeadResearchResponse(
            status="error",
            reply="Missing search query",
            error="missing_query",
        )

    if not icp_id:
        return LeadResearchResponse(
            status="error",
            reply="Missing ICP ID",
            error="missing_icp_id",
        )

    job_id = str(uuid.uuid4())

    # Create trace ID upfront so we can track it
    trace_id = None
    if langfuse.enabled:
        trace = langfuse.trace(
            name="crm-lead-research",
            input={"query": query, "icp_id": icp_id, "count": count, "country": country},
            metadata={"async": True, "job_id": job_id},
        )
        trace_id = trace.id
        langfuse.flush()

    # Schedule research in background
    background_tasks.add_task(_run_research, job_id, query, icp_id, count, country, icp_context, trace_id)

    return LeadResearchResponse(
        status="accepted",
        jobId=job_id,
        reply=f"Research started: '{query}' ({count} lead(s)). Results will be saved to CRM.",
    )
