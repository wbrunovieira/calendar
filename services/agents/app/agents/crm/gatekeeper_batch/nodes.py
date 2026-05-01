from __future__ import annotations

import asyncio
import json
import logging
import re

import httpx

from app.agents.crm.nodes import _call_claude
from app.agents.crm.gatekeeper_batch.state import GatekeeperBatchState
from app.agents.finances.nodes import langfuse
from app.config import settings

logger = logging.getLogger("agents")

_HAIKU = "claude-haiku-4-5-20251001"
_SONNET = "claude-sonnet-4-6"

_RAPORT_DIMS = ("recepcao", "alianca", "perguntas", "objecoes", "resultado", "tecnicas")


# ── Helpers ───────────────────────────────────────────────────────────────

def _parse_json(text: str) -> dict:
    text = re.sub(r"^```(?:json)?\s*", "", text.strip())
    text = re.sub(r"\s*```$", "", text.strip())
    return json.loads(text)


def _auth_headers() -> dict[str, str]:
    headers: dict[str, str] = {"Content-Type": "application/json"}
    if settings.crm_webhook_secret:
        headers["X-Webhook-Secret"] = settings.crm_webhook_secret
    return headers


def _compute_dimension_averages(analyses: list) -> dict:
    """Return average score per RAPORT dimension across all analyses."""
    totals: dict[str, list] = {d: [] for d in _RAPORT_DIMS}
    for analysis in analyses:
        raport = analysis.get("raport") or {}
        for dim in _RAPORT_DIMS:
            score = (raport.get(dim) or {}).get("score")
            if isinstance(score, (int, float)):
                totals[dim].append(score)
    return {
        dim: round(sum(vals) / len(vals), 1) if vals else 0.0
        for dim, vals in totals.items()
    }


def _extract_individual_highlights(analyses: list) -> list:
    """Return best and worst individual analyses by score."""
    scored = [
        {"jobId": a.get("jobId", "?"), "score": a.get("score", 0)}
        for a in analyses
        if isinstance(a.get("score"), (int, float))
    ]
    if not scored:
        return []
    scored.sort(key=lambda x: x["score"], reverse=True)
    highlights = []
    if scored:
        highlights.append({"type": "best", **scored[0]})
    if len(scored) > 1:
        highlights.append({"type": "worst", **scored[-1]})
    return highlights


# ── LangGraph Node 1: compute_batch_scores (deterministic) ───────────────


async def compute_batch_scores(state: GatekeeperBatchState) -> dict:
    analyses = state.get("analyses") or []
    batch_job_id = state.get("batch_job_id", "?")

    if not analyses:
        logger.warning("[GKBatch] no analyses provided batch_job_id=%s", batch_job_id)
        return {"error": "no_analyses_provided"}

    dimension_averages = _compute_dimension_averages(analyses)
    all_scores = [a.get("score", 0) for a in analyses if isinstance(a.get("score"), (int, float))]
    overall_score = round(sum(all_scores) / len(all_scores), 1) if all_scores else 0.0
    individual_highlights = _extract_individual_highlights(analyses)

    logger.info(
        "[GKBatch] compute_batch_scores batch=%s n=%d overall=%.1f",
        batch_job_id, len(analyses), overall_score,
    )
    return {
        "dimension_averages": dimension_averages,
        "overall_score": overall_score,
        "individual_highlights": individual_highlights,
    }


# ── Node 2 helpers: _analyze_patterns + _compare_with_history ────────────

_PATTERNS_SYSTEM = """\
Você é especialista em treinamento de equipes de SDRs. Receberá os dados consolidados de um lote
de análises RAPORT de ligações para porteiros.
Identifique padrões no comportamento da equipe. Retorne APENAS JSON válido, sem markdown."""

_PATTERNS_USER = """\
LOTE: {batch_job_id} | LIGAÇÕES ANALISADAS: {n_analyses}

MÉDIAS POR DIMENSÃO RAPORT:
{dimension_averages_json}

DESTAQUES INDIVIDUAIS:
{highlights_json}

Retorne JSON:
{{
  "working": ["comportamento 1 que está funcionando bem na equipe", "..."],
  "notWorking": ["problema 1 recorrente identificado no lote", "..."],
  "keepDoing": ["prática 1 a reforçar", "..."]
}}

working: 2-4 pontos fortes recorrentes.
notWorking: 2-4 problemas recorrentes que precisam de atenção.
keepDoing: 1-3 práticas a manter e incentivar."""

_HISTORY_SYSTEM = """\
Você é especialista em coaching de vendas com foco em evolução de performance ao longo do tempo.
Compare as médias do lote atual com os resumos de lotes anteriores para identificar evolução.
Retorne APENAS JSON válido, sem markdown."""

_HISTORY_USER = """\
MÉDIAS ATUAIS:
{current_averages_json}

RESUMOS HISTÓRICOS (lotes anteriores, do mais recente ao mais antigo):
{historical_json}

Retorne JSON:
{{
  "improved": ["dimensão/comportamento 1 que melhorou", "..."],
  "worsened": ["dimensão/comportamento 1 que piorou", "..."],
  "unchanged": ["dimensão/comportamento 1 estável", "..."],
  "firstBatch": {first_batch}
}}

Se for o primeiro lote (sem histórico), retorne firstBatch: true e listas vazias."""


async def _analyze_patterns(
    batch_job_id: str,
    analyses: list,
    dimension_averages: dict,
    individual_highlights: list,
    trace: object,
) -> dict:
    user = _PATTERNS_USER.format(
        batch_job_id=batch_job_id,
        n_analyses=len(analyses),
        dimension_averages_json=json.dumps(dimension_averages, ensure_ascii=False, indent=2),
        highlights_json=json.dumps(individual_highlights, ensure_ascii=False, indent=2),
    )
    gen = (
        trace.generation(name="gkbatch_patterns", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_PATTERNS_SYSTEM, user=user, model=_HAIKU, max_tokens=800)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"working_count": len(result.get("working", []))})
        return result
    except Exception:
        logger.exception("[GKBatch] _analyze_patterns failed batch=%s", batch_job_id)
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_patterns_failed"}


async def _compare_with_history(
    dimension_averages: dict,
    historical_summaries: list,
    trace: object,
) -> dict:
    first_batch = "true" if not historical_summaries else "false"
    user = _HISTORY_USER.format(
        current_averages_json=json.dumps(dimension_averages, ensure_ascii=False, indent=2),
        historical_json=json.dumps(historical_summaries or [], ensure_ascii=False, indent=2),
        first_batch=first_batch,
    )
    gen = (
        trace.generation(name="gkbatch_history", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_HISTORY_SYSTEM, user=user, model=_HAIKU, max_tokens=600)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"first_batch": result.get("firstBatch")})
        return result
    except Exception:
        logger.exception("[GKBatch] _compare_with_history failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "compare_history_failed"}


# ── LangGraph Node 2: analyze_batch (2 Haiku parallel) ───────────────────


async def analyze_batch(state: GatekeeperBatchState) -> dict:
    batch_job_id = state.get("batch_job_id", "?")
    analyses = state.get("analyses") or []
    historical_summaries = state.get("historical_summaries") or []
    dimension_averages = state.get("dimension_averages") or {}
    individual_highlights = state.get("individual_highlights") or []
    trace = state.get("_trace")

    try:
        patterns, comparison = await asyncio.gather(
            _analyze_patterns(batch_job_id, analyses, dimension_averages, individual_highlights, trace),
            _compare_with_history(dimension_averages, historical_summaries, trace),
        )
    except Exception:
        logger.exception("[GKBatch] analyze_batch gather failed batch=%s", batch_job_id)
        return {"error": "analyze_batch_failed"}

    for name, result in [("patterns", patterns), ("comparison", comparison)]:
        if "error" in result:
            logger.error("[GKBatch] analyze_batch %s failed batch=%s", name, batch_job_id)
            return {"error": f"analyze_batch_{name}_failed"}

    logger.info("[GKBatch] analyze_batch complete batch=%s", batch_job_id)
    return {
        "patterns": patterns,
        "comparison_with_history": comparison,
    }


# ── LangGraph Node 3: build_batch_synthesis (Sonnet) ─────────────────────

_BATCH_SYNTHESIS_SYSTEM = """\
Você é especialista em coaching de equipes de SDRs com foco em melhoria contínua de prospecção.
Receberá análises de padrões e comparação histórica de um lote de ligações RAPORT.
Gere um resumo executivo e recomendações de coaching acionáveis para o gestor.
Retorne APENAS JSON válido, sem markdown."""

_BATCH_SYNTHESIS_USER = """\
LOTE: {batch_job_id} | LIGAÇÕES: {n_analyses} | SCORE MÉDIO: {overall_score}

MÉDIAS RAPORT:
{dimension_averages_json}

PADRÕES IDENTIFICADOS:
{patterns_json}

COMPARAÇÃO COM HISTÓRICO:
{comparison_json}

Retorne JSON:
{{
  "newSummary": "resumo do lote em 3-5 frases para histórico — inclua score médio, principais forças e fraquezas",
  "positivePoints": ["destaque positivo 1 da equipe neste lote", "..."],
  "improvementPoints": ["área de melhoria 1 prioritária", "..."],
  "recommendations": [
    {{
      "priority": "high"|"medium"|"low",
      "action": "ação de coaching específica e acionável",
      "targetDimension": "recepcao"|"alianca"|"perguntas"|"objecoes"|"resultado"|"tecnicas"|"geral"
    }}
  ]
}}

positivePoints: 2-4 itens.
improvementPoints: 2-4 itens.
recommendations: 3-5 recomendações ordenadas por prioridade."""


async def build_batch_synthesis(state: GatekeeperBatchState) -> dict:
    batch_job_id = state.get("batch_job_id", "?")
    analyses = state.get("analyses") or []
    dimension_averages = state.get("dimension_averages") or {}
    overall_score = state.get("overall_score", 0.0)
    patterns = state.get("patterns") or {}
    comparison = state.get("comparison_with_history") or {}
    trace = state.get("_trace")

    user = _BATCH_SYNTHESIS_USER.format(
        batch_job_id=batch_job_id,
        n_analyses=len(analyses),
        overall_score=overall_score,
        dimension_averages_json=json.dumps(dimension_averages, ensure_ascii=False, indent=2),
        patterns_json=json.dumps(patterns, ensure_ascii=False, indent=2),
        comparison_json=json.dumps(comparison, ensure_ascii=False, indent=2),
    )

    gen = (
        trace.generation(name="gkbatch_synthesis", model=_SONNET, input=[{"role": "user", "content": user}])
        if trace else None
    )

    try:
        raw = await _call_claude(system=_BATCH_SYNTHESIS_SYSTEM, user=user, model=_SONNET, max_tokens=1500)
        text = raw.content[0].text if raw.content else "{}"
        parsed = _parse_json(text)

        new_summary = parsed.get("newSummary", "")
        positive_points = parsed.get("positivePoints", [])
        improvement_points = parsed.get("improvementPoints", [])
        recommendations = parsed.get("recommendations", [])

        if gen:
            gen.end(output={"summary_len": len(new_summary), "recs": len(recommendations)})

        logger.info("[GKBatch] build_batch_synthesis complete batch=%s", batch_job_id)
        return {
            "new_summary": new_summary,
            "positive_points": positive_points,
            "improvement_points": improvement_points,
            "recommendations": recommendations,
        }

    except Exception:
        logger.exception("[GKBatch] build_batch_synthesis failed batch=%s", batch_job_id)
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "build_batch_synthesis_failed"}


# ── LangGraph Node 4: send_webhook ────────────────────────────────────────


async def send_webhook(state: GatekeeperBatchState) -> dict:
    batch_job_id = state.get("batch_job_id", "")
    webhook_url = state.get("webhook_url", "")
    error = state.get("error")

    if not webhook_url:
        logger.error("[GKBatch] no webhook_url — result lost for batch=%s", batch_job_id)
        return {}

    if error:
        payload: dict = {"batchJobId": batch_job_id, "status": "error", "error": error}
    else:
        payload = {
            "batchJobId": batch_job_id,
            "status": "completed",
            "overallScore": state.get("overall_score"),
            "dimensionAverages": state.get("dimension_averages"),
            "individualHighlights": state.get("individual_highlights"),
            "patterns": state.get("patterns"),
            "comparisonWithHistory": state.get("comparison_with_history"),
            "recommendations": state.get("recommendations"),
            "newSummary": state.get("new_summary"),
            "positivePoints": state.get("positive_points"),
            "improvementPoints": state.get("improvement_points"),
        }

    try:
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.post(webhook_url, json=payload, headers=_auth_headers())
            logger.info(
                "[GKBatch] webhook sent batch=%s status=%d error=%s",
                batch_job_id, resp.status_code, error,
            )
    except Exception:
        logger.exception("[GKBatch] webhook send failed batch=%s url=%s", batch_job_id, webhook_url)

    return {}
