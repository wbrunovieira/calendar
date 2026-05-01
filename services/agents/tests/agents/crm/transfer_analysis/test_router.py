from __future__ import annotations

from unittest.mock import AsyncMock, patch

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)

_MOCK_BG = "app.agents.crm.transfer_analysis.router._run_transfer_analysis"

_VALID_PAYLOAD = {
    "gkJobId": "gk-job-uuid-1234",
    "spicedJobId": "sp-job-uuid-1234",
    "gkWebhookUrl": "https://crm.example.com/webhooks/gatekeeper-analysis",
    "spicedWebhookUrl": "https://crm.example.com/webhooks/call-analysis",
    "transcript": "SDR: Bom dia! GK: Bom dia. SDR: Posso falar com o diretor? GK: Claro. Decisor: Alô?",
    "callDurationSeconds": 312,
    "callDate": "2026-05-01T10:00:00Z",
    "lead": {
        "id": "lead-uuid",
        "businessName": "Empresa X LTDA",
        "segment": "Varejo",
        "city": "São Paulo",
    },
    "contact": {"name": "Maria Santos", "role": "Recepcionista"},
    "activity": {"id": "act-uuid", "subject": "Ligação Empresa X"},
}


# ── Validation ────────────────────────────────────────────────────────────


def test_returns_422_missing_gk_job_id():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "gkJobId"}
    resp = client.post("/agents/crm/transfer-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_spiced_job_id():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "spicedJobId"}
    resp = client.post("/agents/crm/transfer-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_gk_webhook_url():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "gkWebhookUrl"}
    resp = client.post("/agents/crm/transfer-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_spiced_webhook_url():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "spicedWebhookUrl"}
    resp = client.post("/agents/crm/transfer-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_transcript():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "transcript"}
    resp = client.post("/agents/crm/transfer-analysis", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_lead():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "lead"}
    resp = client.post("/agents/crm/transfer-analysis", json=payload)
    assert resp.status_code == 422


# ── Accepted ──────────────────────────────────────────────────────────────


def test_accepted_with_valid_payload():
    with patch(_MOCK_BG, new_callable=AsyncMock):
        resp = client.post("/agents/crm/transfer-analysis", json=_VALID_PAYLOAD)

    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "accepted"
    assert data["gkJobId"] == "gk-job-uuid-1234"
    assert data["spicedJobId"] == "sp-job-uuid-1234"


def test_accepted_without_optional_activity():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "activity"}
    with patch(_MOCK_BG, new_callable=AsyncMock):
        resp = client.post("/agents/crm/transfer-analysis", json=payload)

    assert resp.status_code == 200
    assert resp.json()["status"] == "accepted"


def test_accepted_without_optional_call_date():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "callDate"}
    with patch(_MOCK_BG, new_callable=AsyncMock):
        resp = client.post("/agents/crm/transfer-analysis", json=payload)

    assert resp.status_code == 200
    assert resp.json()["status"] == "accepted"


# ── Background task args ──────────────────────────────────────────────────


def test_background_receives_gk_and_spiced_job_ids():
    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        client.post("/agents/crm/transfer-analysis", json=_VALID_PAYLOAD)

    mock.assert_called_once()
    call_args = mock.call_args[0]
    # (gk_job_id, spiced_job_id, gk_webhook_url, spiced_webhook_url, transcript, ...)
    assert call_args[0] == "gk-job-uuid-1234"
    assert call_args[1] == "sp-job-uuid-1234"


def test_background_receives_both_webhook_urls():
    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        client.post("/agents/crm/transfer-analysis", json=_VALID_PAYLOAD)

    call_args = mock.call_args[0]
    assert call_args[2] == _VALID_PAYLOAD["gkWebhookUrl"]
    assert call_args[3] == _VALID_PAYLOAD["spicedWebhookUrl"]


def test_background_receives_transcript():
    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        client.post("/agents/crm/transfer-analysis", json=_VALID_PAYLOAD)

    call_args = mock.call_args[0]
    assert call_args[4] == _VALID_PAYLOAD["transcript"]
