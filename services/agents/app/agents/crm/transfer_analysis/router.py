from __future__ import annotations

import logging

from fastapi import APIRouter, BackgroundTasks
from pydantic import BaseModel

from app.agents.crm.transfer_analysis.graph import build_transfer_analysis_graph
from app.agents.finances.nodes import langfuse

logger = logging.getLogger("agents")

router = APIRouter(prefix="/agents", tags=["agents"])

_graph = None


def _get_graph():
    global _graph
    if _graph is None:
        _graph = build_transfer_analysis_graph()
    return _graph


# ── Request models ────────────────────────────────────────────────────────


class TALeadInput(BaseModel):
    id: str
    businessName: str
    segment: str | None = None
    city: str | None = None
    description: str | None = None


class TAContactInput(BaseModel):
    name: str | None = None
    role: str | None = None


class TAActivityInput(BaseModel):
    id: str
    subject: str | None = None
    notes: str | None = None


class TransferAnalysisRequest(BaseModel):
    gkJobId: str
    spicedJobId: str
    gkWebhookUrl: str
    spicedWebhookUrl: str
    transcript: str
    callDurationSeconds: int = 0
    callDate: str | None = None
    lead: TALeadInput
    contact: TAContactInput
    activity: TAActivityInput | None = None


# ── Response model ────────────────────────────────────────────────────────


class TransferAnalysisResponse(BaseModel):
    status: str
    gkJobId: str
    spicedJobId: str


# ── Background worker ─────────────────────────────────────────────────────


async def _run_transfer_analysis(
    gk_job_id: str,
    spiced_job_id: str,
    gk_webhook_url: str,
    spiced_webhook_url: str,
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
            name="crm-transfer-analysis",
            id=trace_id,
            input={
                "gkJobId": gk_job_id,
                "spicedJobId": spiced_job_id,
                "businessName": lead.get("businessName"),
                "contact": contact.get("name"),
            },
            metadata={"callDurationSeconds": call_duration_seconds},
        )

    graph = _get_graph()
    state_input: dict = {
        "gk_job_id": gk_job_id,
        "spiced_job_id": spiced_job_id,
        "gk_webhook_url": gk_webhook_url,
        "spiced_webhook_url": spiced_webhook_url,
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
        logger.exception(
            "[TransferAnalysis] background task failed gk=%s spiced=%s",
            gk_job_id, spiced_job_id,
        )
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
                err = {"status": "error", "error": "background_task_failed"}
                await client.post(gk_webhook_url, json={"jobId": gk_job_id, **err}, headers=headers)
                await client.post(spiced_webhook_url, json={"jobId": spiced_job_id, **err}, headers=headers)
        except Exception:
            logger.exception("[TransferAnalysis] fallback webhooks failed gk=%s", gk_job_id)
        return

    if trace:
        trace.update(output={"error": result.get("error")})
        langfuse.flush()

    logger.info(
        "[TransferAnalysis] completed gk=%s spiced=%s error=%s",
        gk_job_id, spiced_job_id, result.get("error"),
    )


# ── Endpoint ──────────────────────────────────────────────────────────────


@router.post("/crm/transfer-analysis", response_model=TransferAnalysisResponse)
async def handle_transfer_analysis(
    req: TransferAnalysisRequest,
    background_tasks: BackgroundTasks,
):
    trace_id = None
    if langfuse.enabled:
        trace = langfuse.trace(
            name="crm-transfer-analysis",
            input={
                "gkJobId": req.gkJobId,
                "spicedJobId": req.spicedJobId,
                "businessName": req.lead.businessName,
            },
            metadata={"async": True, "callDurationSeconds": req.callDurationSeconds},
        )
        trace_id = trace.id
        langfuse.flush()

    background_tasks.add_task(
        _run_transfer_analysis,
        req.gkJobId,
        req.spicedJobId,
        req.gkWebhookUrl,
        req.spicedWebhookUrl,
        req.transcript,
        req.callDurationSeconds,
        req.callDate,
        req.lead.model_dump(),
        req.contact.model_dump(),
        req.activity.model_dump() if req.activity else {},
        trace_id,
    )

    logger.info(
        "[TransferAnalysis] accepted gk=%s spiced=%s lead=%s duration=%ds",
        req.gkJobId, req.spicedJobId, req.lead.businessName, req.callDurationSeconds,
    )

    return TransferAnalysisResponse(
        status="accepted",
        gkJobId=req.gkJobId,
        spicedJobId=req.spicedJobId,
    )
