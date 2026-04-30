from __future__ import annotations

import json
import logging

from fastapi import APIRouter, BackgroundTasks
from pydantic import BaseModel

from app.agents.crm.meet_analysis.graph import build_meet_analysis_graph
from app.agents.finances.nodes import langfuse

logger = logging.getLogger("agents")

router = APIRouter(prefix="/agents", tags=["agents"])

_graph = None


def _get_graph():
    global _graph
    if _graph is None:
        _graph = build_meet_analysis_graph()
    return _graph


# ── Request models ────────────────────────────────────────────────────────


class MeetLeadInput(BaseModel):
    id: str
    businessName: str
    description: str | None = None
    segment: str | None = None
    city: str | None = None


class MeetContactInput(BaseModel):
    name: str | None = None
    role: str | None = None


class MeetActivityInput(BaseModel):
    id: str
    subject: str | None = None
    notes: str | None = None


def _normalize_transcript(raw: str) -> str:
    """Convert JSON array transcript (Google Meet format) to plain text if needed."""
    stripped = raw.strip()
    if not stripped.startswith("["):
        return raw
    try:
        segments = json.loads(stripped)
        lines = []
        for seg in segments:
            start = seg.get("start", 0)
            minutes = int(start) // 60
            seconds = int(start) % 60
            speaker = seg.get("speakerName") or seg.get("speaker") or "?"
            text = (seg.get("text") or "").strip()
            if text:
                lines.append(f"{minutes:02d}:{seconds:02d} [{speaker}]: {text}")
        return "\n".join(lines)
    except (json.JSONDecodeError, TypeError):
        return raw


class MeetAnalysisRequest(BaseModel):
    jobId: str
    webhookUrl: str
    transcript: str
    meetingDurationSeconds: int = 0
    meetingDate: str | None = None
    meetingTitle: str | None = None
    lead: MeetLeadInput
    contact: MeetContactInput
    activity: MeetActivityInput


# ── Response model ────────────────────────────────────────────────────────


class MeetAnalysisResponse(BaseModel):
    status: str
    jobId: str


# ── Background worker ─────────────────────────────────────────────────────


async def _run_meet_analysis(
    job_id: str,
    webhook_url: str,
    transcript: str,
    meeting_duration_seconds: int,
    meeting_date: str | None,
    meeting_title: str | None,
    lead: dict,
    contact: dict,
    activity: dict,
    trace_id: str | None,
) -> None:
    trace = None
    if trace_id and langfuse.enabled:
        trace = langfuse.trace(
            name="crm-meet-analysis",
            id=trace_id,
            input={
                "jobId": job_id,
                "businessName": lead.get("businessName"),
                "contact": contact.get("name"),
            },
            metadata={"meetingDurationSeconds": meeting_duration_seconds},
        )

    graph = _get_graph()
    state_input: dict = {
        "job_id": job_id,
        "webhook_url": webhook_url,
        "transcript": transcript,
        "meeting_duration_seconds": meeting_duration_seconds,
        "meeting_date": meeting_date or "",
        "meeting_title": meeting_title or "",
        "lead": lead,
        "contact": contact,
        "activity": activity,
        "_trace": trace,
    }

    try:
        result = await graph.ainvoke(state_input, config={"recursion_limit": 20})
    except Exception:
        logger.exception("[MeetAnalysis] background task failed job=%s", job_id)
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
            logger.exception("[MeetAnalysis] fallback webhook also failed job=%s", job_id)
        return

    if trace:
        trace.update(output={
            "score": result.get("score"),
            "closingProbability": result.get("closing_probability"),
            "error": result.get("error"),
        })
        langfuse.flush()

    logger.info(
        "[MeetAnalysis] completed job=%s score=%s closing_prob=%s error=%s",
        job_id,
        result.get("score"),
        result.get("closing_probability"),
        result.get("error"),
    )


# ── Endpoint ──────────────────────────────────────────────────────────────


@router.post("/crm/meet-analysis", response_model=MeetAnalysisResponse)
async def handle_meet_analysis(
    req: MeetAnalysisRequest,
    background_tasks: BackgroundTasks,
):
    trace_id = None
    if langfuse.enabled:
        trace = langfuse.trace(
            name="crm-meet-analysis",
            input={"jobId": req.jobId, "businessName": req.lead.businessName},
            metadata={"async": True, "meetingDurationSeconds": req.meetingDurationSeconds},
        )
        trace_id = trace.id
        langfuse.flush()

    background_tasks.add_task(
        _run_meet_analysis,
        req.jobId,
        req.webhookUrl,
        _normalize_transcript(req.transcript),
        req.meetingDurationSeconds,
        req.meetingDate,
        req.meetingTitle,
        req.lead.model_dump(),
        req.contact.model_dump(),
        req.activity.model_dump(),
        trace_id,
    )

    logger.info(
        "[MeetAnalysis] accepted job=%s lead=%s contact=%s duration=%ds",
        req.jobId, req.lead.businessName, req.contact.name, req.meetingDurationSeconds,
    )

    return MeetAnalysisResponse(status="accepted", jobId=req.jobId)
