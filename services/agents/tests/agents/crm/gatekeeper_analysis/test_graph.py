from __future__ import annotations

import asyncio

from app.agents.crm.gatekeeper_analysis.nodes import compute_scores
from app.agents.crm.gatekeeper_analysis.graph import (
    _after_analyze_dimensions,
    _after_build_synthesis,
    build_gatekeeper_analysis_graph,
)


def test_graph_compiles():
    graph = build_gatekeeper_analysis_graph()
    assert graph is not None


def test_graph_has_expected_nodes():
    graph = build_gatekeeper_analysis_graph()
    node_names = set(graph.nodes.keys())
    expected = {"analyze_dimensions", "build_synthesis", "compute_scores", "send_webhook"}
    assert expected.issubset(node_names)


def test_after_analyze_dimensions_continues_on_success():
    state = {
        "recepcao": {}, "alianca": {}, "perguntas": {},
        "objecoes": {}, "resultado": {}, "tecnicas": {},
    }
    assert _after_analyze_dimensions(state) == "build_synthesis"


def test_after_analyze_dimensions_routes_to_webhook_on_error():
    state = {"error": "analyze_dimensions_failed"}
    assert _after_analyze_dimensions(state) == "send_webhook"


def test_after_build_synthesis_continues_on_success():
    state = {"summary": "ok", "positive_points": [], "improvement_points": []}
    assert _after_build_synthesis(state) == "compute_scores"


def test_after_build_synthesis_routes_to_webhook_on_error():
    state = {"error": "build_synthesis_failed"}
    assert _after_build_synthesis(state) == "send_webhook"


# ── compute_scores (pure avg, no asyncio needed here — run as sync via event loop) ──

import asyncio


def test_compute_scores_averages_six():
    state = {
        "job_id": "j1",
        "recepcao":  {"score": 4},
        "alianca":   {"score": 3},
        "perguntas": {"score": 4},
        "objecoes":  {"score": 2},
        "resultado": {"score": 3},
        "tecnicas":  {"score": 3},
    }
    result = asyncio.new_event_loop().run_until_complete(compute_scores(state))
    expected = round((4 + 3 + 4 + 2 + 3 + 3) / 6, 1)
    assert result["score"] == expected


def test_compute_scores_handles_empty_state():
    result = asyncio.new_event_loop().run_until_complete(compute_scores({"job_id": "j1"}))
    assert "score" in result
    assert isinstance(result["score"], float)
