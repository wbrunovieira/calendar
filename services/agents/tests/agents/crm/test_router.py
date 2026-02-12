from __future__ import annotations

from unittest.mock import AsyncMock, patch

import pytest
from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)

# Mock _run_research to prevent actual API calls in background tasks
_mock_research = "app.agents.crm.router._run_research"


# ── Validation ──────────────────────────────────────────────


def test_returns_error_when_no_query_direct_format():
    resp = client.post("/agents/crm/lead-research", json={"icp_id": "icp-1"})
    data = resp.json()
    assert resp.status_code == 200
    assert data["status"] == "error"
    assert data["error"] == "missing_query"


def test_returns_error_when_no_icp_id_direct_format():
    resp = client.post("/agents/crm/lead-research", json={"query": "test"})
    data = resp.json()
    assert resp.status_code == 200
    assert data["status"] == "error"
    assert data["error"] == "missing_icp_id"


def test_returns_error_when_count_exceeds_max():
    resp = client.post("/agents/crm/lead-research", json={
        "query": "test", "icp_id": "icp-1", "count": 10
    })
    assert resp.status_code == 422


def test_returns_error_when_count_zero():
    resp = client.post("/agents/crm/lead-research", json={
        "query": "test", "icp_id": "icp-1", "count": 0
    })
    assert resp.status_code == 422


# ── Accepted (async) ───────────────────────────────────────


def test_accepted_direct_format():
    with patch(_mock_research, new_callable=AsyncMock):
        resp = client.post("/agents/crm/lead-research", json={
            "query": "SaaS companies", "icp_id": "icp-1", "count": 1
        })
    data = resp.json()
    assert resp.status_code == 200
    assert data["status"] == "accepted"
    assert "Research started" in data["reply"]


def test_accepted_crm_format():
    with patch(_mock_research, new_callable=AsyncMock):
        resp = client.post("/agents/crm/lead-research", json={
            "icp": {"id": "icp-1", "name": "SaaS B2B", "content": "Target SaaS companies"},
            "searchParams": {"searchTerm": "Hotmart", "country": "Brasil", "quantity": 2, "quality": "cold"},
        })
    data = resp.json()
    assert resp.status_code == 200
    assert data["status"] == "accepted"
    assert "Hotmart" in data["reply"]
    assert "2 lead(s)" in data["reply"]


def test_accepted_crm_format_defaults():
    with patch(_mock_research, new_callable=AsyncMock):
        resp = client.post("/agents/crm/lead-research", json={
            "icp": {"id": "icp-1"},
            "searchParams": {"searchTerm": "edtech"},
        })
    data = resp.json()
    assert resp.status_code == 200
    assert data["status"] == "accepted"


def test_crm_format_quantity_exceeds_max():
    resp = client.post("/agents/crm/lead-research", json={
        "icp": {"id": "icp-1"},
        "searchParams": {"searchTerm": "test", "quantity": 10},
    })
    assert resp.status_code == 422
