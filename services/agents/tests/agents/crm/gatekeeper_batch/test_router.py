from __future__ import annotations

from unittest.mock import AsyncMock, patch

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)

_MOCK_BG = "app.agents.crm.gatekeeper_batch.router._run_gatekeeper_batch"

_ANALYSIS_1 = {
    "jobId": "gk-job-1",
    "score": 4.0,
    "raport": {
        "recepcao": {"score": 4, "text": "ok"},
        "alianca": {"score": 3, "text": "ok"},
        "perguntas": {"score": 4, "text": "ok"},
        "objecoes": {"score": 2, "text": "ok"},
        "resultado": {"score": 4, "text": "ok", "outcome": "got_name"},
        "tecnicas": {"score": 4, "text": "ok"},
    },
}

_VALID_PAYLOAD = {
    "batchJobId": "batch-uuid-1234",
    "webhookUrl": "https://crm.example.com/webhooks/gatekeeper-batch",
    "analyses": [_ANALYSIS_1],
    "historicalSummaries": [],
}


# ── Validation ────────────────────────────────────────────────────────────


def test_returns_422_missing_batch_job_id():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "batchJobId"}
    resp = client.post("/agents/crm/gatekeeper-batch", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_webhook_url():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "webhookUrl"}
    resp = client.post("/agents/crm/gatekeeper-batch", json=payload)
    assert resp.status_code == 422


def test_returns_422_missing_analyses():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "analyses"}
    resp = client.post("/agents/crm/gatekeeper-batch", json=payload)
    assert resp.status_code == 422


# ── Accepted ──────────────────────────────────────────────────────────────


def test_accepted_with_valid_payload():
    with patch(_MOCK_BG, new_callable=AsyncMock):
        resp = client.post("/agents/crm/gatekeeper-batch", json=_VALID_PAYLOAD)

    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "accepted"
    assert data["batchJobId"] == "batch-uuid-1234"


def test_accepted_without_historical_summaries():
    payload = {k: v for k, v in _VALID_PAYLOAD.items() if k != "historicalSummaries"}
    with patch(_MOCK_BG, new_callable=AsyncMock):
        resp = client.post("/agents/crm/gatekeeper-batch", json=payload)

    assert resp.status_code == 200
    assert resp.json()["status"] == "accepted"


def test_accepted_with_multiple_analyses():
    payload = {**_VALID_PAYLOAD, "analyses": [_ANALYSIS_1, _ANALYSIS_1]}
    with patch(_MOCK_BG, new_callable=AsyncMock):
        resp = client.post("/agents/crm/gatekeeper-batch", json=payload)

    assert resp.status_code == 200
    assert resp.json()["status"] == "accepted"


# ── Background task args ──────────────────────────────────────────────────


def test_background_receives_correct_batch_job_id():
    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        resp = client.post("/agents/crm/gatekeeper-batch", json=_VALID_PAYLOAD)

    mock.assert_called_once()
    call_args = mock.call_args[0]
    assert call_args[0] == resp.json()["batchJobId"]


def test_background_receives_analyses():
    mock = AsyncMock()
    with patch(_MOCK_BG, mock):
        client.post("/agents/crm/gatekeeper-batch", json=_VALID_PAYLOAD)

    call_args = mock.call_args[0]
    # (batch_job_id, webhook_url, analyses, historical_summaries, trace_id)
    assert len(call_args[2]) == 1
    assert call_args[2][0]["jobId"] == "gk-job-1"
