from __future__ import annotations

import logging

import httpx
from fastapi import APIRouter, BackgroundTasks, Header, HTTPException
from pydantic import BaseModel

from app.agents.crm.proposal.graph import build_proposal_graph
from app.agents.crm.proposal.store import delete_job, load_job, save_job
from app.agents.finances.nodes import langfuse
from app.config import settings

logger = logging.getLogger("agents")

router = APIRouter(prefix="/agents", tags=["agents"])

_graph = None


def _get_graph():
    global _graph
    if _graph is None:
        _graph = build_proposal_graph()
    return _graph


# ── Request models ────────────────────────────────────────────────────────

class ProposalLeadInput(BaseModel):
    id: str
    businessName: str
    registeredName: str | None = None
    email: str | None = None
    phone: str | None = None
    website: str | None = None
    city: str | None = None
    country: str | None = None
    description: str | None = None
    segment: str | None = None


class ProposalContactInput(BaseModel):
    name: str
    role: str | None = None
    email: str | None = None
    phone: str | None = None
    isPrimary: bool = False
    gender: str | None = None  # "male" | "female" | null (inferred from name if null)


class ProposalProductInput(BaseModel):
    name: str
    businessLine: str | None = None
    estimatedValue: float | None = None
    interestLevel: str | None = None
    notes: str | None = None


class ProposalRequest(BaseModel):
    jobId: str
    webhookUrl: str
    brand: str = "wb"  # "wb" | "salto"
    lead: ProposalLeadInput
    contacts: list[ProposalContactInput] = []
    products: list[ProposalProductInput] = []
    userInstructions: str = ""


class ProposalAnswerRequest(BaseModel):
    jobId: str
    webhookUrl: str
    answer: str


class AcceptedResponse(BaseModel):
    jobId: str
    status: str


# ── Background worker ─────────────────────────────────────────────────────

async def _run_proposal(job_id: str, state_input: dict, trace_id: str | None) -> None:
    trace = None
    if trace_id and langfuse.enabled:
        trace = langfuse.trace(name="crm-proposal-agent", id=trace_id)

    if trace:
        state_input["_trace"] = trace

    graph = _get_graph()
    try:
        result = await graph.ainvoke(state_input, config={"recursion_limit": 30})
    except Exception:
        logger.exception("[Proposal] background task crashed job=%s", job_id)
        if trace:
            trace.update(output={"error": "background_task_failed"})
            langfuse.flush()
        try:
            webhook_url = state_input.get("webhook_url", "")
            headers: dict = {"Content-Type": "application/json"}
            if settings.crm_webhook_secret:
                headers["X-Webhook-Secret"] = settings.crm_webhook_secret
            async with httpx.AsyncClient(timeout=10) as client:
                await client.post(
                    webhook_url,
                    json={"jobId": job_id, "status": "error", "error": "background_task_failed"},
                    headers=headers,
                )
        except Exception:
            logger.exception("[Proposal] fallback webhook also failed job=%s", job_id)
        return

    # If agent sent a question, persist context for the answer endpoint
    if result.get("question") and not result.get("error"):
        await save_job(job_id, {
            "job_id": job_id,
            "webhook_url": state_input["webhook_url"],
            "brand": state_input.get("brand", "wb"),
            "lead": state_input["lead"],
            "contacts": state_input["contacts"],
            "products": state_input["products"],
            "user_instructions": state_input["user_instructions"],
            "qa_history": state_input.get("qa_history", []) + [
                {"question": result["question"], "answer": ""}
            ],
        })
    else:
        # Job finished (completed or error) — clean up persisted context
        await delete_job(job_id)

    if trace:
        trace.update(output={
            "status": "completed" if result.get("proposal_number") else "error",
            "proposal_number": result.get("proposal_number"),
            "error": result.get("error"),
        })
        langfuse.flush()

    logger.info(
        "[Proposal] done job=%s number=%s error=%s",
        job_id, result.get("proposal_number"), result.get("error"),
    )


# ── Endpoints ─────────────────────────────────────────────────────────────

def _validate_secret(secret: str | None) -> None:
    expected = settings.crm_webhook_secret
    if expected and secret != expected:
        raise HTTPException(status_code=401, detail="Invalid webhook secret")


@router.post("/crm/proposal-agent", response_model=AcceptedResponse, status_code=202)
async def handle_proposal_agent(
    req: ProposalRequest,
    background_tasks: BackgroundTasks,
    x_webhook_secret: str | None = Header(default=None),
):
    _validate_secret(x_webhook_secret)

    trace_id = None
    if langfuse.enabled:
        trace = langfuse.trace(
            name="crm-proposal-agent",
            input={"jobId": req.jobId, "businessName": req.lead.businessName},
            metadata={"async": True},
        )
        trace_id = trace.id
        langfuse.flush()

    state_input = {
        "job_id": req.jobId,
        "webhook_url": req.webhookUrl,
        "brand": req.brand,
        "lead": req.lead.model_dump(),
        "contacts": [c.model_dump() for c in req.contacts],
        "products": [p.model_dump() for p in req.products],
        "user_instructions": req.userInstructions,
        "qa_history": [],
    }

    background_tasks.add_task(_run_proposal, req.jobId, state_input, trace_id)

    logger.info("[Proposal] accepted job=%s lead=%s", req.jobId, req.lead.businessName)
    return AcceptedResponse(jobId=req.jobId, status="accepted")


@router.post("/crm/proposal-agent/answer", response_model=AcceptedResponse, status_code=202)
async def handle_proposal_answer(
    req: ProposalAnswerRequest,
    background_tasks: BackgroundTasks,
    x_webhook_secret: str | None = Header(default=None),
):
    _validate_secret(x_webhook_secret)

    job = await load_job(req.jobId)
    if not job:
        raise HTTPException(status_code=404, detail=f"Job {req.jobId} not found or already completed")

    # Attach answer to the last pending question
    qa_history: list = job.get("qa_history", [])
    if qa_history and not qa_history[-1].get("answer"):
        qa_history[-1]["answer"] = req.answer
    else:
        qa_history.append({"question": "", "answer": req.answer})

    state_input = {
        "job_id": req.jobId,
        "webhook_url": req.webhookUrl or job["webhook_url"],
        "brand": job.get("brand", "wb"),
        "lead": job["lead"],
        "contacts": job["contacts"],
        "products": job["products"],
        "user_instructions": job["user_instructions"],
        "qa_history": qa_history,
    }

    background_tasks.add_task(_run_proposal, req.jobId, state_input, None)

    logger.info("[Proposal] answer received job=%s", req.jobId)
    return AcceptedResponse(jobId=req.jobId, status="accepted")
