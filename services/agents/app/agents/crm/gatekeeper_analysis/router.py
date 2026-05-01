from __future__ import annotations

import logging

from fastapi import APIRouter, BackgroundTasks
from pydantic import BaseModel

from app.agents.crm.gatekeeper_analysis.graph import build_gatekeeper_analysis_graph
from app.agents.finances.nodes import langfuse

logger = logging.getLogger("agents")

router = APIRouter(prefix="/agents", tags=["agents"])

_graph = None


def _get_graph():
    global _graph
    if _graph is None:
        _graph = build_gatekeeper_analysis_graph()
    return _graph


# ── Request models ────────────────────────────────────────────────────────


class GKLeadInput(BaseModel):
    id: str
    businessName: str
    segment: str | None = None
    city: str | None = None


class GKContactInput(BaseModel):
    name: str | None = None
    role: str | None = None


class GKActivityInput(BaseModel):
    id: str
    subject: str | None = None
    notes: str | None = None


class GatekeeperAnalysisRequest(BaseModel):
    jobId: str
    webhookUrl: str
    transcript: str
    callDurationSeconds: int = 0
    callDate: str | None = None
    lead: GKLeadInput
    contact: GKContactInput
    activity: GKActivityInput | None = None


# ── Response model ────────────────────────────────────────────────────────


class GatekeeperAnalysisResponse(BaseModel):
    status: str
    jobId: str


# ── Background worker ─────────────────────────────────────────────────────


async def _run_gatekeeper_analysis(
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
            name="crm-gatekeeper-analysis",
            id=trace_id,
            input={
                "jobId": job_id,
                "businessName": lead.get("businessName"),
                "contact": contact.get("name"),
            },
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
        logger.exception("[GKAnalysis] background task failed job=%s", job_id)
        if trace:
            trace.update(output={"error": "background_task_failed"})
            langfuse.flush()
        import httpx
        try:
            from app.config import settings
            headers = {"Content-Type": "application/json"}
            if settings.crm_webhook_secret:
                headers["X-Webhook-Secret"] = settings.crm_webhook_secret
            async with httpx.AsyncClient(timeout=10) as client:
                await client.post(
                    webhook_url,
                    json={"jobId": job_id, "status": "error", "error": "background_task_failed"},
                    headers=headers,
                )
        except Exception:
            logger.exception("[GKAnalysis] fallback webhook also failed job=%s", job_id)
        return

    if trace:
        trace.update(output={
            "score": result.get("score"),
            "error": result.get("error"),
        })
        langfuse.flush()

    logger.info(
        "[GKAnalysis] completed job=%s score=%s error=%s",
        job_id, result.get("score"), result.get("error"),
    )


# ── Endpoint ──────────────────────────────────────────────────────────────


@router.post("/crm/gatekeeper-analysis", response_model=GatekeeperAnalysisResponse)
async def handle_gatekeeper_analysis(
    req: GatekeeperAnalysisRequest,
    background_tasks: BackgroundTasks,
):
    trace_id = None
    if langfuse.enabled:
        trace = langfuse.trace(
            name="crm-gatekeeper-analysis",
            input={"jobId": req.jobId, "businessName": req.lead.businessName},
            metadata={"async": True, "callDurationSeconds": req.callDurationSeconds},
        )
        trace_id = trace.id
        langfuse.flush()

    background_tasks.add_task(
        _run_gatekeeper_analysis,
        req.jobId,
        req.webhookUrl,
        req.transcript,
        req.callDurationSeconds,
        req.callDate,
        req.lead.model_dump(),
        req.contact.model_dump(),
        req.activity.model_dump() if req.activity else {},
        trace_id,
    )

    logger.info(
        "[GKAnalysis] accepted job=%s lead=%s contact=%s duration=%ds",
        req.jobId, req.lead.businessName, req.contact.name, req.callDurationSeconds,
    )

    return GatekeeperAnalysisResponse(status="accepted", jobId=req.jobId)
