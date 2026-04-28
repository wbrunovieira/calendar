from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.crm.deep_research.nodes import (
    extract_updates,
    generate_summary,
    plan_research,
    run_searches,
)
from app.agents.crm.deep_research.state import DeepResearchState


def _after_plan(state: DeepResearchState) -> str:
    if state.get("error"):
        return END
    # Nothing to research — lead is already complete
    if not state.get("missing_fields"):
        return "generate_summary"
    return "run_searches"


def _after_searches(state: DeepResearchState) -> str:
    if state.get("error"):
        return "generate_summary"
    return "extract_updates"


def build_deep_research_graph():
    builder = StateGraph(DeepResearchState)

    builder.add_node("plan_research", plan_research)
    builder.add_node("run_searches", run_searches)
    builder.add_node("extract_updates", extract_updates)
    builder.add_node("generate_summary", generate_summary)

    builder.add_edge(START, "plan_research")
    builder.add_conditional_edges("plan_research", _after_plan)
    builder.add_conditional_edges("run_searches", _after_searches)
    builder.add_edge("extract_updates", "generate_summary")
    builder.add_edge("generate_summary", END)

    return builder.compile()
