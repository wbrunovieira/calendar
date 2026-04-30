from __future__ import annotations

from unittest.mock import AsyncMock, patch

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)

_MOCK_BG = "app.agents.crm.meet_analysis.router._run_meet_analysis"

_VALID_PAYLOAD = {
    "jobId": "job-uuid-1234",
    "webhookUrl": "https://crm.example.com/api/webhooks/meet-analysis",
    "transcript": "00:00 [Vendedor]: Olá João, tudo bem?",
    "meetingDurationSeconds": 3600,
    "meetingDate": "2026-04-30T14:00:00Z",
    "meetingTitle": "Reunião diagnóstico - TechCo",
    "lead": {
        "id": "lead-uuid",
        "businessName": "TechCo",
        "description": "SaaS B2B para RH",
        "segment": "Tech",
        "city": "São Paulo",
    },
    "contact": {"name": "João Silva", "role": "CEO"},
    "activity": {"id": "act-uuid", "subject": "Reunião diagnóstico", "notes": "Lead quente"},
}


# ── Validation ───────────────────────────────────────────────────────────


def test_returns_422_missing_transcript():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "transcript"}
    resp = client.post("/agents/crm/meet-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_lead():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "lead"}
    resp = client.post("/agents/crm/meet-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_webhook_url():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "webhookUrl"}
    resp = client.post("/agents/crm/meet-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_job_id():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "jobId"}
    resp = client.post("/agents/crm/meet-analysis", json=payload)
    assert resp.status_code == 422


# ── Accepted ─────────────────────────────────────────────────────────────


def test_accepted_with_valid_payload():
    with patch(_MOCK_BG, new_callable=AsyncMock):
        resp = client.post("/agents/crm/meet-analysis", json=_VALID_PAYLOAD)

    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "accepted"
    assert data["jobId"] == "job-uuid-1234"


def test_accepted_with_optional_fields_missing():
    payload = {
        "jobId": "job-2",
        "webhookUrl": "https://crm.example.com/api/webhooks/meet-analysis",
        "transcript": "transcrição...",
        "lead": {"id": "lead-1", "businessName": "Empresa X"},
        "contact": {"name": "Maria"},
        "activity": {"id": "act-1"},
    }
    with patch(_MOCK_BG, new_callable=AsyncMock):
        resp = client.post("/agents/crm/meet-analysis", json=payload)

    assert resp.status_code == 200
    assert resp.json()["status"] == "accepted"


# ── Background task args ──────────────────────────────────────────────────


def test_background_receives_correct_job_id():
    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        resp = client.post("/agents/crm/meet-analysis", json=_VALID_PAYLOAD)

    data = resp.json()
    job_id = data["jobId"]

    mock.assert_called_once()
    call_args = mock.call_args
    assert call_args[0][0] == job_id


def test_background_receives_transcript():
    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        client.post("/agents/crm/meet-analysis", json=_VALID_PAYLOAD)

    call_args = mock.call_args[0]
    # transcript is the 3rd positional arg (job_id, webhook_url, transcript, ...)
    assert call_args[2] == _VALID_PAYLOAD["transcript"]


# ── Transcript normalization ──────────────────────────────────────────────


def test_transcript_json_array_normalized_to_plain_text():
    """JSON array transcript (Meet format) must be converted to plain text."""
    import json as _json

    segments = [
        {"start": 0,   "speakerName": "Vendedor", "text": "Olá João!"},
        {"start": 15,  "speakerName": "João",     "text": "Olá, tudo bem?"},
        {"start": 30,  "speakerName": "Vendedor", "text": "Ótimo, vamos começar."},
    ]
    payload = {**_VALID_PAYLOAD, "transcript": _json.dumps(segments)}

    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        client.post("/agents/crm/meet-analysis", json=payload)

    call_args = mock.call_args[0]
    normalized = call_args[2]

    assert "[Vendedor]" in normalized
    assert "[João]" in normalized
    assert "00:00" in normalized
    # should NOT start with "["
    assert not normalized.strip().startswith("[")


def test_plain_text_transcript_passes_through_unchanged():
    plain = "00:00 [Vendedor]: Oi!\n00:15 [João]: Oi!"
    payload = {**_VALID_PAYLOAD, "transcript": plain}

    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        client.post("/agents/crm/meet-analysis", json=payload)

    call_args = mock.call_args[0]
    assert call_args[2] == plain
