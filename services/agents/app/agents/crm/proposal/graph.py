from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.proposal.nodes import (
    analyze_completeness,
    convert_to_pdf,
    layout_agent,
    merge_sections,
    send_asking_webhook,
    send_completed_webhook,
    send_error_webhook,
    write_commercial,
    write_narrative,
    write_optional_sections,
    write_technical,
)
from app.agents.crm.proposal.state import ProposalState


# ── Routing ───────────────────────────────────────────────────────────────

def _after_analyze(state: ProposalState) -> str:
    if state.get("error"):
        return "send_error"
    if state.get("is_sufficient"):
        return "fan_out"
    return "send_asking"


def _after_merge(state: ProposalState) -> str:
    if state.get("error"):
        return "send_error"
    return "layout_agent"


def _after_convert(state: ProposalState) -> str:
    if state.get("error"):
        return "send_error"
    return "send_completed"


async def _fan_out(state: ProposalState) -> dict:
    return {}


# ── Graph builder ─────────────────────────────────────────────────────────

def build_proposal_graph():
    builder = StateGraph(ProposalState)

    builder.add_node("analyze_completeness",    analyze_completeness)
    builder.add_node("fan_out",                 _fan_out)
    builder.add_node("write_narrative",         write_narrative)
    builder.add_node("write_technical",         write_technical)
    builder.add_node("write_commercial",        write_commercial)
    builder.add_node("write_optional_sections", write_optional_sections)
    builder.add_node("merge_sections",          merge_sections)
    builder.add_node("layout_agent",            layout_agent)
    builder.add_node("convert_pdf",             convert_to_pdf)
    builder.add_node("send_asking",             send_asking_webhook)
    builder.add_node("send_completed",          send_completed_webhook)
    builder.add_node("send_error",              send_error_webhook)

    builder.add_edge(START, "analyze_completeness")
    builder.add_conditional_edges("analyze_completeness", _after_analyze)

    builder.add_edge("fan_out", "write_narrative")
    builder.add_edge("fan_out", "write_technical")
    builder.add_edge("fan_out", "write_commercial")
    builder.add_edge("fan_out", "write_optional_sections")

    builder.add_edge("write_narrative",         "merge_sections")
    builder.add_edge("write_technical",         "merge_sections")
    builder.add_edge("write_commercial",        "merge_sections")
    builder.add_edge("write_optional_sections", "merge_sections")

    builder.add_conditional_edges("merge_sections", _after_merge)
    builder.add_edge("layout_agent",            "convert_pdf")
    builder.add_conditional_edges("convert_pdf",    _after_convert)

    builder.add_edge("send_asking",    END)
    builder.add_edge("send_completed", END)
    builder.add_edge("send_error",     END)

    return builder.compile()
