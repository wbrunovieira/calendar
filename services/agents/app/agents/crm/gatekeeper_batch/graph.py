from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.gatekeeper_batch.nodes import (
    analyze_batch,
    build_batch_synthesis,
    compute_batch_scores,
    send_webhook,
)
from app.agents.crm.gatekeeper_batch.state import GatekeeperBatchState


def _after_compute_batch_scores(state: GatekeeperBatchState) -> str:
    if state.get("error"):
        return "send_webhook"
    return "analyze_batch"


def _after_analyze_batch(state: GatekeeperBatchState) -> str:
    if state.get("error"):
        return "send_webhook"
    return "build_batch_synthesis"


def _after_build_batch_synthesis(state: GatekeeperBatchState) -> str:
    if state.get("error"):
        return "send_webhook"
    return "send_webhook"


def build_gatekeeper_batch_graph():
    builder = StateGraph(GatekeeperBatchState)

    builder.add_node("compute_batch_scores", compute_batch_scores)
    builder.add_node("analyze_batch", analyze_batch)
    builder.add_node("build_batch_synthesis", build_batch_synthesis)
    builder.add_node("send_webhook", send_webhook)

    builder.add_edge(START, "compute_batch_scores")
    builder.add_conditional_edges("compute_batch_scores", _after_compute_batch_scores)
    builder.add_conditional_edges("analyze_batch", _after_analyze_batch)
    builder.add_edge("build_batch_synthesis", "send_webhook")
    builder.add_edge("send_webhook", END)

    return builder.compile()
