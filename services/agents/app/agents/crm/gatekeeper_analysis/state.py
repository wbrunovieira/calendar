from __future__ import annotations

from typing import Any

from typing_extensions import TypedDict


class GatekeeperAnalysisState(TypedDict, total=False):
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

    # ── Node 1: analyze_dimensions (6 parallel Haiku calls) ───────────────
    recepcao: dict   # R — abertura, tom, ausência de "cheiro de vendedor"
    alianca: dict    # A — rapport com GK, nome, respeito, conexão humana
    perguntas: dict  # P — perguntas que empoderam o GK
    objecoes: dict   # O — objeções recebidas e como foram tratadas
    resultado: dict  # R — o que foi obtido efetivamente
    tecnicas: dict   # T — técnicas usadas

    # ── Node 2: build_synthesis (Sonnet, recebe só os 6 dicts) ───────────
    summary: str
    positive_points: list
    improvement_points: list

    # ── Node 3: compute_scores (determinístico) ───────────────────────────
    score: float

    # ── Error ─────────────────────────────────────────────────────────────
    error: str | None
