from __future__ import annotations

from typing import Any

from typing_extensions import TypedDict


class GatekeeperBatchState(TypedDict, total=False):
    # ── Input ─────────────────────────────────────────────────────────────
    batch_job_id: str
    webhook_url: str
    analyses: list          # N análises individuais já processadas
    historical_summaries: list   # summaries de lotes anteriores
    _trace: Any

    # ── Node 1: compute_batch_scores (determinístico) ─────────────────────
    dimension_averages: dict   # média por dimensão RAPORT
    overall_score: float
    individual_highlights: list  # best + worst por score

    # ── Node 2: analyze_batch (2 Haiku em paralelo) ───────────────────────
    patterns: dict             # working, notWorking, keepDoing
    comparison_with_history: dict  # improved, worsened, unchanged, firstBatch

    # ── Node 3: build_batch_synthesis (Sonnet) ────────────────────────────
    recommendations: list
    new_summary: str
    positive_points: list
    improvement_points: list

    # ── Error ─────────────────────────────────────────────────────────────
    error: str | None
