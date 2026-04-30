from __future__ import annotations

import json
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from app.agents.crm.meet_analysis.nodes import (
    _analyze_business,
    _analyze_closing,
    _analyze_decision_power,
    _analyze_engagement,
    _analyze_gaps,
    _analyze_urgency,
    analyze_dimensions,
    build_synthesis,
    compute_scores,
    send_webhook,
)

pytestmark = pytest.mark.asyncio(loop_scope="function")

_MODULE = "app.agents.crm.meet_analysis.nodes"

_LEAD = {"businessName": "TechCo", "description": "SaaS B2B", "segment": "Tech", "city": "São Paulo"}
_CONTACT = {"name": "João Silva", "role": "CEO"}


def _mock_claude(text: str) -> MagicMock:
    block = SimpleNamespace(type="text", text=text)
    resp = MagicMock()
    resp.content = [block]
    resp.stop_reason = "end_turn"
    return resp


def _business_payload() -> dict:
    return {
        "currentRevenue": "R$ 3M/ano",
        "model": "SaaS B2B",
        "customers": "~50 clientes",
        "averageTicket": "R$ 5k/mês",
        "objective": "dobrar ARR em 12 meses",
        "alreadyTried": "Google Ads sem resultado",
        "text": "Análise completa...",
        "score": 4,
    }


def _gaps_payload() -> dict:
    return {
        "aquisicao":     {"identified": True,  "text": "Sem prospecção ativa", "severity": "high"},
        "funil":         {"identified": True,  "text": "Conversão < 10%",      "severity": "high"},
        "oferta":        {"identified": False, "text": None,                   "severity": None},
        "timeComercial": {"identified": True,  "text": "Sem metodologia",      "severity": "medium"},
        "retencao":      {"identified": False, "text": None,                   "severity": None},
        "dados":         {"identified": True,  "text": "Sem CRM",              "severity": "medium"},
        "text": "Gargalos encontrados em 4 de 6 áreas.",
        "score": 3,
    }


def _urgency_payload() -> dict:
    return {
        "trigger": "Vai contratar 2 SDRs em maio",
        "criticalEvent": "Contratações em 30 dias",
        "consequence": "Contratar sem processo = dinheiro perdido",
        "text": "Urgência alta...",
        "score": 5,
    }


def _decision_payload() -> dict:
    return {
        "decisionMaker": "CEO presente, decide sozinho",
        "buyingProcess": "Aprovação informal",
        "budget": "Mencionou 'investimento razoável'",
        "text": "Poder de decisão alto...",
        "score": 4,
    }


def _engagement_payload() -> dict:
    return {
        "level": "high",
        "buyerQuestions": ["Como funciona a implementação?", "Quanto tempo leva?"],
        "resistances": ["Preocupado com tempo"],
        "rapport": "Boa conexão",
        "text": "Engajamento alto...",
        "score": 4,
    }


def _closing_payload() -> dict:
    return {
        "buyerSignals": ["Perguntou sobre prazo", "Pediu cases"],
        "sellerCloses": {
            "trialClose":      {"used": True,  "notes": "Perguntou 'faz sentido?'"},
            "nextStepAnchor":  {"used": True,  "notes": "Agendou proposta para sexta"},
            "urgencyCreation": {"used": False, "notes": None},
        },
        "closingProbability": 72,
        "text": "Sinais de compra presentes...",
        "score": 4,
    }


def _synthesis_payload() -> dict:
    return {
        "summary": "Reunião produtiva com CEO de SaaS B2B.",
        "nextStep": "Enviar proposta até quarta",
        "positivePoints": ["Dor clara", "Decisor presente"],
        "improvementPoints": ["Não criou urgência explícita"],
    }


# ── Individual dimension helpers ──────────────────────────────────────────


async def test_analyze_business_returns_expected_fields():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_business_payload()))):
        result = await _analyze_business(_LEAD, _CONTACT, "transcript...", 60, "2026-04-30", "Reunião DIAG", None)

    for key in ("currentRevenue", "model", "customers", "averageTicket", "objective", "alreadyTried", "text", "score"):
        assert key in result, f"missing key: {key}"
    assert "error" not in result


async def test_analyze_gaps_returns_all_six_areas():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_gaps_payload()))):
        result = await _analyze_gaps(_LEAD, _CONTACT, "transcript...", 60, None)

    for area in ("aquisicao", "funil", "oferta", "timeComercial", "retencao", "dados"):
        assert area in result, f"missing gap area: {area}"
    assert "score" in result
    assert "error" not in result


async def test_analyze_urgency_returns_expected_fields():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_urgency_payload()))):
        result = await _analyze_urgency(_LEAD, _CONTACT, "transcript...", 60, None)

    for key in ("trigger", "criticalEvent", "consequence", "text", "score"):
        assert key in result
    assert "error" not in result


async def test_analyze_decision_power_returns_expected_fields():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_decision_payload()))):
        result = await _analyze_decision_power(_LEAD, _CONTACT, "transcript...", 60, None)

    for key in ("decisionMaker", "buyingProcess", "budget", "text", "score"):
        assert key in result
    assert "error" not in result


async def test_analyze_engagement_returns_expected_fields():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_engagement_payload()))):
        result = await _analyze_engagement(_LEAD, _CONTACT, "transcript...", 60, None)

    for key in ("level", "buyerQuestions", "resistances", "rapport", "text", "score"):
        assert key in result
    assert isinstance(result["buyerQuestions"], list)
    assert isinstance(result["resistances"], list)
    assert "error" not in result


async def test_analyze_closing_returns_expected_fields():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_closing_payload()))):
        result = await _analyze_closing(_LEAD, _CONTACT, "transcript...", 60, None)

    for key in ("buyerSignals", "sellerCloses", "closingProbability", "text", "score"):
        assert key in result
    assert isinstance(result["buyerSignals"], list)
    assert "trialClose" in result["sellerCloses"]
    assert "error" not in result


async def test_dimension_helper_returns_error_on_llm_exception():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock, side_effect=RuntimeError("timeout")):
        result = await _analyze_business(_LEAD, _CONTACT, "transcript...", 60, "2026-04-30", "Reunião", None)

    assert "error" in result


async def test_dimension_helper_returns_error_on_invalid_json():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude("not json at all")):
        result = await _analyze_urgency(_LEAD, _CONTACT, "transcript...", 60, None)

    assert "error" in result


# ── analyze_dimensions (LangGraph node) ───────────────────────────────────


async def test_analyze_dimensions_success_returns_all_six():
    payloads = [
        _business_payload(),
        _gaps_payload(),
        _urgency_payload(),
        _decision_payload(),
        _engagement_payload(),
        _closing_payload(),
    ]
    call_count = 0

    async def _fake_claude(*args, **kwargs):
        nonlocal call_count
        resp = _mock_claude(json.dumps(payloads[call_count]))
        call_count += 1
        return resp

    state = {
        "job_id": "job-1",
        "lead": _LEAD,
        "contact": _CONTACT,
        "transcript": "transcript...",
        "meeting_duration_seconds": 3600,
        "meeting_date": "2026-04-30",
        "meeting_title": "Reunião DIAG",
        "_trace": None,
    }

    with patch(f"{_MODULE}._call_claude", side_effect=_fake_claude):
        result = await analyze_dimensions(state)

    for key in ("business", "gaps", "urgency", "decision_power", "engagement", "closing"):
        assert key in result, f"missing dimension: {key}"
    assert "error" not in result


async def test_analyze_dimensions_returns_error_when_one_fails():
    async def _failing_claude(*args, **kwargs):
        raise RuntimeError("API error")

    state = {
        "job_id": "job-1",
        "lead": _LEAD,
        "contact": _CONTACT,
        "transcript": "transcript...",
        "meeting_duration_seconds": 3600,
        "meeting_date": "2026-04-30",
        "meeting_title": "Reunião DIAG",
        "_trace": None,
    }

    with patch(f"{_MODULE}._call_claude", side_effect=_failing_claude):
        result = await analyze_dimensions(state)

    assert "error" in result


# ── build_synthesis (LangGraph node) ──────────────────────────────────────


async def test_build_synthesis_returns_expected_structure():
    state = {
        "job_id": "job-1",
        "lead": _LEAD,
        "contact": _CONTACT,
        "business": _business_payload(),
        "gaps": _gaps_payload(),
        "urgency": _urgency_payload(),
        "decision_power": _decision_payload(),
        "engagement": _engagement_payload(),
        "closing": _closing_payload(),
        "_trace": None,
    }

    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_synthesis_payload()))):
        result = await build_synthesis(state)

    for key in ("summary", "next_step", "positive_points", "improvement_points"):
        assert key in result, f"missing key: {key}"
    assert isinstance(result["positive_points"], list)
    assert isinstance(result["improvement_points"], list)
    assert "error" not in result


async def test_build_synthesis_returns_error_on_llm_failure():
    state = {
        "job_id": "job-1",
        "lead": _LEAD,
        "contact": _CONTACT,
        "business": _business_payload(),
        "gaps": _gaps_payload(),
        "urgency": _urgency_payload(),
        "decision_power": _decision_payload(),
        "engagement": _engagement_payload(),
        "closing": _closing_payload(),
        "_trace": None,
    }

    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock, side_effect=RuntimeError("timeout")):
        result = await build_synthesis(state)

    assert "error" in result


async def test_build_synthesis_does_not_receive_transcript():
    """synthesis node must NOT receive the raw transcript — only the 6 structured dicts."""
    captured_calls = []

    async def _capture(*args, **kwargs):
        captured_calls.append(kwargs.get("user", args[1] if len(args) > 1 else ""))
        return _mock_claude(json.dumps(_synthesis_payload()))

    state = {
        "job_id": "job-1",
        "lead": _LEAD,
        "contact": _CONTACT,
        "transcript": "SHOULD_NOT_APPEAR_IN_SYNTHESIS_PROMPT",
        "business": _business_payload(),
        "gaps": _gaps_payload(),
        "urgency": _urgency_payload(),
        "decision_power": _decision_payload(),
        "engagement": _engagement_payload(),
        "closing": _closing_payload(),
        "_trace": None,
    }

    with patch(f"{_MODULE}._call_claude", side_effect=_capture):
        await build_synthesis(state)

    assert captured_calls, "no LLM call made"
    assert "SHOULD_NOT_APPEAR_IN_SYNTHESIS_PROMPT" not in captured_calls[0]


# ── compute_scores (LangGraph node) ───────────────────────────────────────


async def test_compute_scores_extracts_closing_probability():
    state = {
        "job_id": "job-1",
        "business": _business_payload(),
        "gaps": _gaps_payload(),
        "urgency": _urgency_payload(),
        "decision_power": _decision_payload(),
        "engagement": _engagement_payload(),
        "closing": _closing_payload(),
    }
    result = await compute_scores(state)

    assert result["closing_probability"] == 72
    assert "score" in result
    assert isinstance(result["score"], float)


async def test_compute_scores_score_is_average_of_six():
    closing = _closing_payload()
    closing["score"] = 2

    state = {
        "job_id": "job-1",
        "business":       {**_business_payload(),  "score": 4},
        "gaps":           {**_gaps_payload(),       "score": 3},
        "urgency":        {**_urgency_payload(),    "score": 5},
        "decision_power": {**_decision_payload(),   "score": 4},
        "engagement":     {**_engagement_payload(), "score": 4},
        "closing":        closing,
    }
    result = await compute_scores(state)

    expected = round((4 + 3 + 5 + 4 + 4 + 2) / 6, 1)
    assert result["score"] == expected


async def test_compute_scores_handles_missing_dimensions():
    state = {"job_id": "job-1"}
    result = await compute_scores(state)

    assert "score" in result
    assert "closing_probability" in result


# ── send_webhook (LangGraph node) ─────────────────────────────────────────


async def test_send_webhook_completed_posts_full_payload():
    state = {
        "job_id": "job-1",
        "webhook_url": "https://crm.example.com/api/webhooks/meet-analysis",
        "score": 4.0,
        "summary": "Reunião produtiva.",
        "next_step": "Enviar proposta",
        "positive_points": ["Dor clara"],
        "improvement_points": ["Não criou urgência"],
        "closing_probability": 72,
        "business": _business_payload(),
        "gaps": _gaps_payload(),
        "urgency": _urgency_payload(),
        "decision_power": _decision_payload(),
        "engagement": _engagement_payload(),
        "closing": _closing_payload(),
        "error": None,
    }

    mock_resp = MagicMock()
    mock_resp.status_code = 200

    with patch("httpx.AsyncClient") as mock_client_cls:
        mock_client = AsyncMock()
        mock_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_client.__aexit__ = AsyncMock(return_value=False)
        mock_client.post = AsyncMock(return_value=mock_resp)
        mock_client_cls.return_value = mock_client

        await send_webhook(state)

    mock_client.post.assert_called_once()
    _, kwargs = mock_client.post.call_args
    payload = kwargs["json"]

    assert payload["jobId"] == "job-1"
    assert payload["status"] == "completed"
    assert payload["score"] == 4.0
    assert "summary" in payload
    assert "nextStep" in payload
    assert "positivePoints" in payload
    assert "improvementPoints" in payload
    assert "closingProbability" in payload
    assert "diag" in payload
    diag = payload["diag"]
    for key in ("business", "gaps", "urgency", "decisionPower", "engagement", "closing"):
        assert key in diag, f"missing diag key: {key}"


async def test_send_webhook_error_state_posts_error_payload():
    state = {
        "job_id": "job-err",
        "webhook_url": "https://crm.example.com/api/webhooks/meet-analysis",
        "error": "analyze_dimensions_failed",
    }

    mock_resp = MagicMock()
    mock_resp.status_code = 200

    with patch("httpx.AsyncClient") as mock_client_cls:
        mock_client = AsyncMock()
        mock_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_client.__aexit__ = AsyncMock(return_value=False)
        mock_client.post = AsyncMock(return_value=mock_resp)
        mock_client_cls.return_value = mock_client

        await send_webhook(state)

    _, kwargs = mock_client.post.call_args
    payload = kwargs["json"]
    assert payload["status"] == "error"
    assert payload["error"] == "analyze_dimensions_failed"


async def test_send_webhook_no_url_skips_post():
    state = {"job_id": "job-1", "webhook_url": "", "error": None, "score": 4.0}

    with patch("httpx.AsyncClient") as mock_client_cls:
        mock_client = AsyncMock()
        mock_client_cls.return_value = mock_client
        await send_webhook(state)

    mock_client.__aenter__.assert_not_called()
