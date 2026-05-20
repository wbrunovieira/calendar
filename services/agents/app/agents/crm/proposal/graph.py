from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.proposal.nodes import (
    analyze_completeness,
    convert_to_pdf,
    merge_sections,
    send_asking_webhook,
    send_completed_webhook,
    send_error_webhook,
    upload_to_drive,
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
    return "convert_pdf"


def _after_convert(state: ProposalState) -> str:
    if state.get("error"):
        return "send_error"
    return "upload_drive"


def _after_upload(state: ProposalState) -> str:
    if state.get("error"):
        return "send_error"
    return "send_completed"


# ── No-op router that fans out to 3 parallel generation nodes ─────────────

async def _fan_out(state: ProposalState) -> dict:
    return {}


# ── Graph builder ─────────────────────────────────────────────────────────

def build_proposal_graph():
    builder = StateGraph(ProposalState)

    # Analysis
    builder.add_node("analyze_completeness", analyze_completeness)

    # Fan-out router
    builder.add_node("fan_out", _fan_out)

    # Parallel generation (Sonnet × 2 + Haiku × 2)
    builder.add_node("write_narrative",          write_narrative)
    builder.add_node("write_technical",          write_technical)
    builder.add_node("write_commercial",         write_commercial)
    builder.add_node("write_optional_sections",  write_optional_sections)

    # Merge + PDF pipeline
    builder.add_node("merge_sections",    merge_sections)
    builder.add_node("convert_pdf",       convert_to_pdf)
    builder.add_node("upload_drive",      upload_to_drive)

    # Webhooks
    builder.add_node("send_asking",       send_asking_webhook)
    builder.add_node("send_completed",    send_completed_webhook)
    builder.add_node("send_error",        send_error_webhook)

    # ── Edges ─────────────────────────────────────────────────────────────

    builder.add_edge(START, "analyze_completeness")
    builder.add_conditional_edges("analyze_completeness", _after_analyze)

    # Fan-out: 4 nodes run in parallel
    builder.add_edge("fan_out", "write_narrative")
    builder.add_edge("fan_out", "write_technical")
    builder.add_edge("fan_out", "write_commercial")
    builder.add_edge("fan_out", "write_optional_sections")

    # Fan-in: all 4 converge to merge
    builder.add_edge("write_narrative",         "merge_sections")
    builder.add_edge("write_technical",         "merge_sections")
    builder.add_edge("write_commercial",        "merge_sections")
    builder.add_edge("write_optional_sections", "merge_sections")

    builder.add_conditional_edges("merge_sections", _after_merge)
    builder.add_conditional_edges("convert_pdf",    _after_convert)
    builder.add_conditional_edges("upload_drive",   _after_upload)

    builder.add_edge("send_asking",    END)
    builder.add_edge("send_completed", END)
    builder.add_edge("send_error",     END)

    return builder.compile()
