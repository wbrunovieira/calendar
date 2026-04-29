from __future__ import annotations

import json
import logging
import re

import httpx

from app.agents.crm.nodes import _call_claude
from app.agents.crm.call_analysis.state import CallAnalysisState
from app.agents.finances.nodes import langfuse
from app.config import settings

logger = logging.getLogger("agents")

# ── Prompts: Node 1 — analyze_spiced (Sonnet) ─────────────────────────────

_SPICED_SYSTEM = """\
Você é um coach especializado em vendas B2B SaaS analisando ligações de SDR com o método SPICED.
SPICED = Situation · Pain · Impact · Critical Event · Evidence.

MICRO-PACTOS — 7 checkpoints de comprometimento progressivo que o SDR deve construir:
1. [S] Lead confirma cadastro — SDR fez o lead confirmar/validar o perfil da empresa
2. [S] Lead autoriza perguntas — SDR pediu permissão explícita para fazer perguntas
3. [S] Lead descreve negócio — Lead descreveu voluntariamente como a empresa funciona
4. [P] Lead nomeia e confirma dor duas vezes — Lead nomeou a dor E confirmou em ao menos 2 momentos distintos
5. [I] Lead quantifica custo da dor — Lead deu número concreto (R$, clientes perdidos, tempo)
6. [C] Lead declara urgência — SDR criou senso de urgência e lead verbalizou que precisa resolver
7. [C] Lead confirma compromisso de agendamento — Data/horário fixo confirmado OU micro-comprometimento explícito

TÉCNICAS DE AGENDAMENTO:
- gatilhoDor: SDR reconectou o agendamento à dor antes de propor horário ("para resolver X, precisamos de uma reunião...")
- escolhaAlternativa: Usou "quando" (não "se") + ofereceu alternativas ("terça ou quarta?")
- compromissoVerbalShow: Lead verbalizou comprometimento explícito com o próximo passo
- compromissoEmergencia: SDR pediu aviso em caso de imprevisto ("se surgir algo, você me avisa?")

REGRAS DO SDR que você deve avaliar:
- Regra 2/3: não apresentar solução antes de confirmar dor pelo menos 2x
- Pedir permissão antes de perguntar
- Explorar impacto financeiro/operacional antes de propor agendamento
- Nunca usar "se" ao agendar — usar "quando"
- Identificar evento crítico com prazo ou gatilho concreto

Retorne APENAS JSON válido, sem markdown, sem explicações."""

_SPICED_USER = """\
CONTEXTO DO LEAD:
Empresa: {business_name}
Descrição: {description}
Segmento: {segment}
Cidade: {city}
Atividades: {activities}

CONTATO: {contact_name} — {contact_role}
Duração: {duration_min} minutos | Data: {call_date}

TRANSCRIÇÃO:
{transcript}

Analise a ligação e retorne JSON com esta estrutura exata:
{{
  "spiced": {{
    "situation":     {{"score": 0-10, "text": "O que o SDR descobriu sobre a situação. O que faltou descobrir."}},
    "pain":          {{"score": 0-10, "text": "Dores identificadas e quantas vezes confirmadas. O que faltou."}},
    "impact":        {{"score": 0-10, "text": "Impacto financeiro/operacional explorado. Números citados pelo lead."}},
    "criticalEvent": {{"score": 0-10, "text": "Evento crítico ou prazo identificado. O que faltou criar/explorar."}},
    "evidence":      {{"score": 0-10, "text": "Autoridade e decisão confirmadas. Processo de compra explorado."}}
  }},
  "microPactos": [
    {{"id": 1, "label": "Lead confirma cadastro",                   "spicedDimension": "S", "achieved": true, "notes": "evidência exata ou o que o SDR deveria ter feito"}},
    {{"id": 2, "label": "Lead autoriza perguntas",                  "spicedDimension": "S", "achieved": false, "notes": "..."}},
    {{"id": 3, "label": "Lead descreve negócio",                    "spicedDimension": "S", "achieved": true, "notes": "..."}},
    {{"id": 4, "label": "Lead nomeia e confirma dor duas vezes",    "spicedDimension": "P", "achieved": false, "notes": "..."}},
    {{"id": 5, "label": "Lead quantifica custo da dor",             "spicedDimension": "I", "achieved": true, "notes": "..."}},
    {{"id": 6, "label": "Lead declara urgência",                    "spicedDimension": "C", "achieved": false, "notes": "..."}},
    {{"id": 7, "label": "Lead confirma compromisso de agendamento", "spicedDimension": "C", "achieved": false, "notes": "..."}}
  ],
  "schedulingTechniques": {{
    "gatilhoDor":            {{"used": false, "notes": "evidência exata se used=true, ou o que deveria ter sido dito"}},
    "escolhaAlternativa":    {{"used": false, "notes": "..."}},
    "compromissoVerbalShow": {{"used": false, "notes": "..."}},
    "compromissoEmergencia": {{"used": false, "notes": "..."}}
  }}
}}

IMPORTANTE:
- Nos "notes" dos microPactos: quando achieved=true, cite a fala exata da transcrição. Quando achieved=false, descreva o que o SDR deveria ter feito.
- Scores: seja preciso, não arredonde tudo para 5 ou 10. Use decimais (ex: 6.5, 8.0).
- Os 7 microPactos devem sempre aparecer na ordem exata acima com os ids 1-7."""


# ── Prompts: Node 2 — build_analysis (Haiku) ──────────────────────────────

_ANALYSIS_SYSTEM = """\
Você é um coach de vendas B2B SaaS. Analise a transcrição e o resultado SPICED fornecido.
Retorne APENAS JSON válido, sem markdown."""

_ANALYSIS_USER = """\
EMPRESA: {business_name} | CONTATO: {contact_name} | DURAÇÃO: {duration_min}min

AVALIAÇÃO SPICED (já calculada):
{spiced_json}

MICRO-PACTOS: {achieved_count}/7 atingidos | TÉCNICAS DE AGENDAMENTO: {techniques_used}/4 usadas

TRANSCRIÇÃO:
{transcript}

Retorne JSON:
{{
  "summary": "2-3 frases resumindo a ligação, a qualificação do lead e o resultado do agendamento.",
  "microAnalysis": [
    {{"timestamp": "MM:SS", "text": "descrição concisa do momento-chave [S/P/I/C/E]", "type": "positive"}},
    {{"timestamp": "MM:SS", "text": "...", "type": "warning"}},
    {{"timestamp": "MM:SS", "text": "...", "type": "negative"}}
  ],
  "positivePoints": [
    "ponto positivo específico e acionável",
    "..."
  ],
  "improvementPoints": [
    "melhoria específica e acionável — o que fazer diferente",
    "..."
  ]
}}

REGRAS:
- microAnalysis: 6-10 momentos, timestamps EXATOS da transcrição (MM:SS ou HH:MM:SS), inclua positivos E negativos
- positivePoints: 2-4 itens, cite comportamentos concretos do SDR que funcionaram
- improvementPoints: 2-4 itens, cite o erro exato e como corrigir na próxima ligação
- summary: não repita os pontos — seja sintético e direto"""


# ── Helpers ────────────────────────────────────────────────────────────────

def _parse_json(text: str) -> dict:
    text = re.sub(r"^```(?:json)?\s*", "", text.strip())
    text = re.sub(r"\s*```$", "", text.strip())
    return json.loads(text)


def _compute_scores(
    spiced: dict,
    micro_pactos: list,
    scheduling_techniques: dict,
) -> tuple[float, str, str]:
    """Deterministic: overall score, no-show risk level and risk text."""
    dim_scores = [
        spiced.get("situation", {}).get("score", 0),
        spiced.get("pain", {}).get("score", 0),
        spiced.get("impact", {}).get("score", 0),
        spiced.get("criticalEvent", {}).get("score", 0),
        spiced.get("evidence", {}).get("score", 0),
    ]
    overall_score = round(sum(dim_scores) / max(len(dim_scores), 1), 1)

    achieved_count = sum(1 for mp in micro_pactos if mp.get("achieved"))
    techniques_used = sum(
        1 for t in (scheduling_techniques or {}).values() if t.get("used")
    )

    if achieved_count >= 5:
        risk = "BAIXO"
    elif achieved_count >= 3:
        risk = "MÉDIO"
    else:
        risk = "ALTO"

    achieved_dims = sorted({mp["spicedDimension"] for mp in micro_pactos if mp.get("achieved")})
    missing_dims = sorted({mp["spicedDimension"] for mp in micro_pactos if not mp.get("achieved")})

    risk_text = (
        f"Micro-pactos {achieved_count}/7 construídos "
        f"(fortes em {'/'.join(achieved_dims) or 'nenhum'}, "
        f"fracos em {'/'.join(missing_dims) or 'nenhum'}). "
        f"Técnicas de agendamento: {techniques_used}/4 usadas."
    )

    return overall_score, risk, risk_text


def _auth_headers() -> dict[str, str]:
    headers: dict[str, str] = {"Content-Type": "application/json"}
    if settings.crm_webhook_secret:
        headers["X-Webhook-Secret"] = settings.crm_webhook_secret
    return headers


# ── Node 1: analyze_spiced ─────────────────────────────────────────────────


async def analyze_spiced(state: CallAnalysisState) -> dict:
    lead = state.get("lead", {})
    contact = state.get("contact", {})
    transcript = state.get("transcript", "")
    duration_min = (state.get("call_duration_seconds") or 0) // 60
    call_date = state.get("call_date", "")
    trace = state.get("_trace")

    user = _SPICED_USER.format(
        business_name=lead.get("businessName", "N/A"),
        description=lead.get("description", "N/A"),
        segment=lead.get("segment", "N/A"),
        city=lead.get("city", "N/A"),
        activities=lead.get("activities", "N/A"),
        contact_name=contact.get("name", "N/A"),
        contact_role=contact.get("role", "N/A"),
        duration_min=duration_min,
        call_date=call_date,
        transcript=transcript[:14000],
    )

    gen = (
        trace.generation(
            name="analyze_spiced",
            model="claude-sonnet-4-6",
            input=[{"role": "user", "content": user}],
        )
        if trace else None
    )

    try:
        raw = await _call_claude(
            system=_SPICED_SYSTEM,
            user=user,
            model="claude-sonnet-4-6",
            max_tokens=3000,
        )
        text = raw.content[0].text if raw.content else "{}"
        parsed = _parse_json(text)

        spiced = parsed.get("spiced", {})
        micro_pactos = parsed.get("microPactos", [])
        scheduling = parsed.get("schedulingTechniques", {})

        logger.info(
            "[CallAnalysis] job=%s spiced_dims=%d micro_pactos=%d scheduling=%d",
            state.get("job_id"), len(spiced), len(micro_pactos), len(scheduling),
        )

        if gen:
            achieved = sum(1 for mp in micro_pactos if mp.get("achieved"))
            gen.end(output={"dims": list(spiced.keys()), "achieved": achieved})

        return {"spiced": spiced, "micro_pactos": micro_pactos, "scheduling_techniques": scheduling}

    except Exception:
        logger.exception("[CallAnalysis] analyze_spiced error job=%s", state.get("job_id"))
        if gen:
            gen.end(output={"error": "parse_failed"})
        return {"error": "analyze_spiced_failed"}


# ── Node 2: build_analysis ─────────────────────────────────────────────────


async def build_analysis(state: CallAnalysisState) -> dict:
    lead = state.get("lead", {})
    contact = state.get("contact", {})
    transcript = state.get("transcript", "")
    duration_min = (state.get("call_duration_seconds") or 0) // 60
    spiced = state.get("spiced", {})
    micro_pactos = state.get("micro_pactos", [])
    scheduling = state.get("scheduling_techniques", {})
    trace = state.get("_trace")

    achieved_count = sum(1 for mp in micro_pactos if mp.get("achieved"))
    techniques_used = sum(1 for t in scheduling.values() if t.get("used")) if scheduling else 0

    user = _ANALYSIS_USER.format(
        business_name=lead.get("businessName", "N/A"),
        contact_name=contact.get("name", "N/A"),
        duration_min=duration_min,
        spiced_json=json.dumps(spiced, ensure_ascii=False, indent=2),
        achieved_count=achieved_count,
        techniques_used=techniques_used,
        transcript=transcript[:10000],
    )

    gen = (
        trace.generation(
            name="build_analysis",
            model="claude-haiku-4-5-20251001",
            input=[{"role": "user", "content": user}],
        )
        if trace else None
    )

    try:
        raw = await _call_claude(
            system=_ANALYSIS_SYSTEM,
            user=user,
            model="claude-haiku-4-5-20251001",
            max_tokens=1500,
        )
        text = raw.content[0].text if raw.content else "{}"
        parsed = _parse_json(text)

        summary = parsed.get("summary", "")
        micro_analysis = parsed.get("microAnalysis", [])
        positive_points = parsed.get("positivePoints", [])
        improvement_points = parsed.get("improvementPoints", [])

        if gen:
            gen.end(output={"moments": len(micro_analysis), "summary_len": len(summary)})

        return {
            "summary": summary,
            "micro_analysis": micro_analysis,
            "positive_points": positive_points,
            "improvement_points": improvement_points,
        }

    except Exception:
        logger.exception("[CallAnalysis] build_analysis error job=%s", state.get("job_id"))
        if gen:
            gen.end(output={"error": "parse_failed"})
        return {"error": "build_analysis_failed"}


# ── Node 3: compute_scores ─────────────────────────────────────────────────


async def compute_scores(state: CallAnalysisState) -> dict:
    spiced = state.get("spiced") or {}
    micro_pactos = state.get("micro_pactos") or []
    scheduling = state.get("scheduling_techniques") or {}

    score, risk, risk_text = _compute_scores(spiced, micro_pactos, scheduling)

    logger.info(
        "[CallAnalysis] job=%s score=%.1f risk=%s",
        state.get("job_id"), score, risk,
    )

    return {"score": score, "no_show_risk": risk, "no_show_risk_text": risk_text}


# ── Node 4: send_webhook ───────────────────────────────────────────────────


async def send_webhook(state: CallAnalysisState) -> dict:
    job_id = state.get("job_id", "")
    webhook_url = state.get("webhook_url", "")
    error = state.get("error")

    if error:
        payload: dict = {"jobId": job_id, "status": "error", "error": error}
    else:
        payload = {
            "jobId": job_id,
            "status": "completed",
            "score": state.get("score"),
            "noShowRisk": state.get("no_show_risk"),
            "noShowRiskText": state.get("no_show_risk_text"),
            "summary": state.get("summary"),
            "spiced": state.get("spiced"),
            "microPactos": state.get("micro_pactos"),
            "schedulingTechniques": state.get("scheduling_techniques"),
            "microAnalysis": state.get("micro_analysis"),
            "positivePoints": state.get("positive_points"),
            "improvementPoints": state.get("improvement_points"),
        }

    if not webhook_url:
        logger.error("[CallAnalysis] no webhook_url — result lost for job=%s", job_id)
        return {}

    try:
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.post(webhook_url, json=payload, headers=_auth_headers())
            logger.info(
                "[CallAnalysis] webhook sent job=%s status=%d error=%s",
                job_id, resp.status_code, error,
            )
    except Exception:
        logger.exception("[CallAnalysis] webhook send failed job=%s url=%s", job_id, webhook_url)

    return {}
