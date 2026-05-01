from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.transfer_analysis.nodes import (
    analyze_gk_phase,
    analyze_spiced_phase,
    segment_transcript,
    send_both_errors,
)
from app.agents.crm.transfer_analysis.state import TransferAnalysisState


def _after_segment_transcript(state: TransferAnalysisState) -> str:
    if state.get("error"):
        return "send_both_errors"
    return "analyze_gk_phase"


def build_transfer_analysis_graph():
    builder = StateGraph(TransferAnalysisState)

    builder.add_node("segment_transcript", segment_transcript)
    builder.add_node("analyze_gk_phase", analyze_gk_phase)
    builder.add_node("analyze_spiced_phase", analyze_spiced_phase)
    builder.add_node("send_both_errors", send_both_errors)

    builder.add_edge(START, "segment_transcript")
    builder.add_conditional_edges("segment_transcript", _after_segment_transcript)
    builder.add_edge("analyze_gk_phase", "analyze_spiced_phase")
    builder.add_edge("analyze_spiced_phase", END)
    builder.add_edge("send_both_errors", END)

    return builder.compile()
