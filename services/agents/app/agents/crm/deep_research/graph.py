from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.deep_research.nodes import (
    assess_previous_research,
    enrich_instagram,
    extract_focused_update,
    extract_updates,
    fetch_cnpj_if_discovered,
    generate_focused_summary,
    generate_summary,
    plan_focused_research,
    plan_research,
    run_searches,
)
from app.agents.crm.deep_research.state import DeepResearchState


def _after_start(state: DeepResearchState) -> str:
    if state.get("focus_field"):
        return "plan_focused_research"
    if state.get("previous_summary"):
        return "assess_previous_research"
    return "plan_research"


def _after_plan(state: DeepResearchState) -> str:
    if state.get("error"):
        return END
    if not state.get("missing_fields"):
        return "generate_summary"
    return "run_searches"


def _after_searches(state: DeepResearchState) -> str:
    """Unified dispatch: routes to focused or general extract based on mode."""
    if state.get("focus_field"):
        if state.get("error"):
            return "generate_focused_summary"
        return "extract_focused_update"
    if state.get("error"):
        return "generate_summary"
    return "extract_updates"


def _after_fetch_cnpj(state: DeepResearchState) -> str:
    updates = state.get("updates", {})
    forced_updates = state.get("forced_updates") or {}
    lead = state.get("lead", {})
    has_instagram = updates.get("instagram") or forced_updates.get("instagram") or lead.get("instagram")
    if has_instagram:
        return "enrich_instagram"
    return "generate_summary"


def _after_focused_searches(state: DeepResearchState) -> str:
    if state.get("error"):
        return "generate_focused_summary"
    return "extract_focused_update"


def _after_focused_extract(state: DeepResearchState) -> str:
    focus = state.get("focus_field", "")
    forced = state.get("forced_updates") or {}
    # If instagram was found in focused mode, enrich it with metrics
    if focus == "instagram" and forced.get("instagram"):
        return "enrich_instagram"
    return "generate_focused_summary"


def build_deep_research_graph():
    builder = StateGraph(DeepResearchState)

    # General research path
    builder.add_node("assess_previous_research", assess_previous_research)
    builder.add_node("plan_research", plan_research)
    builder.add_node("run_searches", run_searches)
    builder.add_node("extract_updates", extract_updates)
    builder.add_node("fetch_cnpj_if_discovered", fetch_cnpj_if_discovered)
    builder.add_node("enrich_instagram", enrich_instagram)
    builder.add_node("generate_summary", generate_summary)

    # Focused research path
    builder.add_node("plan_focused_research", plan_focused_research)
    builder.add_node("extract_focused_update", extract_focused_update)
    builder.add_node("generate_focused_summary", generate_focused_summary)

    # General path edges
    builder.add_conditional_edges(START, _after_start)
    builder.add_edge("assess_previous_research", "plan_research")
    builder.add_conditional_edges("plan_research", _after_plan)
    builder.add_conditional_edges("run_searches", _after_searches)
    builder.add_edge("extract_updates", "fetch_cnpj_if_discovered")
    builder.add_conditional_edges("fetch_cnpj_if_discovered", _after_fetch_cnpj)
    builder.add_edge("enrich_instagram", "generate_summary")
    builder.add_edge("generate_summary", END)

    # Focused path edges (reuses run_searches via unified _after_searches dispatch)
    builder.add_edge("plan_focused_research", "run_searches")
    builder.add_conditional_edges("extract_focused_update", _after_focused_extract)
    builder.add_edge("generate_focused_summary", END)

    return builder.compile()
