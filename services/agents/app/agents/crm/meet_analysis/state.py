from __future__ import annotations

from typing import Any

from typing_extensions import TypedDict


class MeetAnalysisState(TypedDict, total=False):
    # ── Input ─────────────────────────────────────────────────────────────
    job_id: str
    webhook_url: str
    transcript: str
    meeting_duration_seconds: int
    meeting_date: str
    meeting_title: str
    lead: dict
    contact: dict
    activity: dict
    _trace: Any

    # ── Node 1: analyze_dimensions (6 parallel Haiku calls) ───────────────
    business: dict        # DIAG-D
    gaps: dict            # DIAG-G
    urgency: dict         # DIAG-U
    decision_power: dict  # DIAG-P
    engagement: dict      # DIAG-E
    closing: dict         # DIAG-F

    # ── Node 2: build_synthesis (Sonnet, receives only structured dicts) ──
    summary: str
    next_step: str
    positive_points: list
    improvement_points: list

    # ── Node 3: compute_scores (deterministic) ────────────────────────────
    score: float
    closing_probability: int

    # ── Error ─────────────────────────────────────────────────────────────
    error: str | None
