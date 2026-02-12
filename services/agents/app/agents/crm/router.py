from __future__ import annotations

from pydantic import BaseModel, Field
from fastapi import APIRouter

from app.agents.crm.graph import build_crm_graph
from app.agents.finances.nodes import langfuse

router = APIRouter(prefix="/agents", tags=["agents"])

_graph = None


def _get_graph():
    global _graph
    if _graph is None:
        _graph = build_crm_graph()
    return _graph


class LeadResearchRequest(BaseModel):
    query: str
    icp_id: str
    count: int = Field(default=1, ge=1, le=5)


class LeadResult(BaseModel):
    lead: dict
    contacts: list[dict]
    supervisor_notes: str


class RejectedLead(BaseModel):
    query_term: str
    reason: str


class LeadResearchResponse(BaseModel):
    status: str
    reply: str
    leads: list[LeadResult] = []
    rejected: list[RejectedLead] = []
    error: str | None = None


@router.post("/crm/lead-research", response_model=LeadResearchResponse)
async def handle_lead_research(req: LeadResearchRequest):
    trace = langfuse.trace(
        name="crm-lead-research",
        input={"query": req.query, "icp_id": req.icp_id, "count": req.count},
    ) if langfuse.enabled else None

    graph = _get_graph()
    result = await graph.ainvoke({
        "query": req.query,
        "icp_id": req.icp_id,
        "count": req.count,
        "_trace": trace,
    })

    error = result.get("error")
    created = result.get("created_leads", [])
    rejected = result.get("rejected", [])
    reply = result.get("reply", "Unknown error")

    if error:
        status = "error"
    elif rejected and created:
        status = "partial"
    else:
        status = "created"

    leads_out = [
        LeadResult(
            lead=c["lead"],
            contacts=c.get("contacts", []),
            supervisor_notes=c.get("supervisor_notes", ""),
        )
        for c in created
    ]
    rejected_out = [
        RejectedLead(query_term=r["query_term"], reason=r["reason"])
        for r in rejected
    ]

    if trace:
        trace.update(output={"reply": reply, "status": status, "error": error})
        langfuse.flush()

    return LeadResearchResponse(
        status=status,
        reply=reply,
        leads=leads_out,
        rejected=rejected_out,
        error=error,
    )
