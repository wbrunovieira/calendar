from __future__ import annotations

import logging

from fastapi import APIRouter, BackgroundTasks
from pydantic import BaseModel

from app.agents.crm.call_analysis.graph import build_call_analysis_graph
from app.agents.finances.nodes import langfuse

logger = logging.getLogger("agents")

router = APIRouter(prefix="/agents", tags=["agents"])

_graph = None


def _get_graph():
    global _graph
    if _graph is None:
        _graph = build_call_analysis_graph()
    return _graph


# ── Request models ────────────────────────────────────────────────────────


class CallLeadInput(BaseModel):
    id: str
    businessName: str
    description: str | None = None
    segment: str | None = None
    city: str | None = None
    activities: str | None = None


class CallContactInput(BaseModel):
    name: str
    role: str | None = None


class CallActivityInput(BaseModel):
    id: str
    subject: str | None = None
    notes: str | None = None


class CallAnalysisRequest(BaseModel):
    jobId: str
    webhookUrl: str
    transcript: str
    callDurationSeconds: int = 0
    callDate: str | None = None
    lead: CallLeadInput
    contact: CallContactInput
    activity: CallActivityInput


# ── Response model ────────────────────────────────────────────────────────


class CallAnalysisResponse(BaseModel):
    status: str
    jobId: str


# ── Background worker ──────────────────────────────────────────────────────


async def _run_call_analysis(
    job_id: str,
    webhook_url: str,
    transcript: str,
    call_duration_seconds: int,
    call_date: str | None,
    lead: dict,
    contact: dict,
    activity: dict,
    trace_id: str | None,
) -> None:
    trace = None
    if trace_id and langfuse.enabled:
        trace = langfuse.trace(
            name="crm-call-analysis",
            id=trace_id,
            input={"jobId": job_id, "businessName": lead.get("businessName"), "contact": contact.get("name")},
            metadata={"callDurationSeconds": call_duration_seconds},
        )

    graph = _get_graph()
    state_input: dict = {
        "job_id": job_id,
        "webhook_url": webhook_url,
        "transcript": transcript,
        "call_duration_seconds": call_duration_seconds,
        "call_date": call_date or "",
        "lead": lead,
        "contact": contact,
        "activity": activity,
        "_trace": trace,
    }

    try:
        result = await graph.ainvoke(state_input, config={"recursion_limit": 20})
    except Exception:
        logger.exception("[CallAnalysis] background task failed job=%s", job_id)
        if trace:
            trace.update(output={"error": "background_task_failed"})
            langfuse.flush()
        # webhook already sent inside send_webhook node on happy path;
        # on graph-level crash we try once more
        import httpx
        try:
            async with httpx.AsyncClient(timeout=10) as client:
                await client.post(webhook_url, json={"jobId": job_id, "status": "error", "error": "background_task_failed"})
        except Exception:
            logger.exception("[CallAnalysis] fallback webhook also failed job=%s", job_id)
        return

    if trace:
        trace.update(output={
            "score": result.get("score"),
            "noShowRisk": result.get("no_show_risk"),
            "error": result.get("error"),
        })
        langfuse.flush()

    logger.info(
        "[CallAnalysis] completed job=%s score=%s risk=%s error=%s",
        job_id,
        result.get("score"),
        result.get("no_show_risk"),
        result.get("error"),
    )


# ── Endpoint ───────────────────────────────────────────────────────────────


@router.post("/crm/call-analysis", response_model=CallAnalysisResponse)
async def handle_call_analysis(
    req: CallAnalysisRequest,
    background_tasks: BackgroundTasks,
):
    trace_id = None
    if langfuse.enabled:
        trace = langfuse.trace(
            name="crm-call-analysis",
            input={"jobId": req.jobId, "businessName": req.lead.businessName},
            metadata={"async": True, "callDurationSeconds": req.callDurationSeconds},
        )
        trace_id = trace.id
        langfuse.flush()

    background_tasks.add_task(
        _run_call_analysis,
        req.jobId,
        req.webhookUrl,
        req.transcript,
        req.callDurationSeconds,
        req.callDate,
        req.lead.model_dump(),
        req.contact.model_dump(),
        req.activity.model_dump(),
        trace_id,
    )

    logger.info(
        "[CallAnalysis] accepted job=%s lead=%s contact=%s duration=%ds",
        req.jobId, req.lead.businessName, req.contact.name, req.callDurationSeconds,
    )

    return CallAnalysisResponse(status="accepted", jobId=req.jobId)
