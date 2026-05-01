from __future__ import annotations

import asyncio
import json
import logging
import re

import httpx

from app.agents.crm.nodes import _call_claude
from app.agents.crm.gatekeeper_analysis.state import GatekeeperAnalysisState
from app.agents.finances.nodes import langfuse
from app.config import settings

logger = logging.getLogger("agents")

_HAIKU = "claude-haiku-4-5-20251001"
_SONNET = "claude-sonnet-4-6"

_TRANSCRIPT_LIMIT = 8_000


# ── Helpers ───────────────────────────────────────────────────────────────

def _context_block(lead: dict, contact: dict, duration_sec: int, call_date: str) -> str:
    minutes = (duration_sec or 0) // 60
    return (
        f"EMPRESA: {lead.get('businessName', 'N/A')} | "
        f"SEGMENTO: {lead.get('segment', 'N/A')} | "
        f"CIDADE: {lead.get('city', 'N/A')}\n"
        f"PORTEIRO: {contact.get('name', 'N/A')} — {contact.get('role', 'N/A')}\n"
        f"DURAÇÃO: {minutes} minutos | DATA: {call_date or 'N/A'}\n"
    )


def _parse_json(text: str) -> dict:
    text = re.sub(r"^```(?:json)?\s*", "", text.strip())
    text = re.sub(r"\s*```$", "", text.strip())
    return json.loads(text)


def _auth_headers() -> dict[str, str]:
    headers: dict[str, str] = {"Content-Type": "application/json"}
    if settings.crm_webhook_secret:
        headers["X-Webhook-Secret"] = settings.crm_webhook_secret
    return headers


# ── R — Recepção ──────────────────────────────────────────────────────────

_RECEPCAO_SYSTEM = """\
Você é especialista em treinamento de SDRs de alta performance. Analise a transcrição de uma ligação
para porteiro/recepcionista e avalie a qualidade da abertura do SDR.
Foco: profissionalismo, ausência de "cheiro de vendedor", tom de voz descrito, fluidez da abertura.
Retorne APENAS JSON válido, sem markdown, sem explicações."""

_RECEPCAO_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "text": "análise da recepção em 2-3 frases",
  "score": 1-5
}}

Score: 5 = abertura impecável e profissional, 1 = abertura com cheiro de vendedor ou agressiva."""


async def _analyze_recepcao(
    lead: dict, contact: dict, transcript: str,
    duration_sec: int, call_date: str, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_sec, call_date)
    user = _RECEPCAO_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])
    gen = (
        trace.generation(name="raport_recepcao", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_RECEPCAO_SYSTEM, user=user, model=_HAIKU, max_tokens=400)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score")})
        return result
    except Exception:
        logger.exception("[GKAnalysis] _analyze_recepcao failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_recepcao_failed"}


# ── A — Aliança ───────────────────────────────────────────────────────────

_ALIANCA_SYSTEM = """\
Você é especialista em rapport e inteligência emocional para vendas. Analise a transcrição e avalie
como o SDR construiu aliança com o porteiro/recepcionista.
Foco: uso do nome da pessoa, respeito genuíno, momentos de conexão humana.
Retorne APENAS JSON válido, sem markdown."""

_ALIANCA_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "text": "análise da aliança em 2-3 frases",
  "score": 1-5,
  "connectionMoments": ["momento de conexão 1", "momento de conexão 2"]
}}

connectionMoments: momentos específicos em que o SDR criou conexão humana com o porteiro (pode ser lista vazia).
Score: 5 = aliança forte e genuína, 1 = relacionamento frio ou transacional."""


async def _analyze_alianca(
    lead: dict, contact: dict, transcript: str,
    duration_sec: int, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_sec, "")
    user = _ALIANCA_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])
    gen = (
        trace.generation(name="raport_alianca", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_ALIANCA_SYSTEM, user=user, model=_HAIKU, max_tokens=500)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score")})
        return result
    except Exception:
        logger.exception("[GKAnalysis] _analyze_alianca failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_alianca_failed"}


# ── P — Perguntas ─────────────────────────────────────────────────────────

_PERGUNTAS_SYSTEM = """\
Você é especialista em técnicas de prospecção. Analise a transcrição e avalie as perguntas feitas
pelo SDR ao porteiro/recepcionista.
Foco: perguntas que empoderam o porteiro (fazem ele se sentir importante), abertas, consultivas.
Retorne APENAS JSON válido, sem markdown."""

_PERGUNTAS_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "text": "análise das perguntas em 2-3 frases",
  "score": 1-5,
  "questionsAsked": ["pergunta 1 feita pelo SDR", "pergunta 2"]
}}

questionsAsked: perguntas literais do SDR ao porteiro.
Score: 5 = perguntas empoderadoras e estratégicas, 1 = nenhuma pergunta ou perguntas fechadas ineficazes."""


async def _analyze_perguntas(
    lead: dict, contact: dict, transcript: str,
    duration_sec: int, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_sec, "")
    user = _PERGUNTAS_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])
    gen = (
        trace.generation(name="raport_perguntas", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_PERGUNTAS_SYSTEM, user=user, model=_HAIKU, max_tokens=500)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score")})
        return result
    except Exception:
        logger.exception("[GKAnalysis] _analyze_perguntas failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_perguntas_failed"}


# ── O — Objeções ──────────────────────────────────────────────────────────

_OBJECOES_SYSTEM = """\
Você é especialista em gestão de objeções em ligações de prospecção. Analise a transcrição e avalie
as objeções recebidas pelo porteiro e como o SDR respondeu.
Objeções comuns: "manda email", "ele não está", "do que se trata?", "não temos interesse".
Retorne APENAS JSON válido, sem markdown."""

_OBJECOES_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "text": "análise das objeções em 2-3 frases",
  "score": 1-5,
  "objectionsReceived": ["objeção literal 1 do porteiro", "objeção 2"],
  "responsesGiven": ["resposta do SDR à objeção 1", "resposta à objeção 2"]
}}

objectionsReceived: objeções literais recebidas do porteiro.
responsesGiven: como o SDR respondeu a cada objeção (mesma ordem).
Score: 5 = respondeu todas as objeções com excelência, 1 = cedeu às objeções sem resposta ou nenhuma objeção recebida e SDR não estava preparado."""


async def _analyze_objecoes(
    lead: dict, contact: dict, transcript: str,
    duration_sec: int, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_sec, "")
    user = _OBJECOES_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])
    gen = (
        trace.generation(name="raport_objecoes", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_OBJECOES_SYSTEM, user=user, model=_HAIKU, max_tokens=600)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score")})
        return result
    except Exception:
        logger.exception("[GKAnalysis] _analyze_objecoes failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_objecoes_failed"}


# ── R — Resultado ─────────────────────────────────────────────────────────

_RESULTADO_SYSTEM = """\
Você é especialista em métricas de prospecção. Analise a transcrição e avalie o resultado concreto
obtido na ligação com o porteiro.
Foco: o que foi efetivamente conseguido — nome do decisor, transferência, agenda, etc.
Retorne APENAS JSON válido, sem markdown."""

_RESULTADO_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "text": "análise do resultado em 2-3 frases",
  "score": 1-5,
  "outcome": "got_transfer"|"got_name"|"got_meeting"|"callback_scheduled"|"rejected"|"voicemail"|"unknown",
  "obtained": ["informação 1 obtida", "informação 2"],
  "nextAttemptTip": "dica específica para a próxima tentativa"
}}

outcome: o resultado principal da ligação.
obtained: tudo que foi conseguido (nomes, cargos, emails, horários, informações sobre o decisor).
nextAttemptTip: dica concreta baseada no que foi descoberto (horário, abordagem, pessoa certa).
Score: 5 = transferido ao decisor ou reunião agendada, 1 = desligado sem informação alguma."""


async def _analyze_resultado(
    lead: dict, contact: dict, transcript: str,
    duration_sec: int, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_sec, "")
    user = _RESULTADO_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])
    gen = (
        trace.generation(name="raport_resultado", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_RESULTADO_SYSTEM, user=user, model=_HAIKU, max_tokens=600)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score"), "outcome": result.get("outcome")})
        return result
    except Exception:
        logger.exception("[GKAnalysis] _analyze_resultado failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_resultado_failed"}


# ── T — Técnicas ──────────────────────────────────────────────────────────

_TECNICAS_SYSTEM = """\
Você é especialista em técnicas de prospecção B2B. Analise a transcrição e identifique as técnicas
usadas pelo SDR na ligação com o porteiro.
Técnicas conhecidas: aliança com GK, urgência suave, nome do referenciador, reciprocidade,
"eu preciso da sua ajuda", ancoragem de próximo passo, bloqueio de manda-email.
Retorne APENAS JSON válido, sem markdown."""

_TECNICAS_USER = """\
{context}
TRANSCRIÇÃO:
{transcript}

Retorne JSON com esta estrutura exata:
{{
  "text": "análise das técnicas em 2-3 frases",
  "score": 1-5,
  "techniquesUsed": ["técnica 1 usada", "técnica 2"]
}}

techniquesUsed: técnicas identificadas que o SDR efetivamente usou.
Score: 5 = múltiplas técnicas avançadas bem aplicadas, 1 = nenhuma técnica identificada."""


async def _analyze_tecnicas(
    lead: dict, contact: dict, transcript: str,
    duration_sec: int, trace: object,
) -> dict:
    ctx = _context_block(lead, contact, duration_sec, "")
    user = _TECNICAS_USER.format(context=ctx, transcript=transcript[:_TRANSCRIPT_LIMIT])
    gen = (
        trace.generation(name="raport_tecnicas", model=_HAIKU, input=[{"role": "user", "content": user}])
        if trace else None
    )
    try:
        raw = await _call_claude(system=_TECNICAS_SYSTEM, user=user, model=_HAIKU, max_tokens=500)
        text = raw.content[0].text if raw.content else "{}"
        result = _parse_json(text)
        if gen:
            gen.end(output={"score": result.get("score")})
        return result
    except Exception:
        logger.exception("[GKAnalysis] _analyze_tecnicas failed")
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "analyze_tecnicas_failed"}


# ── LangGraph Node 1: analyze_dimensions ─────────────────────────────────


async def analyze_dimensions(state: GatekeeperAnalysisState) -> dict:
    lead = state.get("lead", {})
    contact = state.get("contact", {})
    transcript = state.get("transcript", "")
    duration_sec = state.get("call_duration_seconds", 0)
    call_date = state.get("call_date", "")
    trace = state.get("_trace")
    job_id = state.get("job_id", "?")

    try:
        results = await asyncio.gather(
            _analyze_recepcao(lead, contact, transcript, duration_sec, call_date, trace),
            _analyze_alianca(lead, contact, transcript, duration_sec, trace),
            _analyze_perguntas(lead, contact, transcript, duration_sec, trace),
            _analyze_objecoes(lead, contact, transcript, duration_sec, trace),
            _analyze_resultado(lead, contact, transcript, duration_sec, trace),
            _analyze_tecnicas(lead, contact, transcript, duration_sec, trace),
        )
    except Exception:
        logger.exception("[GKAnalysis] analyze_dimensions gather failed job=%s", job_id)
        return {"error": "analyze_dimensions_failed"}

    recepcao, alianca, perguntas, objecoes, resultado, tecnicas = results

    for name, dim in [
        ("recepcao", recepcao), ("alianca", alianca), ("perguntas", perguntas),
        ("objecoes", objecoes), ("resultado", resultado), ("tecnicas", tecnicas),
    ]:
        if "error" in dim:
            logger.error("[GKAnalysis] dimension %s failed job=%s", name, job_id)
            return {"error": f"analyze_dimensions_{name}_failed"}

    logger.info("[GKAnalysis] analyze_dimensions complete job=%s", job_id)
    return {
        "recepcao": recepcao,
        "alianca": alianca,
        "perguntas": perguntas,
        "objecoes": objecoes,
        "resultado": resultado,
        "tecnicas": tecnicas,
    }


# ── LangGraph Node 2: build_synthesis ────────────────────────────────────

_SYNTHESIS_SYSTEM = """\
Você é especialista em coaching de SDRs de prospecção ativa. Receberá os resultados das 6 análises
RAPORT de uma ligação para porteiro — NÃO a transcrição.
Sintetize em um resumo executivo e recomendações de coaching para o SDR.
Retorne APENAS JSON válido, sem markdown."""

_SYNTHESIS_USER = """\
EMPRESA: {business_name} | PORTEIRO: {contact_name} ({contact_role})

R — Recepção:
{recepcao_json}

A — Aliança:
{alianca_json}

P — Perguntas:
{perguntas_json}

O — Objeções:
{objecoes_json}

R — Resultado:
{resultado_json}

T — Técnicas:
{tecnicas_json}

Retorne JSON:
{{
  "summary": "resumo executivo em 2-4 frases: o que aconteceu, resultado e qualidade da abordagem",
  "positivePoints": [
    "ponto positivo 1 — comportamento específico do SDR",
    "..."
  ],
  "improvementPoints": [
    "melhoria 1 — erro específico + como corrigir na próxima ligação",
    "..."
  ]
}}

REGRAS:
- positivePoints: 2-4 itens, baseados nos dados das análises
- improvementPoints: 2-4 itens, com ação concreta de melhoria"""


async def build_synthesis(state: GatekeeperAnalysisState) -> dict:
    lead = state.get("lead", {})
    contact = state.get("contact", {})
    trace = state.get("_trace")
    job_id = state.get("job_id", "?")

    user = _SYNTHESIS_USER.format(
        business_name=lead.get("businessName", "N/A"),
        contact_name=contact.get("name", "N/A"),
        contact_role=contact.get("role", "N/A"),
        recepcao_json=json.dumps(state.get("recepcao", {}), ensure_ascii=False, indent=2),
        alianca_json=json.dumps(state.get("alianca", {}), ensure_ascii=False, indent=2),
        perguntas_json=json.dumps(state.get("perguntas", {}), ensure_ascii=False, indent=2),
        objecoes_json=json.dumps(state.get("objecoes", {}), ensure_ascii=False, indent=2),
        resultado_json=json.dumps(state.get("resultado", {}), ensure_ascii=False, indent=2),
        tecnicas_json=json.dumps(state.get("tecnicas", {}), ensure_ascii=False, indent=2),
    )

    gen = (
        trace.generation(name="raport_synthesis", model=_SONNET, input=[{"role": "user", "content": user}])
        if trace else None
    )

    try:
        raw = await _call_claude(system=_SYNTHESIS_SYSTEM, user=user, model=_SONNET, max_tokens=1200)
        text = raw.content[0].text if raw.content else "{}"
        parsed = _parse_json(text)

        summary = parsed.get("summary", "")
        positive_points = parsed.get("positivePoints", [])
        improvement_points = parsed.get("improvementPoints", [])

        if gen:
            gen.end(output={"summary_len": len(summary), "improvements": len(improvement_points)})

        logger.info("[GKAnalysis] build_synthesis complete job=%s", job_id)
        return {
            "summary": summary,
            "positive_points": positive_points,
            "improvement_points": improvement_points,
        }

    except Exception:
        logger.exception("[GKAnalysis] build_synthesis failed job=%s", job_id)
        if gen:
            gen.end(output={"error": "failed"})
        return {"error": "build_synthesis_failed"}


# ── LangGraph Node 3: compute_scores ─────────────────────────────────────


async def compute_scores(state: GatekeeperAnalysisState) -> dict:
    dims = ["recepcao", "alianca", "perguntas", "objecoes", "resultado", "tecnicas"]
    scores = [
        (state.get(d) or {}).get("score", 0)
        for d in dims
        if isinstance((state.get(d) or {}).get("score", 0), (int, float))
    ]
    score = round(sum(scores) / len(scores), 1) if scores else 0.0

    logger.info("[GKAnalysis] compute_scores job=%s score=%.1f", state.get("job_id", "?"), score)
    return {"score": score}


# ── LangGraph Node 4: send_webhook ────────────────────────────────────────


async def send_webhook(state: GatekeeperAnalysisState) -> dict:
    job_id = state.get("job_id", "")
    webhook_url = state.get("webhook_url", "")
    error = state.get("error")

    if not webhook_url:
        logger.error("[GKAnalysis] no webhook_url — result lost for job=%s", job_id)
        return {}

    if error:
        payload: dict = {"jobId": job_id, "status": "error", "error": error}
    else:
        payload = {
            "jobId": job_id,
            "status": "completed",
            "score": state.get("score"),
            "summary": state.get("summary"),
            "positivePoints": state.get("positive_points"),
            "improvementPoints": state.get("improvement_points"),
            "raport": {
                "recepcao":  state.get("recepcao"),
                "alianca":   state.get("alianca"),
                "perguntas": state.get("perguntas"),
                "objecoes":  state.get("objecoes"),
                "resultado": state.get("resultado"),
                "tecnicas":  state.get("tecnicas"),
            },
        }

    try:
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.post(webhook_url, json=payload, headers=_auth_headers())
            logger.info(
                "[GKAnalysis] webhook sent job=%s status=%d error=%s",
                job_id, resp.status_code, error,
            )
    except Exception:
        logger.exception("[GKAnalysis] webhook send failed job=%s url=%s", job_id, webhook_url)

    return {}
