from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.gatekeeper_analysis.nodes import (
    analyze_dimensions,
    build_synthesis,
    compute_scores,
    send_webhook,
)
from app.agents.crm.gatekeeper_analysis.state import GatekeeperAnalysisState


def _after_analyze_dimensions(state: GatekeeperAnalysisState) -> str:
    if state.get("error"):
        return "send_webhook"
    return "build_synthesis"


def _after_build_synthesis(state: GatekeeperAnalysisState) -> str:
    if state.get("error"):
        return "send_webhook"
    return "compute_scores"


def build_gatekeeper_analysis_graph():
    builder = StateGraph(GatekeeperAnalysisState)

    builder.add_node("analyze_dimensions", analyze_dimensions)
    builder.add_node("build_synthesis", build_synthesis)
    builder.add_node("compute_scores", compute_scores)
    builder.add_node("send_webhook", send_webhook)

    builder.add_edge(START, "analyze_dimensions")
    builder.add_conditional_edges("analyze_dimensions", _after_analyze_dimensions)
    builder.add_conditional_edges("build_synthesis", _after_build_synthesis)
    builder.add_edge("compute_scores", "send_webhook")
    builder.add_edge("send_webhook", END)

    return builder.compile()
