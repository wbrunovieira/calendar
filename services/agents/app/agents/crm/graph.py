from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.nodes import (
    enrich_contacts,
    format_reply,
    investigate_leads,
    load_icp_context,
    save_to_crm,
    structure_leads_data,
    supervisor_review,
)
from app.agents.crm.state import CRMLeadState


def _after_load_icp(state: CRMLeadState) -> str:
    if state.get("error"):
        return END
    return "investigate_leads"


def _after_investigate(state: CRMLeadState) -> str:
    if state.get("error"):
        # On retry rounds with previous results, don't abort — format what we have
        if state.get("created_leads"):
            return "format_reply"
        return END
    return "structure_leads_data"


def _after_structure(state: CRMLeadState) -> str:
    if state.get("error"):
        return END
    return "supervisor_review"


def _after_supervisor(state: CRMLeadState) -> str:
    if state.get("error"):
        return "format_reply"
    if state.get("enrichment_indices"):
        return "enrich_contacts"
    if not state.get("approved_indices"):
        return "format_reply"
    return "save_to_crm"


def _after_enrich(state: CRMLeadState) -> str:
    if state.get("error"):
        return "format_reply"
    if not state.get("approved_indices"):
        return "format_reply"
    return "save_to_crm"


def _after_save(state: CRMLeadState) -> str:
    if state.get("error"):
        return "format_reply"
    remaining = state.get("remaining_count", 0)
    retry_round = state.get("retry_round", 0)
    if remaining > 0 and retry_round < 3:
        return "investigate_leads"
    return "format_reply"


def build_crm_graph():
    builder = StateGraph(CRMLeadState)

    builder.add_node("load_icp_context", load_icp_context)
    builder.add_node("investigate_leads", investigate_leads)
    builder.add_node("structure_leads_data", structure_leads_data)
    builder.add_node("supervisor_review", supervisor_review)
    builder.add_node("enrich_contacts", enrich_contacts)
    builder.add_node("save_to_crm", save_to_crm)
    builder.add_node("format_reply", format_reply)

    builder.add_edge(START, "load_icp_context")
    builder.add_conditional_edges("load_icp_context", _after_load_icp)
    builder.add_conditional_edges("investigate_leads", _after_investigate)
    builder.add_conditional_edges("structure_leads_data", _after_structure)
    builder.add_conditional_edges("supervisor_review", _after_supervisor)
    builder.add_conditional_edges("enrich_contacts", _after_enrich)
    builder.add_conditional_edges("save_to_crm", _after_save)
    builder.add_edge("format_reply", END)

    return builder.compile()
