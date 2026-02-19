from __future__ import annotations

from app.agents.crm.graph import (
    _after_investigate,
    _after_load_icp,
    _after_save,
    build_crm_graph,
)


def test_graph_compiles():
    graph = build_crm_graph()
    assert graph is not None


def test_graph_has_expected_nodes():
    graph = build_crm_graph()
    node_names = set(graph.nodes.keys())
    expected = {
        "load_icp_context",
        "investigate_leads",
        "structure_leads_data",
        "supervisor_review",
        "enrich_contacts",
        "save_to_crm",
        "format_reply",
    }
    assert expected.issubset(node_names)


def test_after_investigate_continues_on_empty_dossiers():
    """When investigation returns empty dossiers (no error), flow continues."""
    state = {"raw_dossiers": [], "created_leads": [{"lead": {}}]}
    assert _after_investigate(state) == "structure_leads_data"


def test_after_investigate_ends_on_real_error_with_results():
    """When investigation has a real error but previous results exist, go to format_reply."""
    state = {"error": "auth_error", "created_leads": [{"lead": {}}]}
    assert _after_investigate(state) == "format_reply"


def test_after_save_retries_when_remaining():
    """After save with remaining count, loop back to investigate."""
    state = {"remaining_count": 2, "retry_round": 1}
    assert _after_save(state) == "investigate_leads"


def test_after_save_stops_at_max_retries():
    """After max retries, go to format_reply even if remaining."""
    state = {"remaining_count": 2, "retry_round": 3}
    assert _after_save(state) == "format_reply"


def test_after_save_stops_when_search_saturated():
    """When search is saturated, stop retrying even if remaining and rounds left."""
    state = {"remaining_count": 3, "retry_round": 1, "search_saturated": True}
    assert _after_save(state) == "format_reply"


def test_after_save_retries_when_not_saturated():
    """When search is NOT saturated and rounds left, continue retrying."""
    state = {"remaining_count": 3, "retry_round": 1, "search_saturated": False}
    assert _after_save(state) == "investigate_leads"


def test_after_load_icp_skips_when_excess_covers_count():
    """When excess cache fully covers requested count, skip to format_reply."""
    state = {"created_leads": [{"lead": {}}], "remaining_count": 0}
    assert _after_load_icp(state) == "format_reply"


def test_after_load_icp_continues_when_no_excess():
    """Normal flow — no excess, continue to investigate."""
    state = {}
    assert _after_load_icp(state) == "investigate_leads"


def test_after_load_icp_continues_when_excess_partial():
    """When excess only partially covers count, continue investigating."""
    state = {"created_leads": [{"lead": {}}], "remaining_count": 3}
    assert _after_load_icp(state) == "investigate_leads"
