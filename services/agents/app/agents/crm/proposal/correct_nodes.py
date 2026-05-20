"""Correction and revision pipeline for existing proposals.

Downloads the PDF from Drive, extracts text, applies LLM changes, regenerates PDF.
"""
from __future__ import annotations

import asyncio
import base64
import logging
import re

import httpx

from app.agents.crm.nodes import _call_claude
from app.agents.crm.proposal.nodes import (
    _auth_headers,
    _html_to_pdf_sync,
    _parse_json,
    _resolve_webhook_url,
    _BRAND_PLACEHOLDER_MAP,
    _TEMPLATE_PATHS,
)

logger = logging.getLogger("agents")

_SONNET = "claude-sonnet-4-6"


# ── PDF helpers ───────────────────────────────────────────────────────────

def _drive_download_url(url: str) -> str:
    """Convert a Google Drive view/share URL to a direct download URL."""
    m = re.search(r'/d/([a-zA-Z0-9_-]+)', url)
    if m:
        return f"https://drive.google.com/uc?export=download&id={m.group(1)}"
    return url


async def _download_pdf_bytes(url: str) -> bytes:
    download_url = _drive_download_url(url)
    async with httpx.AsyncClient(timeout=30, follow_redirects=True) as client:
        resp = await client.get(download_url)
        resp.raise_for_status()
        return resp.content


def _extract_text_from_pdf_sync(pdf_bytes: bytes) -> str:
    from pypdf import PdfReader
    import io
    reader = PdfReader(io.BytesIO(pdf_bytes))
    pages = [page.extract_text() or "" for page in reader.pages]
    return "\n\n".join(p for p in pages if p.strip())


def _extract_proposal_number(text: str, brand: str) -> str | None:
    prefix = "SALTO" if brand == "salto" else "WB"
    m = re.search(rf'({prefix}-P\d+-\d+(?:-REV\d+)?)', text, re.IGNORECASE)
    return m.group(1) if m else None


def _make_revision_number(original: str, rev: int) -> str:
    base = re.sub(r'-REV\d+$', '', original, flags=re.IGNORECASE)
    return f"{base}-REV{rev}"


# ── LLM prompts ───────────────────────────────────────────────────────────

_CORRECT_SYSTEM = """\
Você aplica correções cirúrgicas em uma proposta comercial da WB Digital Solutions.

Receberá o texto completo extraído do PDF e as correções solicitadas.

REGRAS:
- Aplique APENAS as correções especificadas. Todo o restante deve ser idêntico ao original.
- Seções opcionais ausentes no original → retorne "".
- Mantenha as classes CSS do template (section-title, subsection-title, investment-table, total-row, payment-list, sprint-list, sprint-badge, sprint-content, etc.).
- Não invente conteúdo além do pedido.
- GREETING: saudação sem vírgula final (ex: "Prezado Sr. João Silva").

Retorne APENAS JSON válido sem markdown:
{
  "PROJECT_TITLE": "texto",
  "GREETING": "saudação sem vírgula",
  "OPENING_PARAGRAPH_1": "texto",
  "OPENING_PARAGRAPH_2": "texto",
  "SCOPE_INTRO": "texto",
  "CLOSING_PARAGRAPH_1": "texto",
  "TECH_SECTIONS": "(HTML) ou \"\"",
  "FEATURES_LIST": "(HTML ul>li) ou \"\"",
  "PLANNING_SECTION": "(HTML bloco completo com h3.subsection-title) ou \"\"",
  "METHODOLOGY_SECTION": "(HTML bloco completo com h2.section-title) ou \"\"",
  "POST_LAUNCH_SECTION": "(HTML bloco completo com h2.section-title) ou \"\"",
  "PENALTY_CLAUSE": "(HTML bloco completo com h2.section-title) ou \"\"",
  "LGPD_SECTION": "(HTML bloco completo com h2.section-title) ou \"\"",
  "FUTURE_EVOLUTION_SECTION": "(HTML bloco completo com h2.section-title) ou \"\"",
  "INVESTMENT_TABLE": "(HTML table.investment-table completa) ou \"\"",
  "PAYMENT_SCHEDULE": "(HTML ol.payment-list) ou \"\"",
  "ADDITIONAL_CONSIDERATIONS_ITEMS": "(HTML um ou mais li) ou \"\""
}"""

_REVISE_SYSTEM = """\
Você gera uma versão revisada de uma proposta comercial da WB Digital Solutions após negociação.

Receberá o texto completo da proposta original e as alterações negociadas.

REGRAS:
- Aplique TODAS as mudanças negociadas (valores, prazos, escopo, pagamento).
- Mantenha o conteúdo não mencionado idêntico ao original.
- Seções opcionais ausentes no original → retorne "" (salvo se a revisão as adicionar explicitamente).
- Mantenha as classes CSS do template.
- Valores: formato brasileiro R$ X.XXX,00.
- INVESTMENT_TABLE: itens com R$ 0,00 + total-row com o valor real negociado.
- GREETING: saudação sem vírgula final.

Retorne APENAS JSON válido sem markdown:
{
  "PROJECT_TITLE": "texto",
  "GREETING": "saudação sem vírgula",
  "OPENING_PARAGRAPH_1": "texto",
  "OPENING_PARAGRAPH_2": "texto",
  "SCOPE_INTRO": "texto",
  "CLOSING_PARAGRAPH_1": "texto",
  "TECH_SECTIONS": "(HTML) ou \"\"",
  "FEATURES_LIST": "(HTML ul>li) ou \"\"",
  "PLANNING_SECTION": "(HTML bloco completo com h3.subsection-title) ou \"\"",
  "METHODOLOGY_SECTION": "(HTML bloco completo com h2.section-title) ou \"\"",
  "POST_LAUNCH_SECTION": "(HTML bloco completo com h2.section-title) ou \"\"",
  "PENALTY_CLAUSE": "(HTML bloco completo com h2.section-title) ou \"\"",
  "LGPD_SECTION": "(HTML bloco completo com h2.section-title) ou \"\"",
  "FUTURE_EVOLUTION_SECTION": "(HTML bloco completo com h2.section-title) ou \"\"",
  "INVESTMENT_TABLE": "(HTML table.investment-table — itens R$ 0,00 + total-row valor real) ou \"\"",
  "PAYMENT_SCHEDULE": "(HTML ol.payment-list) ou \"\"",
  "ADDITIONAL_CONSIDERATIONS_ITEMS": "(HTML um ou mais li) ou \"\""
}"""


async def _call_correction_llm(extracted_text: str, instructions: str, mode: str) -> dict:
    system = _CORRECT_SYSTEM if mode == "correct" else _REVISE_SYSTEM
    label = "CORREÇÕES A APLICAR" if mode == "correct" else "ALTERAÇÕES DA REVISÃO"
    user = (
        f"PROPOSTA ATUAL (texto extraído do PDF):\n{extracted_text[:12000]}"
        f"\n\n{label}:\n{instructions}"
    )
    resp = await _call_claude(system, user, model=_SONNET, max_tokens=4096)
    return _parse_json(resp.content[0].text)


# ── Main pipeline ─────────────────────────────────────────────────────────

async def run_correction_pipeline(
    job_id: str,
    proposal_id: str,
    drive_url: str,
    instructions: str,
    brand: str,
    lead: dict,
    webhook_url: str,
    mode: str = "correct",      # "correct" | "revise"
    revision_number: int = 1,
) -> None:
    """Download → extract → LLM → fill template → PDF → webhook."""
    webhook_url = _resolve_webhook_url(webhook_url)
    company = re.sub(r"[^\w]", "", lead.get("businessName", "Proposta"))[:20]
    loop = asyncio.get_event_loop()

    try:
        # 1 — Download PDF from Drive
        logger.info("[Proposal/%s] downloading PDF job=%s url=%s", mode, job_id, drive_url)
        pdf_bytes = await _download_pdf_bytes(drive_url)

        # 2 — Extract text
        extracted_text = await loop.run_in_executor(None, _extract_text_from_pdf_sync, pdf_bytes)
        if not extracted_text.strip():
            raise ValueError("PDF sem texto extraível (pode ser escaneado ou protegido)")
        logger.info("[Proposal/%s] extracted %d chars job=%s", mode, len(extracted_text), job_id)

        # 3 — Proposal number
        original_number = _extract_proposal_number(extracted_text, brand) or "WB-P001-0526"
        proposal_number = (
            _make_revision_number(original_number, revision_number)
            if mode == "revise"
            else original_number
        )

        # 4 — Apply corrections via LLM
        sections = await _call_correction_llm(extracted_text, instructions, mode)
        sections["PROPOSAL_NUMBER"] = proposal_number

        # 5 — Fill HTML template
        template_path = _TEMPLATE_PATHS.get(brand, _TEMPLATE_PATHS["wb"])
        with open(template_path, encoding="utf-8") as f:
            template = f.read()

        placeholder_map = _BRAND_PLACEHOLDER_MAP.get(brand, _BRAND_PLACEHOLDER_MAP["wb"])
        filled = template
        for key, value in sections.items():
            tpl_key = placeholder_map.get(key, key)
            filled = filled.replace(f"{{{{{tpl_key}}}}}", str(value) if value else "")

        # 6 — Generate PDF
        pdf_out = await loop.run_in_executor(None, _html_to_pdf_sync, filled)
        pdf_base64 = base64.b64encode(pdf_out).decode("utf-8")
        logger.info("[Proposal/%s] PDF generated job=%s size=%d", mode, job_id, len(pdf_out))

        from datetime import datetime, timezone
        now = datetime.now(timezone.utc)
        prefix = "Proposta_Salto" if brand == "salto" else "Proposta_WB"
        suffix = f"REV{revision_number}" if mode == "revise" else "Corrigida"
        file_name = f"{prefix}_{company}_{suffix}_{now.strftime('%m%y')}.pdf"

        # 7 — Send completed webhook
        async with httpx.AsyncClient(timeout=30) as client:
            await client.post(webhook_url, json={
                "jobId": job_id,
                "proposalId": proposal_id,
                "status": "completed",
                "fileName": file_name,
                "fileSize": len(pdf_out),
                "fileBase64": pdf_base64,
            }, headers=_auth_headers())
        logger.info("[Proposal/%s] completed job=%s number=%s", mode, job_id, proposal_number)

    except Exception as exc:
        logger.exception("[Proposal/%s] pipeline failed job=%s", mode, job_id)
        try:
            async with httpx.AsyncClient(timeout=15) as client:
                await client.post(webhook_url, json={
                    "jobId": job_id,
                    "proposalId": proposal_id,
                    "status": "error",
                    "errorMessage": str(exc),
                }, headers=_auth_headers())
        except Exception:
            logger.exception("[Proposal/%s] error webhook also failed job=%s", mode, job_id)
