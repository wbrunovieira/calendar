from __future__ import annotations

import json
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from app.agents.crm.gatekeeper_analysis.nodes import (
    _analyze_alianca,
    _analyze_objecoes,
    _analyze_perguntas,
    _analyze_recepcao,
    _analyze_resultado,
    _analyze_tecnicas,
    analyze_dimensions,
    build_synthesis,
    compute_scores,
    send_webhook,
)

pytestmark = pytest.mark.asyncio(loop_scope="function")

_MODULE = "app.agents.crm.gatekeeper_analysis.nodes"

_LEAD = {"businessName": "Empresa X", "segment": "Varejo", "city": "São Paulo"}
_CONTACT = {"name": "Maria Santos", "role": "Recepcionista"}


def _mock_claude(text: str) -> MagicMock:
    block = SimpleNamespace(type="text", text=text)
    resp = MagicMock()
    resp.content = [block]
    resp.stop_reason = "end_turn"
    return resp


def _recepcao_payload():
    return {"text": "Abertura profissional sem frases de vendedor.", "score": 4}


def _alianca_payload():
    return {
        "text": "Usou o nome da recepcionista duas vezes.",
        "score": 3,
        "connectionMoments": ["Perguntou como ela estava", "Agradeceu a ajuda"],
    }


def _perguntas_payload():
    return {
        "text": "Fez 2 perguntas empoderadoras.",
        "score": 4,
        "questionsAsked": ["Você poderia me ajudar?", "Quem seria a pessoa certa?"],
    }


def _objecoes_payload():
    return {
        "text": "Recebeu 'manda email' e respondeu de forma razoável.",
        "score": 2,
        "objectionsReceived": ["Manda um email"],
        "responsesGiven": ["Respondeu que preferia alinhar pessoalmente"],
    }


def _resultado_payload():
    return {
        "text": "Conseguiu o nome do DM.",
        "score": 3,
        "outcome": "got_name",
        "obtained": ["Nome do DM: Carlos Mendes", "Cargo: Diretor Comercial"],
        "nextAttemptTip": "Ligar terça às 9h quando Carlos chega",
    }


def _tecnicas_payload():
    return {
        "text": "Usou aliança com GK e urgência suave.",
        "score": 3,
        "techniquesUsed": ["aliança com GK", "urgência suave"],
    }


def _synthesis_payload():
    return {
        "summary": "SDR ligou para a Empresa X e conseguiu o nome do decisor.",
        "positivePoints": ["Tom calmo e respeitoso", "Conseguiu o nome do DM"],
        "improvementPoints": ["Revelou demais ao responder 'do que se trata?'"],
    }


# ── Dimension helpers ─────────────────────────────────────────────────────


async def test_analyze_recepcao_returns_expected_fields():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_recepcao_payload()))):
        result = await _analyze_recepcao(_LEAD, _CONTACT, "transcript...", 180, "2026-05-01", None)
    assert "text" in result and "score" in result
    assert "error" not in result


async def test_analyze_alianca_returns_connection_moments():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_alianca_payload()))):
        result = await _analyze_alianca(_LEAD, _CONTACT, "transcript...", 180, None)
    assert "connectionMoments" in result
    assert isinstance(result["connectionMoments"], list)
    assert "error" not in result


async def test_analyze_perguntas_returns_questions_asked():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_perguntas_payload()))):
        result = await _analyze_perguntas(_LEAD, _CONTACT, "transcript...", 180, None)
    assert "questionsAsked" in result
    assert isinstance(result["questionsAsked"], list)
    assert "error" not in result


async def test_analyze_objecoes_returns_objections_and_responses():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_objecoes_payload()))):
        result = await _analyze_objecoes(_LEAD, _CONTACT, "transcript...", 180, None)
    assert "objectionsReceived" in result
    assert "responsesGiven" in result
    assert isinstance(result["objectionsReceived"], list)
    assert "error" not in result


async def test_analyze_resultado_returns_outcome_and_obtained():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_resultado_payload()))):
        result = await _analyze_resultado(_LEAD, _CONTACT, "transcript...", 180, None)
    assert "outcome" in result
    assert "obtained" in result
    assert "nextAttemptTip" in result
    assert isinstance(result["obtained"], list)
    assert "error" not in result


async def test_analyze_tecnicas_returns_techniques_used():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_tecnicas_payload()))):
        result = await _analyze_tecnicas(_LEAD, _CONTACT, "transcript...", 180, None)
    assert "techniquesUsed" in result
    assert isinstance(result["techniquesUsed"], list)
    assert "error" not in result


async def test_dimension_helper_returns_error_on_llm_exception():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock, side_effect=RuntimeError("timeout")):
        result = await _analyze_recepcao(_LEAD, _CONTACT, "t", 60, "2026-05-01", None)
    assert "error" in result


async def test_dimension_helper_returns_error_on_invalid_json():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude("not valid json")):
        result = await _analyze_objecoes(_LEAD, _CONTACT, "t", 60, None)
    assert "error" in result


# ── analyze_dimensions node ───────────────────────────────────────────────


async def test_analyze_dimensions_success_returns_all_six():
    payloads = [
        _recepcao_payload(), _alianca_payload(), _perguntas_payload(),
        _objecoes_payload(), _resultado_payload(), _tecnicas_payload(),
    ]
    call_count = 0

    async def _fake(system, user, model, max_tokens):
        nonlocal call_count
        resp = _mock_claude(json.dumps(payloads[call_count]))
        call_count += 1
        return resp

    state = {
        "job_id": "job-1", "lead": _LEAD, "contact": _CONTACT,
        "transcript": "transcript...", "call_duration_seconds": 180,
        "call_date": "2026-05-01", "_trace": None,
    }
    with patch(f"{_MODULE}._call_claude", side_effect=_fake):
        result = await analyze_dimensions(state)

    for dim in ("recepcao", "alianca", "perguntas", "objecoes", "resultado", "tecnicas"):
        assert dim in result, f"missing: {dim}"
    assert "error" not in result


async def test_analyze_dimensions_returns_error_when_one_fails():
    async def _fail(*a, **kw):
        raise RuntimeError("API error")

    state = {
        "job_id": "job-1", "lead": _LEAD, "contact": _CONTACT,
        "transcript": "t", "call_duration_seconds": 60,
        "call_date": "2026-05-01", "_trace": None,
    }
    with patch(f"{_MODULE}._call_claude", side_effect=_fail):
        result = await analyze_dimensions(state)
    assert "error" in result


# ── build_synthesis node ──────────────────────────────────────────────────


async def test_build_synthesis_returns_expected_structure():
    state = {
        "job_id": "job-1", "lead": _LEAD, "contact": _CONTACT,
        "recepcao": _recepcao_payload(), "alianca": _alianca_payload(),
        "perguntas": _perguntas_payload(), "objecoes": _objecoes_payload(),
        "resultado": _resultado_payload(), "tecnicas": _tecnicas_payload(),
        "_trace": None,
    }
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_synthesis_payload()))):
        result = await build_synthesis(state)

    assert "summary" in result
    assert "positive_points" in result
    assert "improvement_points" in result
    assert isinstance(result["positive_points"], list)
    assert "error" not in result


async def test_build_synthesis_returns_error_on_llm_failure():
    state = {
        "job_id": "job-1", "lead": _LEAD, "contact": _CONTACT,
        "recepcao": _recepcao_payload(), "alianca": _alianca_payload(),
        "perguntas": _perguntas_payload(), "objecoes": _objecoes_payload(),
        "resultado": _resultado_payload(), "tecnicas": _tecnicas_payload(),
        "_trace": None,
    }
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock, side_effect=RuntimeError("err")):
        result = await build_synthesis(state)
    assert "error" in result


async def test_build_synthesis_does_not_receive_transcript():
    captured = []

    async def _capture(system, user, model, max_tokens):
        captured.append(user)
        return _mock_claude(json.dumps(_synthesis_payload()))

    state = {
        "job_id": "job-1", "lead": _LEAD, "contact": _CONTACT,
        "transcript": "SHOULD_NOT_BE_IN_SYNTHESIS",
        "recepcao": _recepcao_payload(), "alianca": _alianca_payload(),
        "perguntas": _perguntas_payload(), "objecoes": _objecoes_payload(),
        "resultado": _resultado_payload(), "tecnicas": _tecnicas_payload(),
        "_trace": None,
    }
    with patch(f"{_MODULE}._call_claude", side_effect=_capture):
        await build_synthesis(state)

    assert captured, "no LLM call made"
    assert "SHOULD_NOT_BE_IN_SYNTHESIS" not in captured[0]


# ── compute_scores node ───────────────────────────────────────────────────


async def test_compute_scores_averages_six_dimensions():
    state = {
        "job_id": "job-1",
        "recepcao":  {**_recepcao_payload(),  "score": 4},
        "alianca":   {**_alianca_payload(),   "score": 3},
        "perguntas": {**_perguntas_payload(), "score": 4},
        "objecoes":  {**_objecoes_payload(),  "score": 2},
        "resultado": {**_resultado_payload(), "score": 3},
        "tecnicas":  {**_tecnicas_payload(),  "score": 3},
    }
    result = await compute_scores(state)
    expected = round((4 + 3 + 4 + 2 + 3 + 3) / 6, 1)
    assert result["score"] == expected


async def test_compute_scores_handles_missing_dimensions():
    result = await compute_scores({"job_id": "job-1"})
    assert "score" in result
    assert isinstance(result["score"], float)


# ── send_webhook node ─────────────────────────────────────────────────────


async def test_send_webhook_completed_posts_full_payload():
    state = {
        "job_id": "job-1",
        "webhook_url": "https://crm.example.com/webhooks/gatekeeper-analysis",
        "score": 3.2,
        "summary": "Ligação GK resumo.",
        "positive_points": ["Tom calmo"],
        "improvement_points": ["Revelou demais"],
        "recepcao": _recepcao_payload(), "alianca": _alianca_payload(),
        "perguntas": _perguntas_payload(), "objecoes": _objecoes_payload(),
        "resultado": _resultado_payload(), "tecnicas": _tecnicas_payload(),
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
    assert payload["jobId"] == "job-1"
    assert payload["status"] == "completed"
    assert payload["score"] == 3.2
    assert "summary" in payload
    assert "raport" in payload
    for dim in ("recepcao", "alianca", "perguntas", "objecoes", "resultado", "tecnicas"):
        assert dim in payload["raport"], f"missing raport.{dim}"
    assert "positivePoints" in payload
    assert "improvementPoints" in payload


async def test_send_webhook_error_state_posts_error_payload():
    state = {
        "job_id": "job-err",
        "webhook_url": "https://crm.example.com/webhooks/gatekeeper-analysis",
        "error": "analyze_dimensions_failed",
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
    assert payload["error"] == "analyze_dimensions_failed"


async def test_send_webhook_no_url_skips():
    state = {"job_id": "job-1", "webhook_url": "", "error": None, "score": 3.0}
    with patch("httpx.AsyncClient") as mock_cls:
        mc = AsyncMock()
        mock_cls.return_value = mc
        await send_webhook(state)
    mc.__aenter__.assert_not_called()
