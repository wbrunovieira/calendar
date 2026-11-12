from __future__ import annotations

from unittest.mock import patch

import httpx
import pytest
from fastapi.testclient import TestClient

from app.config import settings
from app.main import app

TOKEN = "s3cret-token"
PROTECTED = "/agents/does-not-exist"  # any non-public path: auth runs before routing

client = TestClient(app)


@pytest.fixture
def token(monkeypatch):
    monkeypatch.setattr(settings, "agents_api_token", TOKEN)


# ── Open until configured ───────────────────────────────────


@pytest.mark.parametrize("configured", ["", "   "])
def test_open_when_token_is_not_configured(monkeypatch, configured):
    monkeypatch.setattr(settings, "agents_api_token", configured)
    assert client.get(PROTECTED).status_code == 404


# ── Enforced ────────────────────────────────────────────────


@pytest.mark.parametrize(
    "headers",
    [
        {},
        {"Authorization": "Bearer nope"},
        {"Authorization": "Bearer s3cret"},
        {"Authorization": f"Basic {TOKEN}"},
        {"Authorization": "Bearer "},
        {"X-API-Key": "nope"},
    ],
    ids=["missing", "wrong", "prefix", "basic", "empty", "wrong-api-key"],
)
def test_rejects_missing_or_invalid_token(token, headers):
    resp = client.get(PROTECTED, headers=headers)
    assert resp.status_code == 401
    assert resp.json() == {"detail": "Invalid or missing API token"}
    assert resp.headers["www-authenticate"].startswith("Bearer")


@pytest.mark.parametrize(
    "headers",
    [{"Authorization": f"Bearer {TOKEN}"}, {"X-API-Key": TOKEN}],
    ids=["bearer", "api-key"],
)
def test_accepts_valid_token(token, headers):
    assert client.get(PROTECTED, headers=headers).status_code == 404


def test_protects_real_agent_routes(token):
    resp = client.post("/agents/crm/lead-research", json={"query": "x", "icp_id": "i"})
    assert resp.status_code == 401


def test_health_check_stays_public(token):
    # The health route reads state set at startup, so run the lifespan.
    with TestClient(app) as started:
        assert started.get("/health").status_code == 200


def test_docs_are_not_public(token):
    assert client.get("/docs").status_code == 401
    assert client.get("/openapi.json").status_code == 401


def test_cors_preflight_passes_without_token(token):
    resp = client.options(
        PROTECTED,
        headers={
            "Origin": "https://crm.wbdigitalsolutions.com",
            "Access-Control-Request-Method": "POST",
            "Access-Control-Request-Headers": "authorization",
        },
    )
    assert resp.status_code == 200
    assert resp.headers["access-control-allow-origin"] == "https://crm.wbdigitalsolutions.com"


def test_rejection_carries_cors_headers(token):
    resp = client.get(PROTECTED, headers={"Origin": "https://crm.wbdigitalsolutions.com"})
    assert resp.status_code == 401
    assert resp.headers["access-control-allow-origin"] == "https://crm.wbdigitalsolutions.com"


# ── Outbound: calendar-finances client sends its token ──────


class _Recorder:
    def __init__(self):
        self.headers: httpx.Headers | None = None

    def handler(self, request: httpx.Request) -> httpx.Response:
        self.headers = request.headers
        return httpx.Response(200, json={"data": []})


def _patched_client(recorder: _Recorder):
    real = httpx.AsyncClient

    def factory(*args, **kwargs):
        kwargs["transport"] = httpx.MockTransport(recorder.handler)
        return real(*args, **kwargs)

    return patch("app.clients.finances.httpx.AsyncClient", side_effect=factory)


@pytest.mark.asyncio
async def test_finances_client_sends_token_when_configured(monkeypatch):
    from app.clients import finances

    monkeypatch.setattr(settings, "finances_api_token", "fin-tok")
    rec = _Recorder()
    with _patched_client(rec):
        await finances.get_bank_accounts("p1")
    assert rec.headers["authorization"] == "Bearer fin-tok"


@pytest.mark.asyncio
async def test_finances_client_sends_no_auth_when_unconfigured(monkeypatch):
    from app.clients import finances

    monkeypatch.setattr(settings, "finances_api_token", "")
    rec = _Recorder()
    with _patched_client(rec):
        await finances.get_categories("p1")
    assert "authorization" not in rec.headers
