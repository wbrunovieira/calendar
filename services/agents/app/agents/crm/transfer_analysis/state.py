from __future__ import annotations

from typing import Any

from typing_extensions import TypedDict


class TransferAnalysisState(TypedDict, total=False):
    # ── Input ─────────────────────────────────────────────────────────────
    gk_job_id: str
    spiced_job_id: str
    gk_webhook_url: str
    spiced_webhook_url: str
    transcript: str
    call_duration_seconds: int
    call_date: str
    lead: dict
    contact: dict
    activity: dict
    _trace: Any

    # ── Node 1: segment_transcript ────────────────────────────────────────
    gk_transcript: str          # trecho SDR ↔ porteiro
    decisor_transcript: str     # trecho SDR ↔ decisor
    transition_note: str        # descrição da transição

    # ── Error (only set by segment_transcript on failure) ─────────────────
    error: str | None
