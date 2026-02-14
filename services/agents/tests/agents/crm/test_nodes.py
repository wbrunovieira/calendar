from __future__ import annotations

import json
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from app.agents.crm.nodes import (
    load_icp_context,
    structure_leads_data,
    supervisor_review,
    save_to_crm,
)

pytestmark = pytest.mark.asyncio


def _mock_claude_response(text: str) -> MagicMock:
    """Create a mock Anthropic response with proper content block."""
    block = SimpleNamespace(type="text", text=text)
    resp = MagicMock()
    resp.content = [block]
    resp.stop_reason = "end_turn"
    return resp


# ── load_icp_context ────────────────────────────────────────


async def test_load_icp_context_success():
    fake_icp = {
        "id": "icp-1",
        "name": "SaaS B2B",
        "content": "Companies selling SaaS to SMBs in Brazil",
    }
    with patch("app.agents.crm.nodes.crm.get_icp", new_callable=AsyncMock, return_value=fake_icp), \
         patch("app.agents.crm.nodes.crm.get_leads_by_icp", new_callable=AsyncMock, return_value=[]):
        result = await load_icp_context({"icp_id": "icp-1", "_trace": None})
    assert result["icp_context"] == fake_icp
    assert "error" not in result
    assert result["existing_leads"] == []


async def test_load_icp_context_not_found():
    with patch(
        "app.agents.crm.nodes.crm.get_icp",
        new_callable=AsyncMock,
        side_effect=LookupError("ICP 'bad-id' not found"),
    ), patch("app.agents.crm.nodes.crm.get_leads_by_icp", new_callable=AsyncMock, return_value=[]):
        result = await load_icp_context({"icp_id": "bad-id", "_trace": None})
    assert result["error"] == "icp_not_found"
    assert "reply" in result


async def test_load_icp_context_fetches_existing_leads():
    fake_icp = {"id": "icp-1", "name": "SaaS B2B", "content": "SaaS companies"}
    existing = [
        {"id": "lead-1", "businessName": "Hotmart"},
        {"id": "lead-2", "businessName": "RD Station"},
    ]
    with patch("app.agents.crm.nodes.crm.get_icp", new_callable=AsyncMock, return_value=fake_icp), \
         patch("app.agents.crm.nodes.crm.get_leads_by_icp", new_callable=AsyncMock, return_value=existing):
        result = await load_icp_context({"icp_id": "icp-1", "_trace": None})
    assert result["existing_leads"] == ["Hotmart", "RD Station"]


async def test_load_icp_context_preloaded_still_fetches_existing():
    """When ICP is pre-loaded (CRM format), existing leads should still be fetched."""
    existing = [{"id": "lead-1", "businessName": "Hotmart"}]
    with patch("app.agents.crm.nodes.crm.get_leads_by_icp", new_callable=AsyncMock, return_value=existing):
        result = await load_icp_context({
            "icp_id": "icp-1",
            "icp_context": {"id": "icp-1", "name": "SaaS B2B", "content": "..."},
            "_trace": None,
        })
    assert result["existing_leads"] == ["Hotmart"]
    # icp_context should NOT be overwritten (pre-loaded)
    assert "icp_context" not in result


# ── structure_leads_data ────────────────────────────────────


async def test_structure_leads_data_valid_json():
    structured = json.dumps({
        "lead": {"businessName": "TechCo", "email": "hi@techco.com", "source": "ai-research", "status": "new"},
        "contacts": [{"name": "John", "role": "CEO", "isPrimary": True}],
    })

    mock_resp = _mock_claude_response(structured)

    with patch("app.agents.crm.nodes._call_claude", new_callable=AsyncMock, return_value=mock_resp):
        result = await structure_leads_data({
            "raw_dossiers": ["=== COMPANY: TechCo ===\nWebsite: techco.com"],
            "_trace": None,
        })
    assert len(result["leads_data"]) == 1
    assert result["leads_data"][0]["businessName"] == "TechCo"
    assert len(result["contacts_data"]) == 1


async def test_structure_leads_data_missing_business_name():
    structured = json.dumps({"error": "missing_business_name"})

    mock_resp = _mock_claude_response(structured)

    with patch("app.agents.crm.nodes._call_claude", new_callable=AsyncMock, return_value=mock_resp):
        result = await structure_leads_data({
            "raw_dossiers": ["=== COMPANY: ??? ==="],
            "_trace": None,
        })
    assert len(result["leads_data"]) == 0


# ── supervisor_review ───────────────────────────────────────


async def test_supervisor_approves_good_lead():
    review = json.dumps({"approved": True, "notes": "Data verified.", "issues": []})

    mock_resp = _mock_claude_response(review)

    with patch("app.agents.crm.nodes._call_claude", new_callable=AsyncMock, return_value=mock_resp):
        result = await supervisor_review({
            "leads_data": [{"businessName": "TechCo", "email": "hi@techco.com"}],
            "contacts_data": [[{"name": "John Smith", "role": "CEO"}]],
            "icp_context": {"name": "SaaS B2B", "content": "SaaS companies"},
            "_trace": None,
        })
    assert 0 in result["approved_indices"]
    assert len(result["rejected"]) == 0


async def test_supervisor_rejects_bad_lead():
    review = json.dumps({"approved": False, "notes": "Generic data.", "issues": ["No real contact"]})

    mock_resp = _mock_claude_response(review)

    with patch("app.agents.crm.nodes._call_claude", new_callable=AsyncMock, return_value=mock_resp):
        result = await supervisor_review({
            "leads_data": [{"businessName": "Company XYZ"}],
            "contacts_data": [[{"name": "Unknown"}]],
            "icp_context": {"name": "SaaS B2B", "content": "SaaS companies"},
            "_trace": None,
        })
    assert len(result["approved_indices"]) == 0
    assert len(result["rejected"]) == 1
    assert "Generic data" in result["rejected"][0]["reason"]


async def test_supervisor_accumulates_rejected_across_rounds():
    """Rejected from previous rounds should be preserved."""
    review = json.dumps({"approved": False, "notes": "Bad data.", "issues": []})
    mock_resp = _mock_claude_response(review)

    previous_rejected = [{"query_term": "PrevCo", "reason": "From round 1"}]

    with patch("app.agents.crm.nodes._call_claude", new_callable=AsyncMock, return_value=mock_resp):
        result = await supervisor_review({
            "leads_data": [{"businessName": "NewCo"}],
            "contacts_data": [[{"name": "Unknown"}]],
            "icp_context": {"name": "SaaS B2B", "content": "SaaS companies"},
            "rejected": previous_rejected,
            "_trace": None,
        })
    assert len(result["rejected"]) == 2
    assert result["rejected"][0]["query_term"] == "PrevCo"
    assert result["rejected"][1]["query_term"] == "NewCo"


# ── save_to_crm ────────────────────────────────────────────


async def test_save_to_crm_creates_lead_and_contacts_and_links_icp():
    fake_lead = {"id": "lead-1", "businessName": "TechCo"}
    fake_contact = {"id": "contact-1", "name": "John"}

    with patch("app.agents.crm.nodes.crm.create_lead", new_callable=AsyncMock, return_value=fake_lead) as mock_create, \
         patch("app.agents.crm.nodes.crm.create_lead_contact", new_callable=AsyncMock, return_value=fake_contact), \
         patch("app.agents.crm.nodes.crm.link_lead_to_icp", new_callable=AsyncMock) as mock_link:
        result = await save_to_crm({
            "leads_data": [{"businessName": "TechCo", "email": "hi@techco.com"}],
            "contacts_data": [[{"name": "John Smith", "role": "CEO"}]],
            "approved_indices": [0],
            "icp_id": "icp-123",
            "_trace": None,
        })
    assert len(result["created_leads"]) == 1
    assert result["created_leads"][0]["lead"]["id"] == "lead-1"
    mock_link.assert_called_once_with("lead-1", "icp-123")


async def test_save_to_crm_skips_rejected():
    with patch("app.agents.crm.nodes.crm.create_lead", new_callable=AsyncMock) as mock_create, \
         patch("app.agents.crm.nodes.crm.link_lead_to_icp", new_callable=AsyncMock):
        result = await save_to_crm({
            "leads_data": [{"businessName": "Rejected Co"}, {"businessName": "Approved Co"}],
            "contacts_data": [[{"name": "A"}], [{"name": "B"}]],
            "approved_indices": [1],
            "icp_id": "icp-1",
            "_trace": None,
        })
    # Only the approved lead (index 1) should be created
    assert mock_create.call_count == 1
    assert mock_create.call_args[0][0]["businessName"] == "Approved Co"
