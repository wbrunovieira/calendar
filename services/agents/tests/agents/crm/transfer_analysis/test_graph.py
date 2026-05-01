from __future__ import annotations

from app.agents.crm.transfer_analysis.graph import (
    _after_segment_transcript,
    build_transfer_analysis_graph,
)


def test_graph_compiles():
    graph = build_transfer_analysis_graph()
    assert graph is not None


def test_graph_has_expected_nodes():
    graph = build_transfer_analysis_graph()
    node_names = set(graph.nodes.keys())
    expected = {"segment_transcript", "analyze_gk_phase", "analyze_spiced_phase", "send_both_errors"}
    assert expected.issubset(node_names)


def test_after_segment_routes_to_analyze_gk_on_success():
    state = {
        "gk_transcript": "SDR: Bom dia!",
        "decisor_transcript": "Decisor: Alô?",
        "transition_note": "Maria transferiu",
    }
    assert _after_segment_transcript(state) == "analyze_gk_phase"


def test_after_segment_routes_to_send_both_errors_on_error():
    state = {"error": "segment_transcript_failed"}
    assert _after_segment_transcript(state) == "send_both_errors"


def test_after_segment_routes_to_send_both_errors_when_no_error_key_but_gk_transcript_present():
    state = {"gk_transcript": "text", "decisor_transcript": "text", "transition_note": "ok"}
    assert _after_segment_transcript(state) == "analyze_gk_phase"
