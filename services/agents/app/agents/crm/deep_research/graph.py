from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.deep_research.nodes import (
    assess_previous_research,
    enrich_instagram,
    extract_updates,
    generate_summary,
    plan_research,
    run_searches,
)
from app.agents.crm.deep_research.state import DeepResearchState


def _after_start(state: DeepResearchState) -> str:
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
    if state.get("error"):
        return "generate_summary"
    return "extract_updates"


def _after_extract(state: DeepResearchState) -> str:
    updates = state.get("updates", {})
    lead = state.get("lead", {})
    has_instagram = updates.get("instagram") or lead.get("instagram")
    if has_instagram:
        return "enrich_instagram"
    return "generate_summary"


def build_deep_research_graph():
    builder = StateGraph(DeepResearchState)

    builder.add_node("assess_previous_research", assess_previous_research)
    builder.add_node("plan_research", plan_research)
    builder.add_node("run_searches", run_searches)
    builder.add_node("extract_updates", extract_updates)
    builder.add_node("enrich_instagram", enrich_instagram)
    builder.add_node("generate_summary", generate_summary)

    builder.add_conditional_edges(START, _after_start)
    builder.add_edge("assess_previous_research", "plan_research")
    builder.add_conditional_edges("plan_research", _after_plan)
    builder.add_conditional_edges("run_searches", _after_searches)
    builder.add_conditional_edges("extract_updates", _after_extract)
    builder.add_edge("enrich_instagram", "generate_summary")
    builder.add_edge("generate_summary", END)

    return builder.compile()
