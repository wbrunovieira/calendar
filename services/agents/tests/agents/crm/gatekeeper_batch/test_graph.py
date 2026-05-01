from __future__ import annotations

from app.agents.crm.gatekeeper_batch.graph import (
    _after_analyze_batch,
    _after_build_batch_synthesis,
    _after_compute_batch_scores,
    build_gatekeeper_batch_graph,
)


def test_graph_compiles():
    graph = build_gatekeeper_batch_graph()
    assert graph is not None


def test_graph_has_expected_nodes():
    graph = build_gatekeeper_batch_graph()
    node_names = set(graph.nodes.keys())
    expected = {"compute_batch_scores", "analyze_batch", "build_batch_synthesis", "send_webhook"}
    assert expected.issubset(node_names)


def test_after_compute_batch_scores_continues_on_success():
    state = {"overall_score": 3.2, "dimension_averages": {}}
    assert _after_compute_batch_scores(state) == "analyze_batch"


def test_after_compute_batch_scores_routes_to_webhook_on_error():
    state = {"error": "no_analyses_provided"}
    assert _after_compute_batch_scores(state) == "send_webhook"


def test_after_analyze_batch_continues_on_success():
    state = {"patterns": {}, "comparison_with_history": {}}
    assert _after_analyze_batch(state) == "build_batch_synthesis"


def test_after_analyze_batch_routes_to_webhook_on_error():
    state = {"error": "analyze_batch_failed"}
    assert _after_analyze_batch(state) == "send_webhook"


def test_after_build_batch_synthesis_always_routes_to_webhook():
    assert _after_build_batch_synthesis({"new_summary": "ok"}) == "send_webhook"
    assert _after_build_batch_synthesis({"error": "build_batch_synthesis_failed"}) == "send_webhook"


# ── Pure function tests (no asyncio needed) ───────────────────────────────

from app.agents.crm.gatekeeper_batch.nodes import (
    _compute_dimension_averages,
    _extract_individual_highlights,
)


def _make_analysis(job_id: str, score: float, dim_scores: dict | None = None) -> dict:
    dims = dim_scores or {d: score for d in ("recepcao", "alianca", "perguntas", "objecoes", "resultado", "tecnicas")}
    raport = {d: {"score": s, "text": f"análise {d}"} for d, s in dims.items()}
    return {"jobId": job_id, "score": score, "raport": raport}


def test_compute_dimension_averages_calculates_correctly():
    analyses = [
        _make_analysis("j1", 4.0, {"recepcao": 4, "alianca": 3, "perguntas": 4, "objecoes": 2, "resultado": 3, "tecnicas": 3}),
        _make_analysis("j2", 4.0, {"recepcao": 5, "alianca": 4, "perguntas": 3, "objecoes": 3, "resultado": 4, "tecnicas": 4}),
    ]
    avgs = _compute_dimension_averages(analyses)
    assert avgs["recepcao"] == round((4 + 5) / 2, 1)
    assert avgs["objecoes"] == round((2 + 3) / 2, 1)


def test_compute_dimension_averages_handles_empty():
    assert _compute_dimension_averages([]) == {d: 0.0 for d in ("recepcao", "alianca", "perguntas", "objecoes", "resultado", "tecnicas")}


def test_compute_dimension_averages_handles_missing_raport():
    avgs = _compute_dimension_averages([{"jobId": "j1", "score": 3.0}])
    assert all(v == 0.0 for v in avgs.values())


def test_extract_highlights_returns_best_and_worst():
    analyses = [_make_analysis("j1", 5.0), _make_analysis("j2", 2.0), _make_analysis("j3", 3.5)]
    highlights = _extract_individual_highlights(analyses)
    types = {h["type"] for h in highlights}
    assert "best" in types and "worst" in types
    assert next(h for h in highlights if h["type"] == "best")["score"] == 5.0
    assert next(h for h in highlights if h["type"] == "worst")["score"] == 2.0


def test_extract_highlights_handles_empty():
    assert _extract_individual_highlights([]) == []


def test_extract_highlights_single_analysis():
    highlights = _extract_individual_highlights([_make_analysis("j1", 3.5)])
    assert len(highlights) == 1 and highlights[0]["type"] == "best"
