from __future__ import annotations

import json
import logging
import re

import httpx

from app.agents.crm.nodes import _call_claude
from app.agents.crm.transfer_analysis.state import TransferAnalysisState
from app.agents.crm.gatekeeper_analysis.nodes import (
    analyze_dimensions as _gk_analyze_dimensions,
    build_synthesis as _gk_build_synthesis,
    compute_scores as _gk_compute_scores,
)
from app.agents.crm.call_analysis.nodes import (
    analyze_spiced as _call_analyze_spiced,
    build_analysis as _call_build_analysis,
    compute_scores as _call_compute_scores,
)
from app.agents.finances.nodes import langfuse
from app.config import settings

logger = logging.getLogger("agents")

_SONNET = "claude-sonnet-4-6"


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


async def _post_json(url: str, payload: dict) -> None:
    if not url:
        return
    try:
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.post(url, json=payload, headers=_auth_headers())
            logger.info("[TransferAnalysis] webhook posted url=%s status=%d", url, resp.status_code)
    except Exception:
        logger.exception("[TransferAnalysis] webhook post failed url=%s", url)


async def _post_error(url: str, job_id: str, error_msg: str) -> None:
    await _post_json(url, {"jobId": job_id, "status": "error", "error": error_msg})


# ── Node 1: segment_transcript ────────────────────────────────────────────

_SEGMENT_SYSTEM = """\
Você é especialista em análise de ligações comerciais B2B.
Analise a transcrição completa de uma ligação onde um SDR fala primeiro com
um porteiro (recepcionista/assistente) e depois é transferido ao decisor.

Identifique o ponto exato de transição e divida a transcrição em duas partes.
Retorne APENAS JSON válido, sem markdown, sem explicações."""

_SEGMENT_USER = """\
EMPRESA: {business_name} | CONTATO INICIAL: {contact_name} ({contact_role})

TRANSCRIÇÃO COMPLETA:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "gkTranscript": "trecho completo da conversa entre SDR e porteiro, do início até o momento da transferência (inclusive a fala do porteiro confirmando a transferência)",
  "decisorTranscript": "trecho completo da conversa entre SDR e decisor, a partir da primeira fala do decisor até o fim",
  "transitionNote": "descrição curta da transição (ex: 'Recepcionista Maria transferiu ao Carlos Mendes, Diretor Comercial')"
}}

REGRAS:
- Se não houver transição clara (ligação encerrou no porteiro), coloque toda a transcrição em gkTranscript e decisorTranscript como string vazia ""
- gkTranscript DEVE incluir a fala do porteiro confirmando/realizando a transferência
- decisorTranscript DEVE começar com a primeira fala do decisor após atender"""


async def segment_transcript(state: TransferAnalysisState) -> dict:
    lead = state.get("lead", {})
    contact = state.get("contact", {})
    transcript = state.get("transcript", "")
    trace = state.get("_trace")
    job_pair = f"{state.get('gk_job_id','?')}/{state.get('spiced_job_id','?')}"

    user = _SEGMENT_USER.format(
        business_name=lead.get("businessName", "N/A"),
        contact_name=contact.get("name", "N/A"),
        contact_role=contact.get("role", "N/A"),
        transcript=transcript[:12000],
    )

    gen = (
        trace.generation(name="ta_segment", model=_SONNET, input=[{"role": "user", "content": user}])
        if trace else None
    )

    try:
        raw = await _call_claude(system=_SEGMENT_SYSTEM, user=user, model=_SONNET, max_tokens=4000)
        text = raw.content[0].text if raw.content else "{}"
        parsed = _parse_json(text)

        gk_t = parsed.get("gkTranscript", "")
        dm_t = parsed.get("decisorTranscript", "")
        note = parsed.get("transitionNote", "")

        if gen:
            gen.end(output={"gk_len": len(gk_t), "dm_len": len(dm_t)})

        logger.info(
            "[TransferAnalysis] segment done jobs=%s gk_len=%d dm_len=%d",
            job_pair, len(gk_t), len(dm_t),
        )
        return {
            "gk_transcript": gk_t,
            "decisor_transcript": dm_t,
            "transition_note": note,
        }

    except Exception:
        logger.exception("[TransferAnalysis] segment_transcript failed jobs=%s", job_pair)
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "segment_transcript_failed"}


# ── Node 2: analyze_gk_phase (RAPORT) ─────────────────────────────────────


async def analyze_gk_phase(state: TransferAnalysisState) -> dict:
    gk_job_id = state.get("gk_job_id", "")
    gk_webhook_url = state.get("gk_webhook_url", "")
    gk_transcript = state.get("gk_transcript", "")
    trace = state.get("_trace")

    if not gk_transcript:
        logger.warning("[TransferAnalysis] empty gk_transcript job=%s", gk_job_id)
        await _post_error(gk_webhook_url, gk_job_id, "no_gk_transcript")
        return {}

    gk_state = {
        "job_id": gk_job_id,
        "webhook_url": gk_webhook_url,
        "transcript": gk_transcript,
        "call_duration_seconds": state.get("call_duration_seconds", 0),
        "call_date": state.get("call_date", ""),
        "lead": state.get("lead", {}),
        "contact": state.get("contact", {}),
        "activity": state.get("activity", {}),
        "_trace": trace,
    }

    try:
        # Step 1: 6 parallel RAPORT dimensions
        dims = await _gk_analyze_dimensions(gk_state)
        if "error" in dims:
            logger.error("[TransferAnalysis] gk dimensions failed job=%s error=%s", gk_job_id, dims["error"])
            await _post_error(gk_webhook_url, gk_job_id, dims["error"])
            return {}

        merged = {**gk_state, **dims}

        # Step 2: Sonnet synthesis
        synth = await _gk_build_synthesis(merged)
        if "error" in synth:
            logger.error("[TransferAnalysis] gk synthesis failed job=%s error=%s", gk_job_id, synth["error"])
            await _post_error(gk_webhook_url, gk_job_id, synth["error"])
            return {}

        merged = {**merged, **synth}

        # Step 3: deterministic score
        scores = await _gk_compute_scores(merged)

        payload = {
            "jobId": gk_job_id,
            "status": "completed",
            "score": scores.get("score"),
            "summary": synth.get("summary"),
            "positivePoints": synth.get("positive_points"),
            "improvementPoints": synth.get("improvement_points"),
            "raport": {
                "recepcao":  dims.get("recepcao"),
                "alianca":   dims.get("alianca"),
                "perguntas": dims.get("perguntas"),
                "objecoes":  dims.get("objecoes"),
                "resultado": dims.get("resultado"),
                "tecnicas":  dims.get("tecnicas"),
            },
        }
        await _post_json(gk_webhook_url, payload)
        logger.info("[TransferAnalysis] gk phase complete job=%s score=%s", gk_job_id, scores.get("score"))

    except Exception:
        logger.exception("[TransferAnalysis] analyze_gk_phase failed job=%s", gk_job_id)
        await _post_error(gk_webhook_url, gk_job_id, "analyze_gk_phase_failed")

    return {}


# ── Node 3: analyze_spiced_phase ──────────────────────────────────────────


async def analyze_spiced_phase(state: TransferAnalysisState) -> dict:
    spiced_job_id = state.get("spiced_job_id", "")
    spiced_webhook_url = state.get("spiced_webhook_url", "")
    decisor_transcript = state.get("decisor_transcript", "")
    transition_note = state.get("transition_note", "")
    trace = state.get("_trace")

    if not decisor_transcript:
        logger.warning("[TransferAnalysis] empty decisor_transcript job=%s", spiced_job_id)
        await _post_error(spiced_webhook_url, spiced_job_id, "no_decisor_transcript")
        return {}

    # Prepend transition note so SPICED has context about how the call started
    transcript_with_context = (
        f"[CONTEXTO DA TRANSFERÊNCIA: {transition_note}]\n\n{decisor_transcript}"
        if transition_note else decisor_transcript
    )

    spiced_state = {
        "job_id": spiced_job_id,
        "webhook_url": spiced_webhook_url,
        "transcript": transcript_with_context,
        "call_duration_seconds": state.get("call_duration_seconds", 0),
        "call_date": state.get("call_date", ""),
        "lead": state.get("lead", {}),
        "contact": state.get("contact", {}),
        "activity": state.get("activity", {}),
        "_trace": trace,
    }

    try:
        # Step 1: SPICED analysis (Sonnet)
        spiced_result = await _call_analyze_spiced(spiced_state)
        if "error" in spiced_result:
            logger.error("[TransferAnalysis] spiced analysis failed job=%s error=%s", spiced_job_id, spiced_result["error"])
            await _post_error(spiced_webhook_url, spiced_job_id, spiced_result["error"])
            return {}

        merged = {**spiced_state, **spiced_result}

        # Step 2: build analysis (Haiku)
        analysis_result = await _call_build_analysis(merged)
        if "error" in analysis_result:
            logger.error("[TransferAnalysis] build analysis failed job=%s error=%s", spiced_job_id, analysis_result["error"])
            await _post_error(spiced_webhook_url, spiced_job_id, analysis_result["error"])
            return {}

        merged = {**merged, **analysis_result}

        # Step 3: deterministic scoring
        scores_result = await _call_compute_scores(merged)

        payload = {
            "jobId": spiced_job_id,
            "status": "completed",
            "score": scores_result.get("score"),
            "noShowRisk": scores_result.get("no_show_risk"),
            "noShowRiskText": scores_result.get("no_show_risk_text"),
            "summary": analysis_result.get("summary"),
            "spiced": spiced_result.get("spiced"),
            "microPactos": spiced_result.get("micro_pactos"),
            "schedulingTechniques": spiced_result.get("scheduling_techniques"),
            "microAnalysis": analysis_result.get("micro_analysis"),
            "positivePoints": analysis_result.get("positive_points"),
            "improvementPoints": analysis_result.get("improvement_points"),
        }
        await _post_json(spiced_webhook_url, payload)
        logger.info(
            "[TransferAnalysis] spiced phase complete job=%s score=%s risk=%s",
            spiced_job_id, scores_result.get("score"), scores_result.get("no_show_risk"),
        )

    except Exception:
        logger.exception("[TransferAnalysis] analyze_spiced_phase failed job=%s", spiced_job_id)
        await _post_error(spiced_webhook_url, spiced_job_id, "analyze_spiced_phase_failed")

    return {}


# ── Error fallback: send_both_errors ─────────────────────────────────────


async def send_both_errors(state: TransferAnalysisState) -> dict:
    gk_job_id = state.get("gk_job_id", "")
    spiced_job_id = state.get("spiced_job_id", "")
    gk_webhook_url = state.get("gk_webhook_url", "")
    spiced_webhook_url = state.get("spiced_webhook_url", "")
    error = state.get("error", "segment_transcript_failed")

    logger.error(
        "[TransferAnalysis] send_both_errors gk=%s spiced=%s error=%s",
        gk_job_id, spiced_job_id, error,
    )
    await _post_error(gk_webhook_url, gk_job_id, error)
    await _post_error(spiced_webhook_url, spiced_job_id, error)
    return {}
