from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.meet_analysis.nodes import (
    analyze_dimensions,
    build_synthesis,
    compute_scores,
    send_webhook,
)
from app.agents.crm.meet_analysis.state import MeetAnalysisState


def _after_analyze_dimensions(state: MeetAnalysisState) -> str:
    if state.get("error"):
        return "send_webhook"
    return "build_synthesis"


def _after_build_synthesis(state: MeetAnalysisState) -> str:
    if state.get("error"):
        return "send_webhook"
    return "compute_scores"


def build_meet_analysis_graph():
    builder = StateGraph(MeetAnalysisState)

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
