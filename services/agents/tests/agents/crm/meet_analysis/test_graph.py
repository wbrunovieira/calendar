from __future__ import annotations

from app.agents.crm.meet_analysis.nodes import _compute_overall_score
from app.agents.crm.meet_analysis.graph import (
    _after_analyze_dimensions,
    _after_build_synthesis,
    build_meet_analysis_graph,
)


def test_graph_compiles():
    graph = build_meet_analysis_graph()
    assert graph is not None


def test_graph_has_expected_nodes():
    graph = build_meet_analysis_graph()
    node_names = set(graph.nodes.keys())
    expected = {"analyze_dimensions", "build_synthesis", "compute_scores", "send_webhook"}
    assert expected.issubset(node_names)


def test_after_analyze_dimensions_continues_on_success():
    state = {"business": {}, "gaps": {}, "urgency": {}, "decision_power": {}, "engagement": {}, "closing": {}}
    assert _after_analyze_dimensions(state) == "build_synthesis"


def test_after_analyze_dimensions_routes_to_webhook_on_error():
    state = {"error": "analyze_dimensions_failed"}
    assert _after_analyze_dimensions(state) == "send_webhook"


def test_after_build_synthesis_continues_on_success():
    state = {"summary": "ok", "next_step": "send proposal"}
    assert _after_build_synthesis(state) == "compute_scores"


def test_after_build_synthesis_routes_to_webhook_on_error():
    state = {"error": "build_synthesis_failed"}
    assert _after_build_synthesis(state) == "send_webhook"


# ── _compute_overall_score (pure function, no asyncio needed) ─────────────


def test_compute_overall_score_averages_six_dimensions():
    scores = {"business": 4, "gaps": 3, "urgency": 5, "decision_power": 4, "engagement": 4, "closing": 4}
    result = _compute_overall_score(scores)
    assert result == round((4 + 3 + 5 + 4 + 4 + 4) / 6, 1)


def test_compute_overall_score_handles_missing_scores():
    scores = {"business": 4, "gaps": 0, "urgency": 5, "decision_power": 0, "engagement": 4, "closing": 4}
    result = _compute_overall_score(scores)
    assert isinstance(result, float)


def test_compute_overall_score_clamps_to_valid_range():
    scores = {"business": 5, "gaps": 5, "urgency": 5, "decision_power": 5, "engagement": 5, "closing": 5}
    result = _compute_overall_score(scores)
    assert result <= 5.0
