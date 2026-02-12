from __future__ import annotations

from unittest.mock import AsyncMock, patch

import pytest
from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)


def test_returns_error_when_icp_id_missing():
    resp = client.post("/agents/crm/lead-research", json={"query": "test"})
    assert resp.status_code == 422


def test_returns_error_when_query_missing():
    resp = client.post("/agents/crm/lead-research", json={"icp_id": "icp-1"})
    assert resp.status_code == 422


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


def test_returns_error_when_icp_not_found():
    mock_result = {
        "error": "icp_not_found",
        "reply": "ICP 'bad-id' not found",
    }
    with patch("app.agents.crm.router._get_graph") as mock_graph:
        mock_graph.return_value.ainvoke = AsyncMock(return_value=mock_result)
        resp = client.post("/agents/crm/lead-research", json={
            "query": "test", "icp_id": "bad-id"
        })
    data = resp.json()
    assert resp.status_code == 200
    assert data["status"] == "error"
    assert data["error"] == "icp_not_found"


def test_successful_lead_research():
    mock_result = {
        "created_leads": [{
            "lead": {"id": "lead-1", "businessName": "TechCo"},
            "contacts": [{"id": "c-1", "name": "John"}],
            "supervisor_notes": "Data verified.",
        }],
        "rejected": [],
        "reply": "1 lead created successfully",
    }
    with patch("app.agents.crm.router._get_graph") as mock_graph:
        mock_graph.return_value.ainvoke = AsyncMock(return_value=mock_result)
        resp = client.post("/agents/crm/lead-research", json={
            "query": "SaaS companies", "icp_id": "icp-1", "count": 1
        })
    data = resp.json()
    assert resp.status_code == 200
    assert data["status"] == "created"
    assert len(data["leads"]) == 1
    assert data["leads"][0]["lead"]["businessName"] == "TechCo"


def test_partial_when_some_rejected():
    mock_result = {
        "created_leads": [{
            "lead": {"id": "lead-1", "businessName": "TechCo"},
            "contacts": [{"id": "c-1", "name": "John"}],
            "supervisor_notes": "Verified.",
        }],
        "rejected": [{"query_term": "Bad Co", "reason": "No contact info"}],
        "reply": "1 of 2 leads created. 1 rejected by supervisor.",
    }
    with patch("app.agents.crm.router._get_graph") as mock_graph:
        mock_graph.return_value.ainvoke = AsyncMock(return_value=mock_result)
        resp = client.post("/agents/crm/lead-research", json={
            "query": "test", "icp_id": "icp-1", "count": 2
        })
    data = resp.json()
    assert resp.status_code == 200
    assert data["status"] == "partial"
    assert len(data["leads"]) == 1
    assert len(data["rejected"]) == 1
