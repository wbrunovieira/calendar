from __future__ import annotations

from unittest.mock import AsyncMock, patch

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)

_MOCK_BG = "app.agents.crm.gatekeeper_analysis.router._run_gatekeeper_analysis"

_VALID_PAYLOAD = {
    "jobId": "gk-job-1234",
    "webhookUrl": "https://crm.example.com/webhooks/gatekeeper-analysis",
    "transcript": "00:00 [SDR]: Bom dia, meu nome é Bruno.",
    "callDurationSeconds": 180,
    "callDate": "2026-05-01",
    "lead": {
        "id": "lead-uuid",
        "businessName": "Empresa X",
        "segment": "Varejo",
        "city": "São Paulo",
    },
    "contact": {"name": "Maria Santos", "role": "Recepcionista"},
    "activity": {"id": "act-uuid", "subject": "Ligação GK", "notes": ""},
}


# ── Validation ────────────────────────────────────────────────────────────


def test_returns_422_missing_transcript():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "transcript"}
    resp = client.post("/agents/crm/gatekeeper-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_lead():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "lead"}
    resp = client.post("/agents/crm/gatekeeper-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_webhook_url():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "webhookUrl"}
    resp = client.post("/agents/crm/gatekeeper-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_job_id():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "jobId"}
    resp = client.post("/agents/crm/gatekeeper-analysis", json=payload)
    assert resp.status_code == 422


# ── Accepted ──────────────────────────────────────────────────────────────


def test_accepted_with_valid_payload():
    with patch(_MOCK_BG, new_callable=AsyncMock):
        resp = client.post("/agents/crm/gatekeeper-analysis", json=_VALID_PAYLOAD)

    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "accepted"
    assert data["jobId"] == "gk-job-1234"


def test_accepted_without_optional_activity():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "activity"}
    with patch(_MOCK_BG, new_callable=AsyncMock):
        resp = client.post("/agents/crm/gatekeeper-analysis", json=payload)

    assert resp.status_code == 200
    assert resp.json()["status"] == "accepted"


# ── Background task args ──────────────────────────────────────────────────


def test_background_receives_correct_job_id():
    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        resp = client.post("/agents/crm/gatekeeper-analysis", json=_VALID_PAYLOAD)

    mock.assert_called_once()
    call_args = mock.call_args[0]
    assert call_args[0] == resp.json()["jobId"]


def test_background_receives_transcript():
    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        client.post("/agents/crm/gatekeeper-analysis", json=_VALID_PAYLOAD)

    call_args = mock.call_args[0]
    # (job_id, webhook_url, transcript, ...)
    assert call_args[2] == _VALID_PAYLOAD["transcript"]
