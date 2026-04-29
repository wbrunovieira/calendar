from __future__ import annotations

from typing import Any

from typing_extensions import TypedDict


class CallAnalysisState(TypedDict, total=False):
    # ── Input ─────────────────────────────────────────────────────────────
    job_id: str
    webhook_url: str
    transcript: str
    call_duration_seconds: int
    call_date: str
    lead: dict
    contact: dict
    activity: dict
    _trace: Any

    # ── Node 1: analyze_spiced (Sonnet) ───────────────────────────────────
    spiced: dict        # situation/pain/impact/criticalEvent/evidence → {score, text}
    micro_pactos: list  # 7 fixed checkpoints → {id, label, spicedDimension, achieved, notes}
    scheduling_techniques: dict  # 4 techniques → {used, notes}

    # ── Node 2: build_analysis (Haiku) ────────────────────────────────────
    micro_analysis: list
    summary: str
    positive_points: list
    improvement_points: list

    # ── Node 3: compute_scores (deterministic) ────────────────────────────
    score: float
    no_show_risk: str
    no_show_risk_text: str

    # ── Error ─────────────────────────────────────────────────────────────
    error: str | None
