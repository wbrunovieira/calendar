from __future__ import annotations

import logging

from fastapi import APIRouter, BackgroundTasks
from pydantic import BaseModel

from app.agents.crm.gatekeeper_batch.graph import build_gatekeeper_batch_graph
from app.agents.finances.nodes import langfuse

logger = logging.getLogger("agents")

router = APIRouter(prefix="/agents", tags=["agents"])

_graph = None


def _get_graph():
    global _graph
    if _graph is None:
        _graph = build_gatekeeper_batch_graph()
    return _graph


# ── Request models ────────────────────────────────────────────────────────


class GatekeeperBatchRequest(BaseModel):
    batchJobId: str
    webhookUrl: str
    analyses: list           # list of individual GK analysis results
    historicalSummaries: list = []


# ── Response model ────────────────────────────────────────────────────────


class GatekeeperBatchResponse(BaseModel):
    status: str
    batchJobId: str


# ── Background worker ─────────────────────────────────────────────────────


async def _run_gatekeeper_batch(
    batch_job_id: str,
    webhook_url: str,
    analyses: list,
    historical_summaries: list,
    trace_id: str | None,
) -> None:
    trace = None
    if trace_id and langfuse.enabled:
        trace = langfuse.trace(
            name="crm-gatekeeper-batch",
            id=trace_id,
            input={"batchJobId": batch_job_id, "nAnalyses": len(analyses)},
        )

    graph = _get_graph()
    state_input: dict = {
        "batch_job_id": batch_job_id,
        "webhook_url": webhook_url,
        "analyses": analyses,
        "historical_summaries": historical_summaries,
        "_trace": trace,
    }

    try:
        result = await graph.ainvoke(state_input, config={"recursion_limit": 20})
    except Exception:
        logger.exception("[GKBatch] background task failed batch=%s", batch_job_id)
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
                    json={"batchJobId": batch_job_id, "status": "error", "error": "background_task_failed"},
                    headers=headers,
                )
        except Exception:
            logger.exception("[GKBatch] fallback webhook also failed batch=%s", batch_job_id)
        return

    if trace:
        trace.update(output={
            "overallScore": result.get("overall_score"),
            "error": result.get("error"),
        })
        langfuse.flush()

    logger.info(
        "[GKBatch] completed batch=%s overall_score=%s error=%s",
        batch_job_id, result.get("overall_score"), result.get("error"),
    )


# ── Endpoint ──────────────────────────────────────────────────────────────


@router.post("/crm/gatekeeper-batch", response_model=GatekeeperBatchResponse)
async def handle_gatekeeper_batch(
    req: GatekeeperBatchRequest,
    background_tasks: BackgroundTasks,
):
    trace_id = None
    if langfuse.enabled:
        trace = langfuse.trace(
            name="crm-gatekeeper-batch",
            input={"batchJobId": req.batchJobId, "nAnalyses": len(req.analyses)},
            metadata={"async": True},
        )
        trace_id = trace.id
        langfuse.flush()

    background_tasks.add_task(
        _run_gatekeeper_batch,
        req.batchJobId,
        req.webhookUrl,
        req.analyses,
        req.historicalSummaries,
        trace_id,
    )

    logger.info(
        "[GKBatch] accepted batch=%s n_analyses=%d",
        req.batchJobId, len(req.analyses),
    )

    return GatekeeperBatchResponse(status="accepted", batchJobId=req.batchJobId)
