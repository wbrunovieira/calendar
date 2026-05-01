from __future__ import annotations

import json
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from app.agents.crm.transfer_analysis.nodes import (
    analyze_gk_phase,
    analyze_spiced_phase,
    segment_transcript,
    send_both_errors,
)

pytestmark = pytest.mark.asyncio(loop_scope="function")

_MODULE = "app.agents.crm.transfer_analysis.nodes"

_LEAD = {"businessName": "Empresa X", "segment": "Varejo", "city": "São Paulo"}
_CONTACT = {"name": "Maria Santos", "role": "Recepcionista"}

_GK_STATE = {
    "gk_job_id": "gk-job-1",
    "spiced_job_id": "sp-job-1",
    "gk_webhook_url": "https://crm.example.com/webhooks/gatekeeper-analysis",
    "spiced_webhook_url": "https://crm.example.com/webhooks/call-analysis",
    "transcript": "SDR: Bom dia! GK: Bom dia. SDR: Posso falar com o diretor? GK: Claro, um momento. Decisor: Alô?",
    "call_duration_seconds": 180,
    "call_date": "2026-05-01",
    "lead": _LEAD,
    "contact": _CONTACT,
    "activity": {},
    "_trace": None,
}


def _mock_claude(text: str) -> MagicMock:
    block = SimpleNamespace(type="text", text=text)
    resp = MagicMock()
    resp.content = [block]
    resp.stop_reason = "end_turn"
    return resp


def _segment_payload():
    return {
        "gkTranscript": "SDR: Bom dia! GK: Bom dia. SDR: Posso falar com o diretor? GK: Claro, um momento.",
        "decisorTranscript": "Decisor: Alô?",
        "transitionNote": "Recepcionista Maria transferiu ao diretor comercial",
    }


def _dims_payload():
    return {
        "recepcao":  {"text": "Boa abertura.", "score": 4},
        "alianca":   {"text": "Rapport ok.", "score": 3, "connectionMoments": []},
        "perguntas": {"text": "Pergunta ok.", "score": 3, "questionsAsked": []},
        "objecoes":  {"text": "Sem objeções.", "score": 5, "objectionsReceived": [], "responsesGiven": []},
        "resultado": {"text": "Transferido.", "score": 5, "outcome": "got_transfer", "obtained": [], "nextAttemptTip": None},
        "tecnicas":  {"text": "Aliança com GK.", "score": 3, "techniquesUsed": ["aliança com GK"]},
    }


def _synth_payload():
    return {
        "summary": "SDR transferido ao decisor.",
        "positive_points": ["Rapport com GK"],
        "improvement_points": ["Usar nome do porteiro"],
    }


def _scores_payload():
    return {"score": 3.8}


def _spiced_payload():
    return {
        "spiced": {
            "situation":     {"score": 7.0, "text": "ok"},
            "pain":          {"score": 6.5, "text": "ok"},
            "impact":        {"score": 5.0, "text": "ok"},
            "criticalEvent": {"score": 4.0, "text": "ok"},
            "evidence":      {"score": 3.0, "text": "ok"},
        },
        "micro_pactos": [
            {"id": i, "label": f"pacto {i}", "spicedDimension": "S", "achieved": i <= 4, "notes": ""}
            for i in range(1, 8)
        ],
        "scheduling_techniques": {
            "gatilhoDor":            {"used": True,  "notes": "usou"},
            "escolhaAlternativa":    {"used": True,  "notes": "usou"},
            "compromissoVerbalShow": {"used": False, "notes": ""},
            "compromissoEmergencia": {"used": False, "notes": ""},
        },
    }


def _analysis_payload():
    return {
        "summary": "Boa ligação com decisor.",
        "micro_analysis": [{"timestamp": "01:30", "text": "dor identificada", "type": "positive"}],
        "positive_points": ["Explorou dor bem"],
        "improvement_points": ["Faltou urgência"],
    }


def _spiced_scores_payload():
    return {"score": 5.1, "no_show_risk": "MÉDIO", "no_show_risk_text": "Micro-pactos 4/7..."}


# ── segment_transcript ────────────────────────────────────────────────────


async def test_segment_transcript_returns_both_transcripts():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude(json.dumps(_segment_payload()))):
        result = await segment_transcript(_GK_STATE)

    assert "gk_transcript" in result
    assert "decisor_transcript" in result
    assert "transition_note" in result
    assert "error" not in result
    assert len(result["gk_transcript"]) > 0


async def test_segment_transcript_returns_error_on_llm_failure():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock, side_effect=RuntimeError("timeout")):
        result = await segment_transcript(_GK_STATE)
    assert "error" in result


async def test_segment_transcript_returns_error_on_invalid_json():
    with patch(f"{_MODULE}._call_claude", new_callable=AsyncMock,
               return_value=_mock_claude("not valid json")):
        result = await segment_transcript(_GK_STATE)
    assert "error" in result


# ── send_both_errors ──────────────────────────────────────────────────────


async def test_send_both_errors_posts_to_both_webhooks():
    state = {
        "gk_job_id": "gk-1",
        "spiced_job_id": "sp-1",
        "gk_webhook_url": "https://crm.example.com/webhooks/gatekeeper-analysis",
        "spiced_webhook_url": "https://crm.example.com/webhooks/call-analysis",
        "error": "segment_transcript_failed",
    }
    posts = []

    async def _fake_post(url, payload):
        posts.append((url, payload))

    with patch(f"{_MODULE}._post_json", side_effect=_fake_post):
        await send_both_errors(state)

    assert len(posts) == 2
    urls = {p[0] for p in posts}
    assert "https://crm.example.com/webhooks/gatekeeper-analysis" in urls
    assert "https://crm.example.com/webhooks/call-analysis" in urls

    for url, payload in posts:
        assert payload["status"] == "error"
        assert "jobId" in payload


async def test_send_both_errors_no_url_does_not_crash():
    state = {
        "gk_job_id": "gk-1",
        "spiced_job_id": "sp-1",
        "gk_webhook_url": "",
        "spiced_webhook_url": "",
        "error": "segment_transcript_failed",
    }
    with patch(f"{_MODULE}._post_json", new_callable=AsyncMock):
        await send_both_errors(state)  # must not raise


# ── analyze_gk_phase ──────────────────────────────────────────────────────


async def test_analyze_gk_phase_posts_completed_webhook():
    state = {**_GK_STATE, "gk_transcript": "SDR: Bom dia!"}
    mock_resp = MagicMock(status_code=200)
    posts = []

    async def _fake_post(url, payload):
        posts.append((url, payload))

    with patch(f"{_MODULE}._gk_analyze_dimensions", new_callable=AsyncMock, return_value=_dims_payload()), \
         patch(f"{_MODULE}._gk_build_synthesis", new_callable=AsyncMock, return_value=_synth_payload()), \
         patch(f"{_MODULE}._gk_compute_scores", new_callable=AsyncMock, return_value=_scores_payload()), \
         patch(f"{_MODULE}._post_json", side_effect=_fake_post):
        await analyze_gk_phase(state)

    assert len(posts) == 1
    url, payload = posts[0]
    assert url == state["gk_webhook_url"]
    assert payload["jobId"] == "gk-job-1"
    assert payload["status"] == "completed"
    assert payload["score"] == 3.8
    assert "raport" in payload
    for dim in ("recepcao", "alianca", "perguntas", "objecoes", "resultado", "tecnicas"):
        assert dim in payload["raport"], f"missing raport.{dim}"
    assert "positivePoints" in payload
    assert "improvementPoints" in payload


async def test_analyze_gk_phase_posts_error_on_dimensions_failure():
    state = {**_GK_STATE, "gk_transcript": "SDR: Bom dia!"}
    posts = []

    async def _fake_post(url, payload):
        posts.append((url, payload))

    with patch(f"{_MODULE}._gk_analyze_dimensions", new_callable=AsyncMock,
               return_value={"error": "analyze_dimensions_failed"}), \
         patch(f"{_MODULE}._post_json", side_effect=_fake_post):
        await analyze_gk_phase(state)

    assert len(posts) == 1
    assert posts[0][1]["status"] == "error"


async def test_analyze_gk_phase_posts_error_on_synthesis_failure():
    state = {**_GK_STATE, "gk_transcript": "SDR: Bom dia!"}
    posts = []

    async def _fake_post(url, payload):
        posts.append((url, payload))

    with patch(f"{_MODULE}._gk_analyze_dimensions", new_callable=AsyncMock, return_value=_dims_payload()), \
         patch(f"{_MODULE}._gk_build_synthesis", new_callable=AsyncMock,
               return_value={"error": "build_synthesis_failed"}), \
         patch(f"{_MODULE}._post_json", side_effect=_fake_post):
        await analyze_gk_phase(state)

    assert len(posts) == 1
    assert posts[0][1]["status"] == "error"


async def test_analyze_gk_phase_posts_error_on_exception():
    state = {**_GK_STATE, "gk_transcript": "SDR: Bom dia!"}
    posts = []

    async def _fake_post(url, payload):
        posts.append((url, payload))

    with patch(f"{_MODULE}._gk_analyze_dimensions", new_callable=AsyncMock,
               side_effect=RuntimeError("boom")), \
         patch(f"{_MODULE}._post_json", side_effect=_fake_post):
        await analyze_gk_phase(state)

    assert len(posts) == 1
    assert posts[0][1]["status"] == "error"


async def test_analyze_gk_phase_posts_error_when_no_gk_transcript():
    state = {**_GK_STATE, "gk_transcript": ""}
    posts = []

    async def _fake_post(url, payload):
        posts.append((url, payload))

    with patch(f"{_MODULE}._post_json", side_effect=_fake_post):
        await analyze_gk_phase(state)

    assert len(posts) == 1
    assert posts[0][1]["status"] == "error"


async def test_analyze_gk_phase_always_returns_empty_dict():
    state = {**_GK_STATE, "gk_transcript": "SDR: Bom dia!"}

    with patch(f"{_MODULE}._gk_analyze_dimensions", new_callable=AsyncMock, return_value=_dims_payload()), \
         patch(f"{_MODULE}._gk_build_synthesis", new_callable=AsyncMock, return_value=_synth_payload()), \
         patch(f"{_MODULE}._gk_compute_scores", new_callable=AsyncMock, return_value=_scores_payload()), \
         patch(f"{_MODULE}._post_json", new_callable=AsyncMock):
        result = await analyze_gk_phase(state)

    assert result == {}


# ── analyze_spiced_phase ──────────────────────────────────────────────────


async def test_analyze_spiced_phase_posts_completed_webhook():
    state = {
        **_GK_STATE,
        "decisor_transcript": "Decisor: Alô? SDR: Boa tarde Carlos!",
        "transition_note": "Maria transferiu ao Carlos Mendes",
    }
    posts = []

    async def _fake_post(url, payload):
        posts.append((url, payload))

    with patch(f"{_MODULE}._call_analyze_spiced", new_callable=AsyncMock, return_value=_spiced_payload()), \
         patch(f"{_MODULE}._call_build_analysis", new_callable=AsyncMock, return_value=_analysis_payload()), \
         patch(f"{_MODULE}._call_compute_scores", new_callable=AsyncMock, return_value=_spiced_scores_payload()), \
         patch(f"{_MODULE}._post_json", side_effect=_fake_post):
        await analyze_spiced_phase(state)

    assert len(posts) == 1
    url, payload = posts[0]
    assert url == state["spiced_webhook_url"]
    assert payload["jobId"] == "sp-job-1"
    assert payload["status"] == "completed"
    assert payload["score"] == 5.1
    assert payload["noShowRisk"] == "MÉDIO"
    assert "spiced" in payload
    assert "microPactos" in payload
    assert "schedulingTechniques" in payload
    assert "microAnalysis" in payload
    assert "positivePoints" in payload
    assert "improvementPoints" in payload


async def test_analyze_spiced_phase_prepends_transition_note():
    state = {
        **_GK_STATE,
        "decisor_transcript": "Decisor: Alô?",
        "transition_note": "Maria transferiu ao Carlos",
    }
    captured_states = []

    async def _capture_spiced(s):
        captured_states.append(s)
        return _spiced_payload()

    with patch(f"{_MODULE}._call_analyze_spiced", side_effect=_capture_spiced), \
         patch(f"{_MODULE}._call_build_analysis", new_callable=AsyncMock, return_value=_analysis_payload()), \
         patch(f"{_MODULE}._call_compute_scores", new_callable=AsyncMock, return_value=_spiced_scores_payload()), \
         patch(f"{_MODULE}._post_json", new_callable=AsyncMock):
        await analyze_spiced_phase(state)

    assert captured_states, "spiced analysis not called"
    transcript_used = captured_states[0]["transcript"]
    assert "Maria transferiu ao Carlos" in transcript_used
    assert "Decisor: Alô?" in transcript_used


async def test_analyze_spiced_phase_posts_error_on_spiced_failure():
    state = {
        **_GK_STATE,
        "decisor_transcript": "Decisor: Alô?",
        "transition_note": "",
    }
    posts = []

    async def _fake_post(url, payload):
        posts.append((url, payload))

    with patch(f"{_MODULE}._call_analyze_spiced", new_callable=AsyncMock,
               return_value={"error": "analyze_spiced_failed"}), \
         patch(f"{_MODULE}._post_json", side_effect=_fake_post):
        await analyze_spiced_phase(state)

    assert len(posts) == 1
    assert posts[0][1]["status"] == "error"


async def test_analyze_spiced_phase_posts_error_when_no_decisor_transcript():
    state = {**_GK_STATE, "decisor_transcript": "", "transition_note": ""}
    posts = []

    async def _fake_post(url, payload):
        posts.append((url, payload))

    with patch(f"{_MODULE}._post_json", side_effect=_fake_post):
        await analyze_spiced_phase(state)

    assert len(posts) == 1
    assert posts[0][1]["status"] == "error"


async def test_analyze_spiced_phase_always_returns_empty_dict():
    state = {
        **_GK_STATE,
        "decisor_transcript": "Decisor: Alô?",
        "transition_note": "",
    }
    with patch(f"{_MODULE}._call_analyze_spiced", new_callable=AsyncMock, return_value=_spiced_payload()), \
         patch(f"{_MODULE}._call_build_analysis", new_callable=AsyncMock, return_value=_analysis_payload()), \
         patch(f"{_MODULE}._call_compute_scores", new_callable=AsyncMock, return_value=_spiced_scores_payload()), \
         patch(f"{_MODULE}._post_json", new_callable=AsyncMock):
        result = await analyze_spiced_phase(state)

    assert result == {}
