from __future__ import annotations

import asyncio
import json
import logging
import re

import httpx

from app.agents.crm.nodes import _call_claude
from app.agents.crm.meet_analysis.state import MeetAnalysisState
from app.agents.finances.nodes import langfuse
from app.config import settings

logger = logging.getLogger("agents")

_HAIKU = "claude-haiku-4-5-20251001"
_SONNET = "claude-sonnet-4-6"

_TRANSCRIPT_LIMIT = 10_000  # chars per dimension call


# ── Shared context block ──────────────────────────────────────────────────

def _context_block(lead: dict, contact: dict, duration_min: int, meeting_date: str, meeting_title: str) -> str:
    return (
        f"REUNIÃO: {meeting_title or 'N/A'}\n"
        f"EMPRESA: {lead.get('businessName', 'N/A')} | "
        f"SEGMENTO: {lead.get('segment', 'N/A')} | "
        f"CIDADE: {lead.get('city', 'N/A')}\n"
        f"DESCRIÇÃO: {lead.get('description', 'N/A')}\n"
        f"CONTATO: {contact.get('name', 'N/A')} — {contact.get('role', 'N/A')}\n"
        f"DURAÇÃO: {duration_min} minutos | DATA: {meeting_date or 'N/A'}\n"
    )


# ── Helper ────────────────────────────────────────────────────────────────

def _parse_json(text: str) -> dict:
    text = re.sub(r"^```(?:json)?\s*", "", text.strip())
    text = re.sub(r"\s*```$", "", text.strip())
    return json.loads(text)


def _auth_headers() -> dict[str, str]:
    headers: dict[str, str] = {"Content-Type": "application/json"}
    if settings.crm_webhook_secret:
        headers["X-Webhook-Secret"] = settings.crm_webhook_secret
    return headers


# ── Deterministic scoring helper ──────────────────────────────────────────

def _compute_overall_score(dim_scores: dict[str, float]) -> float:
    values = [v for v in dim_scores.values() if isinstance(v, (int, float))]
    if not values:
        return 0.0
    return round(sum(values) / len(values), 1)


# ── DIAG-D: Diagnóstico do Negócio ────────────────────────────────────────

_BUSINESS_SYSTEM = """\
Você é especialista em diagnóstico de negócios B2B. Analise a transcrição de uma reunião de diagnóstico \
e extraia informações sobre o negócio atual do prospecto.
Foco: receita aproximada, modelo de negócio, base de clientes, ticket médio, objetivo declarado e tentativas anteriores.
Retorne APENAS JSON válido, sem markdown, sem explicações."""

_BUSINESS_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "currentRevenue": "receita aproximada mencionada ou null",
  "model": "como a empresa opera/vende",
  "customers": "quantidade ou perfil da base de clientes",
  "averageTicket": "ticket médio mencionado ou null",
  "objective": "meta de crescimento declarada pelo lead",
  "alreadyTried": "o que já tentou e por que não funcionou",
  "text": "análise completa em 3-5 frases",
  "score": 1-5
}}

Score: 5 = diagnóstico completo com números, 1 = informações vagas ou ausentes."""


async def _analyze_business(
    lead: dict, contact: dict, transcript: str,
    duration_min: int, meeting_date: str, meeting_title: str, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_min, meeting_date, meeting_title)
    user = _BUSINESS_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])

    gen = (
        trace.generation(name="diag_business", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_BUSINESS_SYSTEM, user=user, model=_HAIKU, max_tokens=800)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score")})
        return result
    except Exception:
        logger.exception("[MeetAnalysis] _analyze_business failed job=%s", "?")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_business_failed"}


# ── DIAG-G: Gargalos nas 6 Áreas ──────────────────────────────────────────

_GAPS_SYSTEM = """\
Você é especialista na metodologia Salto de crescimento comercial. Analise a transcrição e identifique \
gargalos nas 6 áreas de crescimento.
Áreas: Aquisição, Funil de Vendas, Oferta, Time Comercial, Retenção/Reativação, Dados e Decisão.
Retorne APENAS JSON válido, sem markdown, sem explicações."""

_GAPS_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "aquisicao":     {{"identified": bool, "text": "evidência ou null", "severity": "high"|"medium"|"low"|null}},
  "funil":         {{"identified": bool, "text": "evidência ou null", "severity": "high"|"medium"|"low"|null}},
  "oferta":        {{"identified": bool, "text": "evidência ou null", "severity": "high"|"medium"|"low"|null}},
  "timeComercial": {{"identified": bool, "text": "evidência ou null", "severity": "high"|"medium"|"low"|null}},
  "retencao":      {{"identified": bool, "text": "evidência ou null", "severity": "high"|"medium"|"low"|null}},
  "dados":         {{"identified": bool, "text": "evidência ou null", "severity": "high"|"medium"|"low"|null}},
  "text": "síntese dos gargalos em 2-3 frases",
  "score": 1-5
}}

Score: 5 = múltiplos gargalos claros e urgentes identificados, 1 = nenhum identificado ou empresa sem problemas evidentes.
severity: null quando identified=false."""


async def _analyze_gaps(
    lead: dict, contact: dict, transcript: str,
    duration_min: int, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_min, "", "")
    user = _GAPS_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])

    gen = (
        trace.generation(name="diag_gaps", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_GAPS_SYSTEM, user=user, model=_HAIKU, max_tokens=1000)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score")})
        return result
    except Exception:
        logger.exception("[MeetAnalysis] _analyze_gaps failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_gaps_failed"}


# ── DIAG-U: Urgência e Timing ─────────────────────────────────────────────

_URGENCY_SYSTEM = """\
Você é especialista em qualificação comercial B2B. Analise a transcrição e identifique a urgência \
e o timing do prospecto para resolver o problema.
Foco: o que mudou ou vai mudar, evento crítico com prazo, consequência de não agir.
Retorne APENAS JSON válido, sem markdown."""

_URGENCY_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "trigger": "o que mudou ou vai mudar que torna urgente resolver agora",
  "criticalEvent": "prazo, contratação, sazonalidade ou evento concreto mencionado",
  "consequence": "o que acontece em 90 dias se não resolver",
  "text": "análise de urgência em 2-3 frases",
  "score": 1-5
}}

Score: 5 = urgência explícita com prazo concreto, 1 = sem senso de urgência identificado."""


async def _analyze_urgency(
    lead: dict, contact: dict, transcript: str,
    duration_min: int, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_min, "", "")
    user = _URGENCY_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])

    gen = (
        trace.generation(name="diag_urgency", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_URGENCY_SYSTEM, user=user, model=_HAIKU, max_tokens=600)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score")})
        return result
    except Exception:
        logger.exception("[MeetAnalysis] _analyze_urgency failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_urgency_failed"}


# ── DIAG-P: Poder de Decisão ──────────────────────────────────────────────

_DECISION_SYSTEM = """\
Você é especialista em qualificação BANT/MEDDIC para vendas B2B. Analise a transcrição e avalie \
o poder de decisão e o processo de compra do prospecto.
Foco: quem decide, processo de aprovação, orçamento disponível.
Retorne APENAS JSON válido, sem markdown."""

_DECISION_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "decisionMaker": "quem decide a compra e se estava presente na reunião",
  "buyingProcess": "como a empresa toma decisões de compra declarado pelo lead",
  "budget": "orçamento mencionado, aprovado ou estimado",
  "text": "análise de poder de decisão em 2-3 frases",
  "score": 1-5
}}

Score: 5 = decisor presente, processo claro e budget confirmado, 1 = decisor ausente e processo opaco."""


async def _analyze_decision_power(
    lead: dict, contact: dict, transcript: str,
    duration_min: int, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_min, "", "")
    user = _DECISION_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])

    gen = (
        trace.generation(name="diag_decision", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_DECISION_SYSTEM, user=user, model=_HAIKU, max_tokens=600)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score")})
        return result
    except Exception:
        logger.exception("[MeetAnalysis] _analyze_decision_power failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_decision_power_failed"}


# ── DIAG-E: Engajamento e Clima ───────────────────────────────────────────

_ENGAGEMENT_SYSTEM = """\
Você é especialista em inteligência emocional para vendas B2B. Analise a transcrição e avalie \
o nível de engajamento e o clima da reunião.
Foco: participação ativa, perguntas feitas pelo cliente, resistências e rapport.
Retorne APENAS JSON válido, sem markdown."""

_ENGAGEMENT_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "level": "high"|"medium"|"low",
  "buyerQuestions": ["pergunta 1 do lead", "pergunta 2", "..."],
  "resistances": ["resistência 1 identificada", "..."],
  "rapport": "descrição do rapport — houve conexão? pontos em comum?",
  "text": "análise de engajamento em 2-3 frases",
  "score": 1-5
}}

Score: 5 = lead muito engajado, perguntas proativas e rapport forte, 1 = passivo, monossilábico, sem interesse."""


async def _analyze_engagement(
    lead: dict, contact: dict, transcript: str,
    duration_min: int, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_min, "", "")
    user = _ENGAGEMENT_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])

    gen = (
        trace.generation(name="diag_engagement", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_ENGAGEMENT_SYSTEM, user=user, model=_HAIKU, max_tokens=800)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score"), "level": result.get("level")})
        return result
    except Exception:
        logger.exception("[MeetAnalysis] _analyze_engagement failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_engagement_failed"}


# ── DIAG-F: Momentum de Fechamento ────────────────────────────────────────

_CLOSING_SYSTEM = """\
Você é especialista em técnicas de fechamento e momentum comercial B2B. Analise a transcrição e \
avalie os sinais de compra do cliente e as técnicas de fechamento usadas pelo vendedor.
Retorne APENAS JSON válido, sem markdown."""

_CLOSING_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "buyerSignals": [
    "sinal de compra 1 (pergunta sobre prazo, preço, implementação, cases, pagamento...)",
    "..."
  ],
  "sellerCloses": {{
    "trialClose":      {{"used": bool, "notes": "como tentou ou null"}},
    "nextStepAnchor":  {{"used": bool, "notes": "próximo passo ancorado ou null"}},
    "urgencyCreation": {{"used": bool, "notes": "urgência criada ou null"}}
  }},
  "closingProbability": 0-100,
  "text": "análise de momentum de fechamento em 2-3 frases",
  "score": 1-5
}}

closingProbability: baseado na densidade de sinais de compra + qualidade das técnicas do vendedor.
Score: 5 = múltiplos sinais fortes + vendedor fechou bem, 1 = sem sinais e sem técnicas."""


async def _analyze_closing(
    lead: dict, contact: dict, transcript: str,
    duration_min: int, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_min, "", "")
    user = _CLOSING_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])

    gen = (
        trace.generation(name="diag_closing", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_CLOSING_SYSTEM, user=user, model=_HAIKU, max_tokens=800)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score"), "prob": result.get("closingProbability")})
        return result
    except Exception:
        logger.exception("[MeetAnalysis] _analyze_closing failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_closing_failed"}


# ── DIAG-S: Síntese ───────────────────────────────────────────────────────

_SYNTHESIS_SYSTEM = """\
Você é especialista em coaching de vendas consultivas B2B com metodologia Salto.
Receberá os resultados de 6 análises DIAG de uma reunião de diagnóstico — NÃO a transcrição.
Sintetize em um resumo executivo e recomendações de coaching para o vendedor.
Retorne APENAS JSON válido, sem markdown."""

_SYNTHESIS_USER = """\
EMPRESA: {business_name} | CONTATO: {contact_name} ({contact_role})

DIAG-D — Negócio:
{business_json}

DIAG-G — Gargalos:
{gaps_json}

DIAG-U — Urgência:
{urgency_json}

DIAG-P — Poder de Decisão:
{decision_json}

DIAG-E — Engajamento:
{engagement_json}

DIAG-F — Fechamento:
{closing_json}

Retorne JSON:
{{
  "summary": "resumo executivo em 3-5 linhas: o negócio, a dor principal, o timing e o resultado da reunião",
  "nextStep": "próximo passo recomendado para as próximas 48h — concreto e acionável",
  "positivePoints": [
    "ponto positivo 1 — comportamento específico do vendedor ou característica do lead",
    "..."
  ],
  "improvementPoints": [
    "melhoria 1 — erro específico + como corrigir na próxima reunião",
    "..."
  ]
}}

REGRAS:
- positivePoints: 3-5 itens, cite fatos da análise
- improvementPoints: 3-5 itens, cite o gap exato e a ação corretiva
- nextStep: inclua prazo e formato (ex: "Enviar proposta por email até quarta 18h, confirmar reunião sexta 10h")"""


# ── LangGraph Node 1: analyze_dimensions ─────────────────────────────────


async def analyze_dimensions(state: MeetAnalysisState) -> dict:
    lead = state.get("lead", {})
    contact = state.get("contact", {})
    transcript = state.get("transcript", "")
    duration_min = (state.get("meeting_duration_seconds") or 0) // 60
    meeting_date = state.get("meeting_date", "")
    meeting_title = state.get("meeting_title", "")
    trace = state.get("_trace")
    job_id = state.get("job_id", "?")

    try:
        results = await asyncio.gather(
            _analyze_business(lead, contact, transcript, duration_min, meeting_date, meeting_title, trace),
            _analyze_gaps(lead, contact, transcript, duration_min, trace),
            _analyze_urgency(lead, contact, transcript, duration_min, trace),
            _analyze_decision_power(lead, contact, transcript, duration_min, trace),
            _analyze_engagement(lead, contact, transcript, duration_min, trace),
            _analyze_closing(lead, contact, transcript, duration_min, trace),
        )
    except Exception:
        logger.exception("[MeetAnalysis] analyze_dimensions gather failed job=%s", job_id)
        return {"error": "analyze_dimensions_failed"}

    business, gaps, urgency, decision_power, engagement, closing = results

    for name, dim in [
        ("business", business), ("gaps", gaps), ("urgency", urgency),
        ("decision_power", decision_power), ("engagement", engagement), ("closing", closing),
    ]:
        if "error" in dim:
            logger.error("[MeetAnalysis] dimension %s failed job=%s", name, job_id)
            return {"error": f"analyze_dimensions_{name}_failed"}

    logger.info("[MeetAnalysis] analyze_dimensions complete job=%s", job_id)
    return {
        "business": business,
        "gaps": gaps,
        "urgency": urgency,
        "decision_power": decision_power,
        "engagement": engagement,
        "closing": closing,
    }


# ── LangGraph Node 2: build_synthesis ────────────────────────────────────


async def build_synthesis(state: MeetAnalysisState) -> dict:
    lead = state.get("lead", {})
    contact = state.get("contact", {})
    trace = state.get("_trace")
    job_id = state.get("job_id", "?")

    user = _SYNTHESIS_USER.format(
        business_name=lead.get("businessName", "N/A"),
        contact_name=contact.get("name", "N/A"),
        contact_role=contact.get("role", "N/A"),
        business_json=json.dumps(state.get("business", {}), ensure_ascii=False, indent=2),
        gaps_json=json.dumps(state.get("gaps", {}), ensure_ascii=False, indent=2),
        urgency_json=json.dumps(state.get("urgency", {}), ensure_ascii=False, indent=2),
        decision_json=json.dumps(state.get("decision_power", {}), ensure_ascii=False, indent=2),
        engagement_json=json.dumps(state.get("engagement", {}), ensure_ascii=False, indent=2),
        closing_json=json.dumps(state.get("closing", {}), ensure_ascii=False, indent=2),
    )

    gen = (
        trace.generation(name="diag_synthesis", model=_SONNET, input=[{"role": "user", "content": user}])
        if trace else None
    )

    try:
        raw = await _call_claude(system=_SYNTHESIS_SYSTEM, user=user, model=_SONNET, max_tokens=1500)
        text = raw.content[0].text if raw.content else "{}"
        parsed = _parse_json(text)

        summary = parsed.get("summary", "")
        next_step = parsed.get("nextStep", "")
        positive_points = parsed.get("positivePoints", [])
        improvement_points = parsed.get("improvementPoints", [])

        if gen:
            gen.end(output={"summary_len": len(summary), "improvements": len(improvement_points)})

        logger.info("[MeetAnalysis] build_synthesis complete job=%s", job_id)
        return {
            "summary": summary,
            "next_step": next_step,
            "positive_points": positive_points,
            "improvement_points": improvement_points,
        }

    except Exception:
        logger.exception("[MeetAnalysis] build_synthesis failed job=%s", job_id)
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "build_synthesis_failed"}


# ── LangGraph Node 3: compute_scores ─────────────────────────────────────


async def compute_scores(state: MeetAnalysisState) -> dict:
    dim_scores = {
        "business":       (state.get("business") or {}).get("score", 0),
        "gaps":           (state.get("gaps") or {}).get("score", 0),
        "urgency":        (state.get("urgency") or {}).get("score", 0),
        "decision_power": (state.get("decision_power") or {}).get("score", 0),
        "engagement":     (state.get("engagement") or {}).get("score", 0),
        "closing":        (state.get("closing") or {}).get("score", 0),
    }
    score = _compute_overall_score(dim_scores)
    closing_probability = int((state.get("closing") or {}).get("closingProbability", 0))

    logger.info(
        "[MeetAnalysis] compute_scores job=%s score=%.1f closing_prob=%d",
        state.get("job_id", "?"), score, closing_probability,
    )
    return {"score": score, "closing_probability": closing_probability}


# ── LangGraph Node 4: send_webhook ────────────────────────────────────────


async def send_webhook(state: MeetAnalysisState) -> dict:
    job_id = state.get("job_id", "")
    webhook_url = state.get("webhook_url", "")
    error = state.get("error")

    if not webhook_url:
        logger.error("[MeetAnalysis] no webhook_url — result lost for job=%s", job_id)
        return {}

    if error:
        payload: dict = {"jobId": job_id, "status": "error", "error": error}
    else:
        payload = {
            "jobId": job_id,
            "status": "completed",
            "score": state.get("score"),
            "summary": state.get("summary"),
            "nextStep": state.get("next_step"),
            "positivePoints": state.get("positive_points"),
            "improvementPoints": state.get("improvement_points"),
            "closingProbability": state.get("closing_probability"),
            "diag": {
                "business":      state.get("business"),
                "gaps":          state.get("gaps"),
                "urgency":       state.get("urgency"),
                "decisionPower": state.get("decision_power"),
                "engagement":    state.get("engagement"),
                "closing":       state.get("closing"),
            },
        }

    try:
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.post(webhook_url, json=payload, headers=_auth_headers())
            logger.info(
                "[MeetAnalysis] webhook sent job=%s status=%d error=%s",
                job_id, resp.status_code, error,
            )
    except Exception:
        logger.exception("[MeetAnalysis] webhook send failed job=%s url=%s", job_id, webhook_url)

    return {}
