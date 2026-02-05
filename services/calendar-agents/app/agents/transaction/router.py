from __future__ import annotations

from pydantic import BaseModel
from fastapi import APIRouter

from app.agents.transaction.evaluator import evaluate_trace
from app.agents.transaction.graph import build_transaction_graph
from app.agents.transaction.nodes import langfuse
from app.config import settings

router = APIRouter(prefix="/agents", tags=["agents"])

_graph = None


def _get_graph():
    global _graph
    if _graph is None:
        _graph = build_transaction_graph()
    return _graph


class TransactionRequest(BaseModel):
    phone: str
    text: str
    profile_id: str | None = None


class TransactionResponse(BaseModel):
    status: str
    reply: str
    transaction: dict | None = None
    error: str | None = None


@router.post("/transaction", response_model=TransactionResponse)
async def handle_transaction(req: TransactionRequest):
    profile_id = req.profile_id or settings.finance_personal_profile_id

    if not profile_id:
        return TransactionResponse(
            status="error",
            reply="profile_id não configurado. Defina FINANCE_PERSONAL_PROFILE_ID no .env",
            error="missing_profile",
        )

    # Create Langfuse trace for this request
    trace = langfuse.trace(
        name="transaction-agent",
        input={"phone": req.phone, "text": req.text},
        metadata={"profile_id": profile_id},
    ) if langfuse.enabled else None

    graph = _get_graph()
    result = await graph.ainvoke({
        "phone": req.phone,
        "raw_text": req.text,
        "profile_id": profile_id,
        "_trace": trace,
    })

    status = "error" if result.get("error") else "created"
    reply = result.get("reply", "Erro desconhecido")

    # Run LLM-as-judge evaluation
    if trace and result.get("parsed"):
        await evaluate_trace(trace, req.text, result["parsed"])

    # Finalize trace
    if trace:
        trace.update(output={"reply": reply, "status": status, "error": result.get("error")})
        langfuse.flush()

    if result.get("error"):
        return TransactionResponse(
            status="error",
            reply=reply,
            error=result.get("error"),
        )

    return TransactionResponse(
        status="created",
        reply=reply,
        transaction=result.get("transaction"),
    )
