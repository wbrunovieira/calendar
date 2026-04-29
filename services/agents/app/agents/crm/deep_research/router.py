from __future__ import annotations

import logging
import uuid
from typing import Any

from fastapi import APIRouter, BackgroundTasks
from pydantic import BaseModel

from app.agents.crm.deep_research.graph import build_deep_research_graph
from app.clients import crm
from app.agents.finances.nodes import langfuse

logger = logging.getLogger("agents")

router = APIRouter(prefix="/agents", tags=["agents"])

_graph = None


def _get_graph():
    global _graph
    if _graph is None:
        _graph = build_deep_research_graph()
    return _graph


# ── Request models ────────────────────────────────────────────────────────


class ContactInput(BaseModel):
    name: str
    email: str | None = None
    phone: str | None = None
    role: str | None = None
    linkedin: str | None = None


class LeadInput(BaseModel):
    businessName: str
    companyRegistrationID: str | None = None
    foundationDate: str | None = None
    companyOwner: str | None = None
    legalNature: str | None = None
    segment: str | None = None
    branchType: str | None = None
    simplesNacional: bool | None = None
    isMei: bool | None = None
    companySize: str | None = None
    employeesCount: int | None = None
    revenue: float | None = None
    revenueRange: str | None = None
    description: str | None = None
    address: str | None = None
    city: str | None = None
    state: str | None = None
    country: str | None = None
    zipCode: str | None = None
    phone: str | None = None
    phone2: str | None = None
    whatsapp: str | None = None
    email: str | None = None
    website: str | None = None
    instagram: str | None = None
    linkedin: str | None = None
    facebook: str | None = None
    twitter: str | None = None
    tiktok: str | None = None
    categories: str | None = None
    internationalActivity: str | None = None
    source: str | None = None
    quality: str | None = None
    metaAds: dict | None = None


class DeepResearchRequest(BaseModel):
    leadId: str
    requesterId: str
    lead: LeadInput
    contacts: list[ContactInput] = []
    previousSummary: str | None = None
    previousResearchAt: str | None = None
    focusField: str | None = None
    customInstruction: str | None = None


# ── Response model ────────────────────────────────────────────────────────


class DeepResearchResponse(BaseModel):
    status: str
    jobId: str | None = None
    reply: str
    error: str | None = None


# ── Background worker ──────────────────────────────────────────────────────


async def _run_deep_research(
    job_id: str,
    lead_id: str,
    requester_id: str,
    lead: dict,
    contacts: list[dict],
    trace_id: str | None,
    previous_summary: str | None = None,
    previous_research_at: str | None = None,
    focus_field: str | None = None,
    custom_instruction: str | None = None,
) -> None:
    mode = "focused" if focus_field else ("reresearch" if previous_summary else "full")
    trace: Any = None
    if trace_id and langfuse.enabled:
        trace = langfuse.trace(
            name="crm-lead-deep-research",
            id=trace_id,
            input={"leadId": lead_id, "businessName": lead.get("businessName")},
            metadata={"mode": mode, "focusField": focus_field},
        )

    graph = _get_graph()
    state_input: dict = {
        "lead_id": lead_id,
        "requester_id": requester_id,
        "lead": lead,
        "contacts": contacts,
        "previous_summary": previous_summary,
        "previous_research_at": previous_research_at,
        "focus_field": focus_field,
        "custom_instruction": custom_instruction,
        "_trace": trace,
    }

    try:
        result = await graph.ainvoke(state_input, config={"recursion_limit": 40})
    except Exception:
        logger.exception("Background deep research failed for lead %s", lead_id)
        if trace:
            trace.update(output={"error": "background_task_failed"})
            langfuse.flush()
        await crm.send_deep_research_webhook({
            "jobId": job_id,
            "leadId": lead_id,
            "status": "error",
            "updates": {},
            "forcedUpdates": {},
            "proposedFields": {},
            "newContacts": [],
            "summary": "Falha inesperada na execução da pesquisa.",
            "error": "background_task_failed",
        })
        return

    error = result.get("error")
    updates = result.get("updates") or {}
    forced_updates = result.get("forced_updates") or {}
    proposed_fields = result.get("proposed_fields") or {}
    new_contacts = result.get("new_contacts") or []
    summary = result.get("summary") or ""
    status = "error" if error else "completed"

    if trace:
        trace.update(output={
            "status": status,
            "mode": mode,
            "updatedFields": list(updates.keys()),
            "forcedFields": list(forced_updates.keys()),
            "proposedFields": list(proposed_fields.keys()),
            "newContacts": len(new_contacts),
        })
        langfuse.flush()

    logger.info(
        "[DeepResearch] lead=%s mode=%s status=%s updates=%d forced=%d contacts=%d",
        lead_id, mode, status, len(updates), len(forced_updates), len(new_contacts),
    )

    await crm.send_deep_research_webhook({
        "jobId": job_id,
        "leadId": lead_id,
        "status": status,
        "updates": updates,
        "forcedUpdates": forced_updates,
        "proposedFields": proposed_fields,
        "newContacts": new_contacts,
        "summary": summary,
        "error": error,
    })


# ── Endpoint ───────────────────────────────────────────────────────────────


@router.post("/crm/lead-deep-research", response_model=DeepResearchResponse)
async def handle_lead_deep_research(
    req: DeepResearchRequest,
    background_tasks: BackgroundTasks,
):
    job_id = str(uuid.uuid4())

    trace_id = None
    if langfuse.enabled:
        trace = langfuse.trace(
            name="crm-lead-deep-research",
            input={"leadId": req.leadId, "businessName": req.lead.businessName},
            metadata={"async": True, "job_id": job_id},
        )
        trace_id = trace.id
        langfuse.flush()

    background_tasks.add_task(
        _run_deep_research,
        job_id,
        req.leadId,
        req.requesterId,
        req.lead.model_dump(),
        [c.model_dump() for c in req.contacts],
        trace_id,
        req.previousSummary,
        req.previousResearchAt,
        req.focusField,
        req.customInstruction,
    )

    if req.focusField:
        mode = f"Pesquisa focada em \"{req.focusField}\""
    elif req.previousSummary:
        mode = "Re-pesquisa"
    else:
        mode = "Pesquisa aprofundada"

    return DeepResearchResponse(
        status="accepted",
        jobId=job_id,
        reply=f"{mode} iniciada para \"{req.lead.businessName}\". O CRM será notificado ao concluir.",
    )
