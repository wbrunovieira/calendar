from __future__ import annotations

from app.agents.crm.graph import build_crm_graph


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
