from __future__ import annotations

import json
import logging
import re

import httpx

from app.agents.crm.nodes import _call_claude, _format_tavily_results, _tavily_search
from app.agents.crm.deep_research.state import DeepResearchState
from app.agents.finances.nodes import langfuse

logger = logging.getLogger("agents")

# ── Helpers ────────────────────────────────────────────────────────────────


def _is_empty(value) -> bool:
    if value is None:
        return True
    if isinstance(value, str) and not value.strip():
        return True
    return False


def _clean_cnpj(cnpj: str) -> str:
    return re.sub(r"\D", "", cnpj)


async def _fetch_cnpj(cnpj: str) -> dict | None:
    clean = _clean_cnpj(cnpj)
    if len(clean) != 14:
        return None
    try:
        async with httpx.AsyncClient(timeout=15) as client:
            resp = await client.get(f"https://brasilapi.com.br/api/cnpj/v1/{clean}")
            if resp.status_code == 200:
                return resp.json()
            logger.warning("BrasilAPI CNPJ returned %s for %s", resp.status_code, clean)
            return None
    except Exception:
        logger.warning("BrasilAPI CNPJ fetch failed for %s", clean)
        return None


def _extract_cnpj_updates(data: dict, lead: dict) -> dict:
    updates: dict = {}

    if _is_empty(lead.get("registeredName")):
        val = data.get("razao_social", "")
        if val:
            updates["registeredName"] = val.title()

    if _is_empty(lead.get("foundationDate")):
        val = data.get("data_abertura") or data.get("data_inicio_atividade", "")
        if val:
            updates["foundationDate"] = val

    if _is_empty(lead.get("legalNature")):
        val = data.get("natureza_juridica", "")
        if val:
            updates["legalNature"] = val

    if _is_empty(lead.get("companySize")):
        val = data.get("porte", "")
        if val:
            updates["companySize"] = val

    if _is_empty(lead.get("companyOwner")):
        qsa = data.get("qsa") or []
        admins = [
            m.get("nome_socio", "")
            for m in qsa
            if "administrador" in (m.get("qualificacao_socio", "") or "").lower()
            or "sócio-administrador" in (m.get("qualificacao_socio", "") or "").lower()
        ]
        if not admins:
            admins = [m.get("nome_socio", "") for m in qsa[:1]]
        if admins:
            updates["companyOwner"] = admins[0].title()

    if _is_empty(lead.get("address")):
        parts = [
            data.get("logradouro", ""),
            data.get("numero", ""),
            data.get("complemento", ""),
            data.get("bairro", ""),
        ]
        addr = ", ".join(p for p in parts if p)
        if addr:
            updates["address"] = addr

    if _is_empty(lead.get("city")):
        val = data.get("municipio", "")
        if val:
            updates["city"] = val.title()

    if _is_empty(lead.get("state")):
        val = data.get("uf", "")
        if val:
            updates["state"] = val

    if _is_empty(lead.get("zipCode")):
        val = data.get("cep", "")
        if val:
            updates["zipCode"] = val

    if _is_empty(lead.get("email")):
        val = data.get("email", "")
        if val and "@" in val:
            updates["email"] = val.lower()

    if _is_empty(lead.get("phone")):
        val = data.get("telefone", "")
        if val:
            updates["phone"] = val

    # simplesNacional / isMei
    simples = data.get("simples") or {}
    if lead.get("simplesNacional") is None and simples.get("optante") is not None:
        updates["simplesNacional"] = simples["optante"]

    if lead.get("isMei") is None:
        mei = data.get("mei") or {}
        if mei.get("optante") is not None:
            updates["isMei"] = mei["optante"]

    return updates


# ── Node: plan_research ────────────────────────────────────────────────────


async def plan_research(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    contacts = state.get("contacts", [])
    name = lead.get("businessName", "") or lead.get("registeredName", "")
    cnpj = lead.get("companyRegistrationID", "")

    missing: list[str] = []
    queries: list[str] = []

    # CNPJ data (legal, fiscal, address)
    legal_fields = ["legalNature", "companyOwner", "foundationDate", "companySize"]
    if cnpj and any(_is_empty(lead.get(f)) for f in legal_fields):
        missing.append("cnpj")

    # Website / description / services
    web_fields = ["website", "description", "segment"]
    if name and any(_is_empty(lead.get(f)) for f in web_fields):
        missing.append("web")
        queries.append(f'"{name}" site oficial serviços')
        if lead.get("city"):
            queries.append(f'"{name}" {lead["city"]} empresa')

    # Instagram
    if _is_empty(lead.get("instagram")) and name:
        missing.append("instagram")
        queries.append(f'"{name}" instagram perfil')

    # LinkedIn
    if _is_empty(lead.get("linkedin")) and name:
        missing.append("linkedin")
        queries.append(f'"{name}" linkedin empresa')

    # Facebook / TikTok
    social_missing = [
        s for s in ["facebook", "tiktok"]
        if _is_empty(lead.get(s)) and name
    ]
    if social_missing:
        missing.append("social")
        queries.append(f'"{name}" facebook tiktok redes sociais')

    # Contacts (email, phone, decision makers)
    needs_contacts = (
        not contacts
        or all(_is_empty(c.get("email")) and _is_empty(c.get("phone")) for c in contacts)
    )
    if needs_contacts and name:
        missing.append("contacts")
        queries.append(f'"{name}" contato email telefone diretor gerente')
        if lead.get("website"):
            queries.append(f'site:{lead["website"]} contato equipe')

    logger.info(
        "[DeepResearch] lead=%s missing=%s queries=%d",
        state.get("lead_id"), missing, len(queries),
    )

    return {"missing_fields": missing, "search_queries": queries}


# ── Node: run_searches ─────────────────────────────────────────────────────


async def run_searches(state: DeepResearchState) -> dict:
    missing = state.get("missing_fields", [])
    queries = state.get("search_queries", [])
    lead = state.get("lead", {})
    trace = state.get("_trace")

    updates: dict = {}
    cnpj_raw: dict | None = None

    # CNPJ lookup via BrasilAPI
    if "cnpj" in missing:
        cnpj = lead.get("companyRegistrationID", "")
        span = trace.span(name="cnpj_lookup", input={"cnpj": cnpj}) if trace else None
        cnpj_raw = await _fetch_cnpj(cnpj)
        if cnpj_raw:
            updates.update(_extract_cnpj_updates(cnpj_raw, lead))
            logger.info("[DeepResearch] CNPJ data retrieved for %s", cnpj)
        if span:
            span.end(output={"found": cnpj_raw is not None})

    # Tavily web searches
    web_results: list[dict] = []
    if queries:
        span = trace.span(name="web_searches", input={"queries": queries}) if trace else None
        web_results = await _tavily_search(queries, max_results=5, search_depth="basic")
        logger.info("[DeepResearch] Tavily returned %d result groups", len(web_results))
        if span:
            span.end(output={"groups": len(web_results)})

    return {
        "cnpj_raw": cnpj_raw,
        "web_results": web_results,
        "updates": updates,  # pre-populated with CNPJ data
    }


# ── Node: extract_updates ──────────────────────────────────────────────────


_EXTRACT_SYSTEM = """\
Você é um analista de dados B2B especializado em enriquecimento de leads.
Extraia informações sobre a empresa a partir dos resultados de busca fornecidos.
Retorne APENAS JSON válido, sem markdown, sem explicações.
Inclua apenas campos com dados confiáveis encontrados — omita campos incertos."""

_EXTRACT_USER = """\
Lead atual (campos null/vazio indicam o que precisa ser preenchido):
{lead_json}

Campos que já encontramos (CNPJ lookup):
{already_found}

Resultados de busca:
{search_context}

Extraia os campos ausentes no lead usando os resultados de busca.
Retorne um objeto JSON com APENAS os campos encontrados, usando estas chaves exatas:
{{
  "registeredName": "razão social",
  "companyOwner": "nome do sócio/dono principal",
  "foundationDate": "data no formato YYYY-MM-DD ou DD/MM/YYYY",
  "legalNature": "natureza jurídica",
  "segment": "setor/segmento de atuação",
  "companySize": "MEI | ME | EPP | MEDIA | GRANDE",
  "employeesCount": número inteiro,
  "revenueRange": "faixa de faturamento",
  "description": "descrição da empresa (2-3 frases)",
  "website": "URL com https://",
  "email": "email@dominio.com",
  "phone": "número com DDD",
  "instagram": "URL ou @handle",
  "linkedin": "URL do perfil/página",
  "facebook": "URL da página",
  "tiktok": "URL ou @handle"
}}

Também extraia contatos encontrados como array separado "contacts":
[{{"name": "...", "email": "...", "phone": "...", "role": "..."}}]

Responda SOMENTE com JSON: {{"updates": {{...}}, "contacts": [...]}}"""


async def extract_updates(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    web_results = state.get("web_results", [])
    existing_updates = state.get("updates", {})
    missing = state.get("missing_fields", [])
    trace = state.get("_trace")

    # If nothing to extract from web (only CNPJ was missing), skip
    web_relevant = [m for m in missing if m != "cnpj"]
    if not web_relevant or not web_results:
        return {}

    search_context = _format_tavily_results(web_results)
    if not search_context.strip():
        return {}

    already_found = json.dumps(existing_updates, ensure_ascii=False) if existing_updates else "nenhum"

    # Mask sensitive existing data — only show null/empty fields
    lead_for_prompt = {
        k: v for k, v in lead.items()
        if _is_empty(v) or k in ("businessName", "city", "state", "segment", "companyRegistrationID")
    }

    user = _EXTRACT_USER.format(
        lead_json=json.dumps(lead_for_prompt, ensure_ascii=False, indent=2),
        already_found=already_found,
        search_context=search_context[:6000],
    )

    gen = (
        trace.generation(
            name="extract_updates",
            model="claude-haiku-4-5-20251001",
            input=[{"role": "user", "content": user}],
        )
        if trace
        else None
    )

    try:
        raw = await _call_claude(
            system=_EXTRACT_SYSTEM,
            user=user,
            model="claude-haiku-4-5-20251001",
            max_tokens=2048,
        )
        text = raw.content[0].text if raw.content else "{}"

        # Strip markdown fences if present
        text = re.sub(r"^```(?:json)?\s*", "", text.strip())
        text = re.sub(r"\s*```$", "", text.strip())

        parsed = json.loads(text)
        llm_updates = parsed.get("updates", {})
        llm_contacts = parsed.get("contacts", [])
        if gen:
            gen.end(output={"fields": list(llm_updates.keys()), "contacts": len(llm_contacts)})

    except Exception:
        logger.exception("[DeepResearch] extract_updates LLM/parse error")
        if gen:
            gen.end(output={"error": "parse_failed"})
        return {}

    # Merge: CNPJ data takes priority over LLM for overlapping fields
    merged = {**llm_updates, **existing_updates}

    # Only keep fields that were actually missing
    merged = {k: v for k, v in merged.items() if not _is_empty(v)}

    # Validate contacts
    valid_contacts = [
        c for c in llm_contacts
        if c.get("name") and len(c["name"].split()) >= 2
    ]

    return {"updates": merged, "new_contacts": valid_contacts}


# ── Node: generate_summary ─────────────────────────────────────────────────


_SUMMARY_SYSTEM = """\
Você é um analista de CRM. Escreva resumos de pesquisa de leads em português brasileiro,
de forma profissional, objetiva e útil para o time comercial."""

_SUMMARY_USER = """\
Lead pesquisado:
Nome: {business_name}
CNPJ: {cnpj}
Cidade: {city} / {state}

Campos atualizados ({update_count} encontrados):
{updates_json}

Novos contatos encontrados ({contact_count}):
{contacts_json}

Campos que permanecem vazios:
{still_missing}

Escreva um resumo em pt-BR (3-5 parágrafos) cobrindo:
1. O que foi encontrado e atualizado
2. Qualidade e confiabilidade dos dados (mencione se veio do CNPJ oficial)
3. Destaques sobre a empresa (porte, segmento, presença digital)
4. Contatos encontrados e seus cargos
5. Lacunas que permanecem e pontos de atenção para o time comercial"""


async def generate_summary(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    updates = state.get("updates", {})
    new_contacts = state.get("new_contacts", [])
    missing = state.get("missing_fields", [])
    trace = state.get("_trace")

    # Fields still missing after research
    all_nullable = [
        "registeredName", "companyOwner", "foundationDate", "legalNature",
        "segment", "companySize", "employeesCount", "revenueRange", "description",
        "website", "email", "phone", "instagram", "linkedin", "facebook", "tiktok",
    ]
    still_missing = [
        f for f in all_nullable
        if _is_empty(lead.get(f)) and _is_empty(updates.get(f))
    ]

    user = _SUMMARY_USER.format(
        business_name=lead.get("businessName", "N/A"),
        cnpj=lead.get("companyRegistrationID", "N/A"),
        city=lead.get("city", "N/A"),
        state=lead.get("state", "N/A"),
        update_count=len(updates),
        updates_json=json.dumps(updates, ensure_ascii=False, indent=2) if updates else "Nenhum campo atualizado.",
        contact_count=len(new_contacts),
        contacts_json=json.dumps(new_contacts, ensure_ascii=False, indent=2) if new_contacts else "Nenhum contato novo.",
        still_missing=", ".join(still_missing) if still_missing else "Nenhum — pesquisa completa!",
    )

    gen = (
        trace.generation(
            name="generate_summary",
            model="claude-haiku-4-5-20251001",
            input=[{"role": "user", "content": user}],
        )
        if trace
        else None
    )

    if not missing and not updates:
        summary = (
            f"Pesquisa concluída para {lead.get('businessName', 'o lead')}. "
            "Todos os campos principais já estavam preenchidos — nenhuma atualização necessária."
        )
        if gen:
            gen.end(output={"skipped": True})
        return {"summary": summary}

    try:
        raw = await _call_claude(
            system=_SUMMARY_SYSTEM,
            user=user,
            model="claude-haiku-4-5-20251001",
            max_tokens=1024,
        )
        summary = raw.content[0].text.strip() if raw.content else "Resumo indisponível."
        if gen:
            gen.end(output={"length": len(summary)})
    except Exception:
        logger.exception("[DeepResearch] generate_summary LLM error")
        summary = (
            f"Pesquisa concluída para {lead.get('businessName', 'o lead')}: "
            f"{len(updates)} campo(s) atualizado(s), {len(new_contacts)} contato(s) novo(s)."
        )
        if gen:
            gen.end(output={"error": "llm_failed"})

    return {"summary": summary}
