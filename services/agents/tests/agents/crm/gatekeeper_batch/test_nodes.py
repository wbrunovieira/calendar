from __future__ import annotations

import json
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from app.agents.crm.gatekeeper_batch.nodes import (
    _analyze_patterns,
    _compare_with_history,
    _compute_dimension_averages,
    _extract_individual_highlights,
    analyze_batch,
    build_batch_synthesis,
    compute_batch_scores,
    send_webhook,
)

pytestmark = pytest.mark.asyncio(loop_scope="function")

_MODULE = "app.agents.crm.gatekeeper_batch.nodes"


def _mock_claude(text: str) -> MagicMock:
    block = SimpleNamespace(type="text", text=text)
    resp = MagicMock()
    resp.content = [block]
    resp.stop_reason = "end_turn"
    return resp


def _make_analysis(job_id: str, score: float, dim_scores: dict | None = None) -> dict:
    dims = dim_scores or {d: score for d in ("recepcao", "alianca", "perguntas", "objecoes", "resultado", "tecnicas")}
    raport = {d: {"score": s, "text": f"análise {d}"} for d, s in dims.items()}
    return {"jobId": job_id, "score": score, "raport": raport}


def _patterns_payload():
    return {
        "working": ["Tom profissional consistente", "Aliança com GK bem executada"],
        "notWorking": ["Cede ao 'manda email' sem resposta"],
        "keepDoing": ["Uso do nome do porteiro"],
    }


def _history_payload():
    return {
        "improved": ["recepcao melhorou"],
        "worsened": [],
        "unchanged": ["tecnicas estável"],
        "firstBatch": False,
    }


def _synthesis_payload():
    return {
        "newSummary": "Lote com 5 ligações, score médio 3.2. Equipe evolui em aliança.",
        "positivePoints": ["Tom consistente", "Aliança com GK"],
        "improvementPoints": ["Resposta ao manda-email"],
        "recommendations": [
            {"priority": "high", "action": "Treinar resposta ao manda-email", "targetDimension": "objecoes"},
        ],
    }


# ── compute_batch_scores node ─────────────────────────────────────────────


async def test_compute_batch_scores_returns_overall_and_averages():
    analyses = [
        _make_analysis("j1", 4.0),
        _make_analysis("j2", 3.0),
        _make_analysis("j3", 5.0),
    ]
    state = {"batch_job_id": "batch-1", "analyses": analyses}
    result = await compute_batch_scores(state)
    assert "overall_score" in result
    assert result["overall_score"] == round((4.0 + 3.0 + 5.0) / 3, 1)
    assert "dimension_averages" in result
    assert "individual_highlights" in result
    assert "error" not in result


async def test_compute_batch_scores_returns_error_on_empty():
    result = await compute_batch_scores({"batch_job_id": "batch-1", "analyses": []})
    assert "error" in result


# ── _analyze_patterns ─────────────────────────────────────────────────────


async def test_analyze_patterns_returns_expected_fields():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_patterns_payload()))):
        result = await _analyze_patterns("batch-1", [], {}, [], None)
    assert "working" in result
    assert "notWorking" in result
    assert "keepDoing" in result
    assert "error" not in result


async def test_analyze_patterns_returns_error_on_llm_failure():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock, side_effect=RuntimeError("api error")):
        result = await _analyze_patterns("batch-1", [], {}, [], None)
    assert "error" in result


async def test_analyze_patterns_returns_error_on_invalid_json():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude("not json")):
        result = await _analyze_patterns("batch-1", [], {}, [], None)
    assert "error" in result


# ── _compare_with_history ─────────────────────────────────────────────────


async def test_compare_with_history_returns_expected_fields():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_history_payload()))):
        result = await _compare_with_history({}, [], None)
    assert "improved" in result
    assert "worsened" in result
    assert "unchanged" in result
    assert "firstBatch" in result
    assert "error" not in result


async def test_compare_with_history_returns_error_on_failure():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock, side_effect=RuntimeError("err")):
        result = await _compare_with_history({}, [], None)
    assert "error" in result


# ── analyze_batch node ────────────────────────────────────────────────────


async def test_analyze_batch_returns_patterns_and_comparison():
    call_count = 0

    async def _fake(system, user, model, max_tokens):
        nonlocal call_count
        payloads = [_patterns_payload(), _history_payload()]
        resp = _mock_claude(json.dumps(payloads[call_count]))
        call_count += 1
        return resp

    state = {
        "batch_job_id": "batch-1",
        "analyses": [_make_analysis("j1", 3.0)],
        "historical_summaries": [],
        "dimension_averages": {},
        "individual_highlights": [],
        "_trace": None,
    }
    with patch(f"{_MODULE}._call_claude", side_effect=_fake):
        result = await analyze_batch(state)

    assert "patterns" in result
    assert "comparison_with_history" in result
    assert "error" not in result


async def test_analyze_batch_returns_error_on_failure():
    async def _fail(*a, **kw):
        raise RuntimeError("fail")

    state = {
        "batch_job_id": "b1",
        "analyses": [], "historical_summaries": [],
        "dimension_averages": {}, "individual_highlights": [],
        "_trace": None,
    }
    with patch(f"{_MODULE}._call_claude", side_effect=_fail):
        result = await analyze_batch(state)
    assert "error" in result


# ── build_batch_synthesis node ────────────────────────────────────────────


async def test_build_batch_synthesis_returns_expected_structure():
    state = {
        "batch_job_id": "batch-1",
        "analyses": [_make_analysis("j1", 3.0)],
        "overall_score": 3.0,
        "dimension_averages": {"recepcao": 3.0},
        "patterns": _patterns_payload(),
        "comparison_with_history": _history_payload(),
        "_trace": None,
    }
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_synthesis_payload()))):
        result = await build_batch_synthesis(state)

    assert "new_summary" in result
    assert "positive_points" in result
    assert "improvement_points" in result
    assert "recommendations" in result
    assert isinstance(result["recommendations"], list)
    assert "error" not in result


async def test_build_batch_synthesis_returns_error_on_failure():
    state = {
        "batch_job_id": "b1", "analyses": [], "overall_score": 0.0,
        "dimension_averages": {}, "patterns": {}, "comparison_with_history": {}, "_trace": None,
    }
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock, side_effect=RuntimeError("err")):
        result = await build_batch_synthesis(state)
    assert "error" in result


# ── send_webhook node ─────────────────────────────────────────────────────


async def test_send_webhook_completed_posts_full_payload():
    state = {
        "batch_job_id": "batch-1",
        "webhook_url": "https://crm.example.com/webhooks/gatekeeper-batch",
        "overall_score": 3.2,
        "dimension_averages": {"recepcao": 3.5, "alianca": 3.0},
        "individual_highlights": [{"type": "best", "jobId": "j1", "score": 5.0}],
        "patterns": _patterns_payload(),
        "comparison_with_history": _history_payload(),
        "recommendations": _synthesis_payload()["recommendations"],
        "new_summary": "Resumo do lote.",
        "positive_points": ["Tom consistente"],
        "improvement_points": ["Resposta ao manda-email"],
        "error": None,
    }
    mock_resp = MagicMock()
    mock_resp.status_code = 200

    with patch("httpx.AsyncClient") as mock_cls:
        mc = AsyncMock()
        mc.__aenter__ = AsyncMock(return_value=mc)
        mc.__aexit__ = AsyncMock(return_value=False)
        mc.post = AsyncMock(return_value=mock_resp)
        mock_cls.return_value = mc
        await send_webhook(state)

    mc.post.assert_called_once()
    payload = mc.post.call_args[1]["json"]
    assert payload["batchJobId"] == "batch-1"
    assert payload["status"] == "completed"
    assert payload["overallScore"] == 3.2
    assert "dimensionAverages" in payload
    assert "patterns" in payload
    assert "comparisonWithHistory" in payload
    assert "recommendations" in payload
    assert "newSummary" in payload
    assert "positivePoints" in payload
    assert "improvementPoints" in payload


async def test_send_webhook_error_state_posts_error_payload():
    state = {
        "batch_job_id": "batch-err",
        "webhook_url": "https://crm.example.com/webhooks/gatekeeper-batch",
        "error": "analyze_batch_failed",
    }
    mock_resp = MagicMock()
    mock_resp.status_code = 200

    with patch("httpx.AsyncClient") as mock_cls:
        mc = AsyncMock()
        mc.__aenter__ = AsyncMock(return_value=mc)
        mc.__aexit__ = AsyncMock(return_value=False)
        mc.post = AsyncMock(return_value=mock_resp)
        mock_cls.return_value = mc
        await send_webhook(state)

    payload = mc.post.call_args[1]["json"]
    assert payload["status"] == "error"
    assert payload["error"] == "analyze_batch_failed"


async def test_send_webhook_no_url_skips():
    state = {"batch_job_id": "b1", "webhook_url": "", "error": None}
    with patch("httpx.AsyncClient") as mock_cls:
        mc = AsyncMock()
        mock_cls.return_value = mc
        await send_webhook(state)
    mc.__aenter__.assert_not_called()
