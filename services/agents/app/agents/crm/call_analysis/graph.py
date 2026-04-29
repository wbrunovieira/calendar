from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.call_analysis.nodes import (
    analyze_spiced,
    build_analysis,
    compute_scores,
    send_webhook,
)
from app.agents.crm.call_analysis.state import CallAnalysisState


def _after_analyze_spiced(state: CallAnalysisState) -> str:
    if state.get("error"):
        return "send_webhook"
    return "build_analysis"


def _after_build_analysis(state: CallAnalysisState) -> str:
    if state.get("error"):
        return "send_webhook"
    return "compute_scores"


def build_call_analysis_graph():
    builder = StateGraph(CallAnalysisState)

    builder.add_node("analyze_spiced", analyze_spiced)
    builder.add_node("build_analysis", build_analysis)
    builder.add_node("compute_scores", compute_scores)
    builder.add_node("send_webhook", send_webhook)

    builder.add_edge(START, "analyze_spiced")
    builder.add_conditional_edges("analyze_spiced", _after_analyze_spiced)
    builder.add_conditional_edges("build_analysis", _after_build_analysis)
    builder.add_edge("compute_scores", "send_webhook")
    builder.add_edge("send_webhook", END)

    return builder.compile()
