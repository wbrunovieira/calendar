from __future__ import annotations

import json
import logging
import re
import unicodedata
from datetime import datetime, timezone

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


def _to_e164_br(phone: str) -> str | None:
    """Convert a Brazilian phone number to E.164 (+55XXXXXXXXXXX)."""
    digits = re.sub(r"\D", "", phone)
    if digits.startswith("55") and len(digits) in (12, 13):
        return "+" + digits
    if len(digits) in (10, 11):
        return "+55" + digits
    return None


_LINK_AGGREGATORS = {"linktr.ee", "linktree.com", "bio.link", "beacons.ai", "tap.bio", "campsite.bio", "koji.to", "linkin.bio"}
_THIRD_PARTY_EMAIL_HINTS = ["contabilidade", "contador", "contabil", "escritorio", "advocacia", "juridico", "fercon", "assessoria"]


def _is_link_aggregator(url: str) -> bool:
    if not url:
        return False
    return any(d in url.lower() for d in _LINK_AGGREGATORS)


def _to_domain_slug(text: str) -> str:
    """Strip accents and normalize to ASCII for domain-name slug building."""
    return unicodedata.normalize("NFKD", text).encode("ASCII", "ignore").decode()


def _is_third_party_email(email: str) -> bool:
    if not email:
        return False
    return any(p in email.lower() for p in _THIRD_PARTY_EMAIL_HINTS)


_FREE_EMAIL_DOMAINS = {"gmail.com", "hotmail.com", "yahoo.com", "outlook.com", "uol.com.br", "bol.com.br", "terra.com.br", "ig.com.br", "live.com"}
_NAME_STOP_WORDS = {"de", "da", "do", "dos", "das", "e", "em", "materiais", "comercio", "servicos", "industria", "construcao", "construção", "ltda", "me", "eireli", "sa", "epp", "mei"}


def _email_domain_matches_company(email: str, business_name: str) -> bool:
    """True if the email domain plausibly belongs to the company (or is a free provider)."""
    if not email or "@" not in email:
        return True
    domain = email.split("@")[1].lower()
    if domain in _FREE_EMAIL_DOMAINS:
        return True
    name_lower = re.sub(r"[^\w\s]", "", business_name.lower())
    name_words = [w for w in name_lower.split() if len(w) > 2 and w not in _NAME_STOP_WORDS]
    return any(w in domain for w in name_words)


def _extract_email_prefix(business_name: str) -> str | None:
    """Derive the likely email prefix from the business name (e.g. 'AREMIX LTDA' → 'aremix')."""
    name = re.sub(r"[^\w\s]", "", _to_domain_slug(business_name).lower())
    words = [w for w in name.split() if len(w) > 2 and w not in _NAME_STOP_WORDS]
    return words[0] if words else None


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


def _extract_cnpj_contacts(data: dict) -> list[dict]:
    contacts = []
    for member in (data.get("qsa") or [])[:5]:
        nome = (member.get("nome_socio") or "").strip()
        if nome and len(nome.split()) >= 2 and not nome.startswith("***"):
            qual = member.get("qualificacao_socio") or "Sócio"
            contacts.append({"name": nome.title(), "role": qual, "email": None, "phone": None})
    return contacts


def _extract_tiktok_handle(value: str) -> str | None:
    if not value:
        return None
    m = re.search(r"tiktok\.com/@([A-Za-z0-9_.]+)/?", value)
    if m:
        return m.group(1)
    m = re.match(r"@?([A-Za-z0-9_.]+)$", value.strip())
    if m:
        return m.group(1)
    return None


# ── Node: assess_previous_research ────────────────────────────────────────


_ASSESS_SYSTEM = """\
Você é um analista de CRM. Analise o resumo de uma pesquisa anterior sobre um lead
e decida quais campos ainda precisam de pesquisa e quais podem ser ignorados.
Retorne APENAS JSON válido, sem markdown."""

_ASSESS_USER = """\
Lead atual (campos com valor = já preenchidos no CRM):
{lead_json}

Data da pesquisa anterior: {previous_research_at}
Resumo da pesquisa anterior:
{previous_summary}

Com base no que o resumo anterior JÁ confirmou e no estado atual do lead:
1. Liste campos que ainda estão vazios E que a pesquisa anterior NÃO conseguiu encontrar
   → devem ser re-pesquisados (podem ter info nova desde {previous_research_at})
2. Liste campos que o resumo anterior JÁ verificou com sucesso
   → podem ser ignorados nesta rodada

Retorne JSON:
{{
  "skip_categories": ["cnpj"|"web"|"instagram"|"linkedin"|"social"|"contacts"],
  "extra_queries": ["query focada em info nova 1", ...],
  "research_note": "contexto em 1-2 frases para guiar esta re-pesquisa"
}}

Regras:
- Coloque em skip_categories APENAS categorias cujos campos já estão preenchidos
  E foram confirmados no resumo anterior
- extra_queries deve ter no máximo 3 queries adicionais focadas em info que pode
  ter surgido DESDE {previous_research_at} (ex: novos produtos, mudança de endereço)
- Se nenhum campo relevante estiver vazio, retorne skip_categories com todas as
  categorias e extra_queries vazio"""


async def assess_previous_research(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    previous_summary = state.get("previous_summary") or ""
    previous_research_at = state.get("previous_research_at") or "data desconhecida"
    trace = state.get("_trace")

    # Only show filled fields to the LLM (empty ones are what we need to find)
    lead_for_prompt = {k: v for k, v in lead.items() if not _is_empty(v)}

    gen = (
        trace.generation(
            name="assess_previous_research",
            model="claude-haiku-4-5-20251001",
            input=[{"role": "user", "content": _ASSESS_USER}],
        )
        if trace else None
    )

    skip_categories: list[str] = []
    extra_queries: list[str] = []
    research_note: str | None = None

    try:
        raw = await _call_claude(
            system=_ASSESS_SYSTEM,
            user=_ASSESS_USER.format(
                lead_json=json.dumps(lead_for_prompt, ensure_ascii=False, indent=2),
                previous_summary=previous_summary[:4000],
                previous_research_at=previous_research_at,
            ),
            model="claude-haiku-4-5-20251001",
            max_tokens=512,
        )
        text = raw.content[0].text if raw.content else "{}"
        text = re.sub(r"^```(?:json)?\s*", "", text.strip())
        text = re.sub(r"\s*```$", "", text.strip())
        parsed = json.loads(text)

        skip_categories = parsed.get("skip_categories") or []
        extra_queries = parsed.get("extra_queries") or []
        research_note = parsed.get("research_note")

        if gen:
            gen.end(output={"skip": skip_categories, "extra_queries": len(extra_queries)})

    except Exception:
        logger.exception("[DeepResearch] assess_previous_research error — proceeding with full research")
        if gen:
            gen.end(output={"error": "parse_failed"})

    logger.info(
        "[DeepResearch] re-research lead=%s skip=%s extra_queries=%d",
        state.get("lead_id"), skip_categories, len(extra_queries),
    )

    return {
        "_skip_categories": skip_categories,
        "_extra_queries": extra_queries,
        "research_note": research_note,
    }


# ── Node: plan_research ────────────────────────────────────────────────────


async def plan_research(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    contacts = state.get("contacts", [])
    name = lead.get("businessName", "") or lead.get("registeredName", "")
    cnpj = lead.get("companyRegistrationID", "")
    skip = set(state.get("_skip_categories") or [])
    extra_queries = list(state.get("_extra_queries") or [])

    missing: list[str] = []
    queries: list[str] = []

    # CNPJ data (legal, fiscal, address)
    legal_fields = ["legalNature", "companyOwner", "foundationDate", "companySize"]
    if "cnpj" not in skip and cnpj and any(_is_empty(lead.get(f)) for f in legal_fields):
        missing.append("cnpj")

    # Website / description / services
    web_fields = ["website", "description", "segment"]
    website_is_aggregator = _is_link_aggregator(lead.get("website") or "")
    if "web" not in skip and name and (any(_is_empty(lead.get(f)) for f in web_fields) or website_is_aggregator):
        missing.append("web")
        queries.append(f'"{name}" site oficial serviços')
        if lead.get("city"):
            queries.append(f'"{name}" {lead["city"]} empresa')
        if website_is_aggregator:
            queries.append(f'"{name}" site:.com.br OR site:.com -linktr.ee -instagram.com -facebook.com')

    # Instagram
    if "instagram" not in skip and _is_empty(lead.get("instagram")) and name:
        missing.append("instagram")
        queries.append(f'"{name}" instagram perfil')

    # LinkedIn
    if "linkedin" not in skip and _is_empty(lead.get("linkedin")) and name:
        missing.append("linkedin")
        queries.append(f'"{name}" linkedin empresa')

    # Facebook / TikTok
    social_missing = [
        s for s in ["facebook", "tiktok"]
        if _is_empty(lead.get(s)) and name
    ]
    if "social" not in skip and social_missing:
        missing.append("social")
        queries.append(f'"{name}" facebook tiktok redes sociais')

    # Contacts: enrich existing contacts missing email/phone, or discover new ones
    if "contacts" not in skip:
        if contacts:
            incomplete = [c for c in contacts if _is_empty(c.get("email")) or _is_empty(c.get("phone"))]
            if incomplete:
                missing.append("contacts")
                for c in incomplete[:3]:
                    cname = (c.get("name") or "").strip()
                    if cname:
                        queries.append(f'"{name}" "{cname}" email telefone linkedin contato')
                queries.append(f'"{name}" contato email telefone')
        elif name:
            missing.append("contacts")
            queries.append(f'"{name}" contato email telefone diretor gerente')
            if lead.get("website"):
                queries.append(f'site:{lead["website"]} contato equipe')

    # Inject extra queries from assess_previous_research (info nova desde a última pesquisa)
    queries.extend(extra_queries)
    if extra_queries and not missing:
        missing.append("web")  # ensure run_searches executes web path

    logger.info(
        "[DeepResearch] lead=%s missing=%s queries=%d skipped=%s",
        state.get("lead_id"), missing, len(queries), list(skip),
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

    # CNPJ lookup via BrasilAPI — always when CNPJ exists (QSA contacts), legal fields only if missing
    cnpj = lead.get("companyRegistrationID", "")
    if cnpj:
        span = trace.span(name="cnpj_lookup", input={"cnpj": cnpj}) if trace else None
        cnpj_raw = await _fetch_cnpj(cnpj)
        if cnpj_raw:
            if "cnpj" in missing:
                updates.update(_extract_cnpj_updates(cnpj_raw, lead))
            logger.info("[DeepResearch] CNPJ data retrieved for %s (legal_update=%s)", cnpj, "cnpj" in missing)
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

Contatos já cadastrados no CRM (busque email e telefone para eles nos resultados):
{existing_contacts_json}

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

Para a chave "contacts", retorne TODOS os contatos relevantes encontrados (incluindo os já cadastrados se encontrou dados novos para eles):
[{{"name": "...", "email": "...", "phone": "...", "role": "..."}}]
Inclua email e telefone sempre que encontrar nos resultados. Omita campos que não encontrou.

Responda SOMENTE com JSON: {{"updates": {{...}}, "contacts": [...]}}"""


async def extract_updates(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    contacts = state.get("contacts", [])
    web_results = state.get("web_results", [])
    existing_updates = state.get("updates", {})
    cnpj_raw = state.get("cnpj_raw")
    missing = state.get("missing_fields", [])
    trace = state.get("_trace")

    llm_updates: dict = {}
    llm_contacts: list = []

    # LLM extraction only when web searches were needed
    web_relevant = [m for m in missing if m != "cnpj"]
    if web_relevant and web_results:
        search_context = _format_tavily_results(web_results)
        if search_context.strip():
            already_found = json.dumps(existing_updates, ensure_ascii=False) if existing_updates else "nenhum"

            # Only show empty fields to LLM (plus minimal context fields)
            lead_for_prompt = {
                k: v for k, v in lead.items()
                if _is_empty(v) or k in ("businessName", "city", "state", "companyRegistrationID")
            }

            # Show existing contacts so LLM can enrich them
            existing_contacts_str = (
                json.dumps(contacts, ensure_ascii=False, indent=2)
                if contacts else "Nenhum contato cadastrado."
            )

            user = _EXTRACT_USER.format(
                lead_json=json.dumps(lead_for_prompt, ensure_ascii=False, indent=2),
                already_found=already_found,
                existing_contacts_json=existing_contacts_str,
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

    # Merge: CNPJ data takes priority over LLM for overlapping fields
    merged = {**llm_updates, **existing_updates}

    # Capture proposed fields: agent found a value but lead already has a different one
    _audit_fields = {"email", "website", "phone"}
    proposed: dict = {}
    for field in _audit_fields:
        found_val = merged.get(field)
        lead_val = lead.get(field)
        if not _is_empty(found_val) and not _is_empty(lead_val):
            if str(found_val).lower().strip() != str(lead_val).lower().strip():
                proposed[field] = found_val

    # Fix 1: only update fields that were truly empty in the original lead
    merged = {k: v for k, v in merged.items() if _is_empty(lead.get(k)) and not _is_empty(v)}

    # Linktree detection: if website found (or already on lead) is an aggregator, flag it
    website_val = merged.get("website") or lead.get("website")
    if website_val and _is_link_aggregator(str(website_val)):
        proposed["_websiteNote"] = f"Website detectado é um agregador de links ({website_val}) — empresa pode não ter site próprio"

    # Fix 2: infer whatsapp from phone when whatsapp is empty
    if _is_empty(lead.get("whatsapp")) and "whatsapp" not in merged:
        phone_source = merged.get("phone") or lead.get("phone")
        if phone_source:
            e164 = _to_e164_br(str(phone_source))
            if e164:
                merged["whatsapp"] = e164

    # QSA contacts from CNPJ: add socios as contacts if none found yet
    cnpj_contacts = _extract_cnpj_contacts(cnpj_raw) if cnpj_raw else []

    # Merge LLM contacts with existing contacts (enrich existing, add new ones)
    existing_names = {(c.get("name") or "").lower().strip() for c in contacts}
    enriched_existing: list[dict] = []
    truly_new: list[dict] = []

    for c in llm_contacts:
        if not c.get("name") or len(c["name"].split()) < 2:
            continue
        cname_lower = c["name"].lower().strip()
        if cname_lower in existing_names:
            # Enrich existing contact: only add fields that were missing
            match = next((e for e in contacts if (e.get("name") or "").lower().strip() == cname_lower), None)
            if match:
                enriched = {**match}
                if _is_empty(enriched.get("email")) and not _is_empty(c.get("email")):
                    enriched["email"] = c["email"]
                if _is_empty(enriched.get("phone")) and not _is_empty(c.get("phone")):
                    enriched["phone"] = c["phone"]
                if _is_empty(enriched.get("role")) and not _is_empty(c.get("role")):
                    enriched["role"] = c["role"]
                enriched_existing.append(enriched)
        else:
            truly_new.append(c)

    # Build final contacts list: enriched existing (only if we added data) + truly new
    final_contacts: list[dict] = []
    for orig in contacts:
        oname = (orig.get("name") or "").lower().strip()
        enriched = next((e for e in enriched_existing if (e.get("name") or "").lower().strip() == oname), None)
        if enriched:
            final_contacts.append(enriched)
        else:
            final_contacts.append(orig)
    final_contacts.extend(truly_new)

    # Add QSA contacts from CNPJ as fallback (only if not already in contacts)
    all_known_names = {(c.get("name") or "").lower().strip() for c in final_contacts}
    for qsa_contact in cnpj_contacts:
        qname = (qsa_contact.get("name") or "").lower().strip()
        if qname and qname not in all_known_names:
            final_contacts.append(qsa_contact)
            all_known_names.add(qname)

    result: dict = {"updates": merged, "new_contacts": final_contacts}
    if proposed:
        result["proposed_fields"] = proposed
    return result


# ── Node: enrich_instagram ────────────────────────────────────────────────


def _extract_instagram_handle(value: str) -> str | None:
    """Return the bare handle from a URL or @handle string."""
    if not value:
        return None
    # Strip URL parts: instagram.com/handle or instagram.com/handle/
    m = re.search(r"instagram\.com/([A-Za-z0-9_.]+)/?", value)
    if m:
        return m.group(1)
    # @handle
    m = re.match(r"@?([A-Za-z0-9_.]+)$", value.strip())
    if m:
        return m.group(1)
    return None


_SOCIAL_SYSTEM = """\
Você é um analista de presença digital. Analise os dados encontrados sobre os perfis
sociais e a Meta Ads Library da empresa. Retorne APENAS JSON válido, sem markdown."""

_SOCIAL_USER = """\
Instagram: @{ig_handle}
TikTok: @{tt_handle}
Empresa: {business_name}

Dados encontrados:
{search_context}

Extraia o que encontrar nos resultados. Use null para campos não encontrados.
Retorne JSON com esta estrutura exata:
{{
  "instagram": {{
    "followers": <número inteiro ou null>,
    "posts": <número inteiro ou null>,
    "lastPostDays": <dias desde o último post (inteiro) ou null>,
    "frequency": <"ativo" | "irregular" | "abandonado" | null>,
    "frequencyNote": <"breve descrição" ou null>
  }},
  "tiktok": {{
    "followers": <número inteiro ou null>,
    "videos": <número inteiro ou null>,
    "frequency": <"ativo" | "irregular" | "abandonado" | null>
  }},
  "metaAds": {{
    "hasAds": <true | false | null>,
    "activeCount": <número inteiro ou null>
  }}
}}

Critérios de frequency:
- "ativo": publicações regulares nos últimos 30 dias
- "irregular": publicações esporádicas ou última entre 31-90 dias
- "abandonado": sem publicações nos últimos 90 dias ou conta inativa
- null: não foi possível determinar

Para metaAds.hasAds: true apenas com evidência clara de anúncios ativos na Meta Ads
Library. Sem referência alguma → null (não false).
Se o perfil não existir ou não houver dados, retorne o objeto com todos os campos null."""


async def enrich_instagram(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    updates = state.get("updates", {})
    forced_updates = state.get("forced_updates") or {}
    trace = state.get("_trace")

    # Determine instagram/tiktok source (forced, newly found, or already on lead)
    instagram_val = forced_updates.get("instagram") or updates.get("instagram") or lead.get("instagram")
    tiktok_val = forced_updates.get("tiktok") or updates.get("tiktok") or lead.get("tiktok")

    if not instagram_val and not tiktok_val:
        return {"instagram_insights": None}

    ig_handle = _extract_instagram_handle(str(instagram_val)) if instagram_val else "N/A"
    tt_handle = _extract_tiktok_handle(str(tiktok_val)) if tiktok_val else "N/A"
    business_name = lead.get("businessName", "")
    checked_at = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    queries: list[str] = []
    if ig_handle != "N/A":
        queries += [
            f'"@{ig_handle}" instagram seguidores posts',
            f'"{ig_handle}" site:socialblade.com OR site:ninjalitics.com OR site:hypeauditor.com',
        ]
    if tt_handle != "N/A":
        queries.append(f'"@{tt_handle}" tiktok seguidores videos site:socialblade.com OR tiktok.com/@{tt_handle}')
    queries.append(f'facebook.com/ads/library "{business_name}" anúncios ativos')

    span = trace.span(name="social_enrichment", input={"ig": ig_handle, "tt": tt_handle}) if trace else None
    results = await _tavily_search(queries, max_results=5, search_depth="advanced")
    search_context = _format_tavily_results(results)

    insights: dict | None = None
    meta_ads_update: dict | None = None

    insights: dict | None = None
    meta_ads_update: dict | None = None

    if search_context.strip():
        gen = (
            trace.generation(
                name="enrich_social",
                model="claude-haiku-4-5-20251001",
                input=[{"role": "user", "content": _SOCIAL_USER}],
            )
            if trace else None
        )
        try:
            raw = await _call_claude(
                system=_SOCIAL_SYSTEM,
                user=_SOCIAL_USER.format(
                    ig_handle=ig_handle,
                    tt_handle=tt_handle,
                    business_name=business_name,
                    search_context=search_context[:5000],
                ),
                model="claude-haiku-4-5-20251001",
                max_tokens=512,
            )
            text = raw.content[0].text if raw.content else "{}"
            text = re.sub(r"^```(?:json)?\s*", "", text.strip())
            text = re.sub(r"\s*```$", "", text.strip())
            parsed = json.loads(text)

            ig_raw = parsed.get("instagram") or {}
            tt_raw = parsed.get("tiktok") or {}
            insights = {
                "handle": ig_handle,
                "followers": ig_raw.get("followers"),
                "posts": ig_raw.get("posts"),
                "lastPostDays": ig_raw.get("lastPostDays"),
                "frequency": ig_raw.get("frequency"),
                "frequencyNote": ig_raw.get("frequencyNote"),
                "tiktok": {
                    "handle": tt_handle,
                    "followers": tt_raw.get("followers"),
                    "videos": tt_raw.get("videos"),
                    "frequency": tt_raw.get("frequency"),
                } if tt_handle != "N/A" else None,
            }

            meta_raw = parsed.get("metaAds") or {}
            has_ads = meta_raw.get("hasAds")
            if has_ads is not None:
                meta_ads_update = {
                    "hasAds": bool(has_ads),
                    "activeCount": int(meta_raw.get("activeCount") or 0),
                    "checkedAt": checked_at,
                }

            if gen:
                gen.end(output={"ig_followers": insights.get("followers"), "hasAds": has_ads})
        except Exception:
            logger.exception("[DeepResearch] enrich_instagram parse error ig=@%s tt=@%s", ig_handle, tt_handle)
            if gen:
                gen.end(output={"error": "parse_failed"})

    if span:
        span.end(output={"insights_found": insights is not None})

    logger.info(
        "[DeepResearch] instagram @%s followers=%s frequency=%s hasAds=%s",
        ig_handle,
        insights.get("followers") if insights else None,
        insights.get("frequency") if insights else None,
        meta_ads_update.get("hasAds") if meta_ads_update else None,
    )

    result: dict = {"instagram_insights": insights}
    if meta_ads_update is not None and _is_empty(lead.get("metaAds")):
        result["updates"] = {**updates, "metaAds": meta_ads_update}
    return result


# ── Focused research nodes ────────────────────────────────────────────────


_FOCUSED_EXTRACT_SYSTEM = """\
Você é um analista de dados B2B especializado em enriquecimento de leads.
Retorne APENAS JSON válido, sem markdown, sem explicações."""

_FOCUSED_EXTRACT_USER = """\
Lead: {business_name}
CNPJ: {cnpj}
Cidade: {city} / {state}
Campo alvo: {focus_field}
{instruction_block}

Resultados de busca:
{search_context}

Encontre o valor mais confiável para "{focus_field}" nos resultados.

{field_instructions}

Retorne JSON:
{{
  "value": <valor encontrado ou null>,
  "confidence": "alta" | "media" | "baixa",
  "source": "breve descrição da fonte onde encontrou",
  "note": "observação relevante (ex: perfil parece inativo, email pode ser de terceiro) ou null"
}}

Se não encontrar com confiança suficiente, retorne value: null."""

_FIELD_INSTRUCTIONS: dict[str, str] = {
    "instagram": "Retorne a URL completa (https://www.instagram.com/handle) ou apenas o @handle.",
    "facebook": "Retorne a URL completa da página do Facebook.",
    "linkedin": "Retorne a URL completa da página da empresa no LinkedIn.",
    "website": "Retorne a URL com https://. NÃO retorne Linktree, bio.link ou outros agregadores — procure o domínio próprio.",
    "email": "Retorne o email mais provável da empresa. Aceite emails @hotmail.com e @gmail.com — são comuns em PMEs brasileiras. Rejeite apenas se o domínio for claramente de outra empresa (ex: escritório de contabilidade). Se encontrar múltiplos, prefira o que parece ser da própria empresa.",
    "phone": "Retorne o telefone com DDD. Priorize número comercial/fixo.",
    "phone2": "Retorne um segundo telefone com DDD, diferente do principal.",
    "whatsapp": "Retorne em formato E.164 (+55XXXXXXXXXXX) se possível.",
    "companyRegistrationID": "Retorne no formato XX.XXX.XXX/XXXX-XX. Só retorne se tiver certeza — CNPJ errado é pior que null.",
    "description": "Escreva 2-4 frases descrevendo a empresa: o que faz, segmento, diferenciais, localização.",
    "companyOwner": "Retorne o nome completo do proprietário, sócio administrador ou CEO.",
    "metaAds": "Retorne null — este campo usa tratamento especial abaixo.",
    "contacts": "Retorne null — contatos usam tratamento especial abaixo.",
    "custom": "Siga a instrução personalizada fornecida.",
}


async def plan_focused_research(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    focus = state.get("focus_field", "") or ""
    instruction = state.get("custom_instruction", "") or ""
    name = lead.get("businessName", "") or lead.get("registeredName", "")
    city = lead.get("city", "")
    cnpj = lead.get("companyRegistrationID", "")

    queries: list[str] = []
    missing: list[str] = [focus] if focus else ["web"]

    if focus == "instagram":
        queries = [
            f'"{name}" instagram perfil',
            f'"{name}" {city} instagram' if city else f'"{name}" instagram @',
            f'site:instagram.com "{name}"',
        ]
    elif focus == "facebook":
        queries = [
            f'"{name}" facebook página',
            f'site:facebook.com "{name}"',
        ]
    elif focus == "linkedin":
        queries = [
            f'"{name}" linkedin empresa página',
            f'site:linkedin.com/company "{name}"',
        ]
    elif focus == "website":
        queries = [
            f'"{name}" site oficial',
            f'"{name}" {city} site:.com.br -linktr.ee -instagram.com -facebook.com' if city else f'"{name}" site:.com.br -linktr.ee',
        ]
    elif focus == "email":
        queries: list[str] = []

        # Strip branch suffix for cleaner directory searches (e.g. "Empresa - Centro" → "Empresa")
        clean_name = re.sub(r"\s*[-–]\s*\w[\w\s]*$", "", name).strip() or name

        # Detect domain mismatch on existing email — treat as "no email" if domain is wrong company
        existing_email = lead.get("email") or ""
        domain_mismatch = existing_email and not _email_domain_matches_company(existing_email, name)
        if domain_mismatch:
            logger.info("[DeepResearch] email domain mismatch: %s vs company '%s' — searching for real email", existing_email, name)

        # 1. Domain-slug targeted search (finds email even when website is down/uncrawlable)
        # Builds "temtudopetropolis" from "Tem Tudo Petrópolis" → searches "@temtudopetropolis.com.br"
        _slug_words = [w for w in re.sub(r"[^\w\s]", "", _to_domain_slug(clean_name).lower()).split() if len(w) > 2 and w not in _NAME_STOP_WORDS]
        if _slug_words:
            full_slug = "".join(_slug_words)  # e.g. "temtudopetropolis"
            queries.append(f'"@{full_slug}.com.br" OR "{full_slug}.com.br" email')
            # Also try first meaningful word alone (e.g. "aremix.com.br") if different from full slug
            first_slug = _slug_words[0]
            if len(first_slug) >= 4 and first_slug != full_slug:
                queries.append(f'"@{first_slug}.com.br" OR "{first_slug}.com.br" email')

        # 2. Website domain first when available (highest confidence)
        website = lead.get("website") or ""
        if website and not _is_link_aggregator(website):
            wdomain = re.sub(r"https?://(www\.)?", "", website).rstrip("/").split("/")[0]
            if wdomain:
                queries.append(f'site:{wdomain} contato email OR "fale conosco"')
        else:
            # Discover the company's own .com.br domain
            queries.append(
                f'"{clean_name}" site:.com.br -instagram.com -facebook.com -guiafacil.com -solutudo.com.br'
                if city else
                f'"{clean_name}" site:.com.br -instagram.com -facebook.com'
            )

        # 3. Brazilian business directories
        queries += [
            f'site:guiafacil.com "{clean_name}" {city}' if city else f'site:guiafacil.com "{clean_name}"',
            f'site:solutudo.com.br "{clean_name}" {city}' if city else f'site:solutudo.com.br "{clean_name}"',
        ]

        # 4. General email search using clean name
        queries.append(
            f'"{clean_name}" {city} email contato'
            if city else
            f'"{clean_name}" email contato fale conosco'
        )

        # 5. Pattern-based candidates (hotmail/gmail common in Brazilian SMEs — min prefix 4 chars)
        prefix = _extract_email_prefix(clean_name)
        if prefix and len(prefix) >= 4:
            queries.append(f'"{prefix}@hotmail.com" OR "{prefix}@gmail.com"')
    elif focus in ("phone", "phone2", "whatsapp"):
        queries = [
            f'"{name}" telefone whatsapp contato',
            f'"{name}" {city} telefone comercial' if city else f'"{name}" telefone DDD',
        ]
        website = lead.get("website") or ""
        if website and not _is_link_aggregator(website):
            domain = re.sub(r"https?://(www\.)?", "", website).rstrip("/").split("/")[0]
            if domain:
                queries.insert(0, f'site:{domain} telefone contato OR "fale conosco"')
    elif focus == "companyRegistrationID":
        queries = [
            f'"{name}" CNPJ site:cnpj.biz OR site:casadosdados.com.br',
            f'"{name}" {city} CNPJ cadastro empresa' if city else f'"{name}" CNPJ',
            f'"{name}" site:receitaws.com.br OR site:cnpja.com.br',
        ]
    elif focus == "description":
        queries = [
            f'"{name}" empresa sobre história serviços',
            f'"{name}" {city} quem somos o que fazemos' if city else f'"{name}" sobre a empresa',
        ]
    elif focus == "companyOwner":
        queries = [
            f'"{name}" proprietário sócio fundador CEO dono',
            f'"{name}" {city} responsável administrador' if city else f'"{name}" sócio administrador',
        ]
    elif focus == "contacts":
        queries = [
            f'"{name}" contato email telefone diretor gerente sócio',
            f'"{name}" {city} equipe liderança linkedin' if city else f'"{name}" equipe liderança',
        ]
        if cnpj:
            missing.append("cnpj")
    elif focus == "metaAds":
        queries = [
            f'facebook.com/ads/library "{name}" anúncios ativos',
            f'"{name}" anúncios facebook meta ads biblioteca',
        ]
    elif focus == "custom":
        queries = [instruction] if instruction else [f'"{name}" informações']
    else:
        queries = [f'"{name}" {instruction}' if instruction else f'"{name}" informações']

    # Append customInstruction as extra query (for non-custom modes)
    if instruction and focus != "custom":
        queries.append(instruction)

    logger.info("[DeepResearch] focused lead=%s field=%s queries=%d", state.get("lead_id"), focus, len(queries))
    limit = 5 if focus == "email" else 4
    return {"missing_fields": missing, "search_queries": queries[:limit]}


async def extract_focused_update(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    focus = state.get("focus_field", "") or ""
    instruction = state.get("custom_instruction", "") or ""
    web_results = state.get("web_results", [])
    cnpj_raw = state.get("cnpj_raw")
    trace = state.get("_trace")

    forced_updates: dict = {}
    new_contacts: list[dict] = []

    # Special case: contacts — use QSA + LLM contact extraction
    if focus == "contacts":
        if cnpj_raw:
            new_contacts = _extract_cnpj_contacts(cnpj_raw)
        if web_results:
            search_context = _format_tavily_results(web_results)
            if search_context.strip():
                user = _EXTRACT_USER.format(
                    lead_json=json.dumps({"businessName": lead.get("businessName"), "city": lead.get("city")}, ensure_ascii=False),
                    already_found="nenhum",
                    existing_contacts_json=json.dumps(state.get("contacts") or [], ensure_ascii=False),
                    search_context=search_context[:5000],
                )
                try:
                    raw = await _call_claude(system=_EXTRACT_SYSTEM, user=user, model="claude-haiku-4-5-20251001", max_tokens=1024)
                    text = re.sub(r"^```(?:json)?\s*", "", (raw.content[0].text if raw.content else "{}").strip())
                    text = re.sub(r"\s*```$", "", text.strip())
                    parsed = json.loads(text)
                    llm_contacts = [c for c in (parsed.get("contacts") or []) if c.get("name") and len(c["name"].split()) >= 2]
                    existing_names = {(c.get("name") or "").lower().strip() for c in new_contacts}
                    for c in llm_contacts:
                        if c["name"].lower().strip() not in existing_names:
                            new_contacts.append(c)
                except Exception:
                    logger.exception("[DeepResearch] extract_focused_update contacts LLM error")
        return {"forced_updates": {}, "new_contacts": new_contacts}

    # Special case: metaAds — use social enrichment logic
    if focus == "metaAds":
        name = lead.get("businessName", "")
        checked_at = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        search_context = _format_tavily_results(web_results) if web_results else ""
        if search_context.strip():
            try:
                raw = await _call_claude(
                    system=_SOCIAL_SYSTEM,
                    user=_SOCIAL_USER.format(ig_handle="N/A", tt_handle="N/A", business_name=name, search_context=search_context[:4000]),
                    model="claude-haiku-4-5-20251001", max_tokens=256,
                )
                text = re.sub(r"^```(?:json)?\s*", "", (raw.content[0].text if raw.content else "{}").strip())
                parsed = json.loads(re.sub(r"\s*```$", "", text.strip()))
                meta = parsed.get("metaAds") or {}
                has_ads = meta.get("hasAds")
                if has_ads is not None:
                    forced_updates["metaAds"] = {"hasAds": bool(has_ads), "activeCount": int(meta.get("activeCount") or 0), "checkedAt": checked_at}
            except Exception:
                logger.exception("[DeepResearch] extract_focused_update metaAds error")
        return {"forced_updates": forced_updates, "new_contacts": []}

    # General case: extract scalar value for the target field
    if not web_results:
        return {"forced_updates": {}, "new_contacts": []}

    search_context = _format_tavily_results(web_results)
    if not search_context.strip():
        return {"forced_updates": {}, "new_contacts": []}

    field_instruction = _FIELD_INSTRUCTIONS.get(focus, "Extraia o valor solicitado.")
    instruction_parts = []
    if instruction:
        instruction_parts.append(f"Instrução adicional: {instruction}")
    # For email: warn LLM if existing email has mismatched domain
    if focus == "email":
        existing_email = lead.get("email") or ""
        if existing_email and not _email_domain_matches_company(existing_email, lead.get("businessName", "")):
            instruction_parts.append(f"ATENÇÃO: O email atual '{existing_email}' parece pertencer a outra empresa — o domínio não corresponde ao nome desta empresa. Ignore-o e busque um email novo nos resultados.")
    instruction_block = "\n".join(instruction_parts)

    gen = trace.generation(name="extract_focused", model="claude-haiku-4-5-20251001", input=[]) if trace else None
    try:
        raw = await _call_claude(
            system=_FOCUSED_EXTRACT_SYSTEM,
            user=_FOCUSED_EXTRACT_USER.format(
                business_name=lead.get("businessName", "N/A"),
                cnpj=lead.get("companyRegistrationID", "N/A"),
                city=lead.get("city", "N/A"),
                state=lead.get("state", "N/A"),
                focus_field=focus,
                instruction_block=instruction_block,
                search_context=search_context[:6000],
                field_instructions=field_instruction,
            ),
            model="claude-haiku-4-5-20251001",
            max_tokens=512,
        )
        text = re.sub(r"^```(?:json)?\s*", "", (raw.content[0].text if raw.content else "{}").strip())
        parsed = json.loads(re.sub(r"\s*```$", "", text.strip()))
        value = parsed.get("value")
        confidence = parsed.get("confidence", "baixa")
        note = parsed.get("note")

        if value and confidence in ("alta", "media"):
            forced_updates[focus] = value
            logger.info("[DeepResearch] focused field=%s value=%s confidence=%s note=%s", focus, str(value)[:80], confidence, note)
        else:
            logger.info("[DeepResearch] focused field=%s not found confidence=%s rejected_value=%s note=%s", focus, confidence, str(value)[:80] if value else "null", note)

        if gen:
            gen.end(output={"found": bool(value), "confidence": confidence})
    except Exception:
        logger.exception("[DeepResearch] extract_focused_update error for field=%s", focus)
        if gen:
            gen.end(output={"error": "parse_failed"})

    return {"forced_updates": forced_updates, "new_contacts": []}


_FOCUSED_SUMMARY_SYSTEM = """\
Você é um analista de CRM. Escreva resumos concisos de pesquisa focada em português
brasileiro, de forma profissional e útil para o time comercial."""

_FOCUSED_SUMMARY_USER = """\
Lead: {business_name} (CNPJ: {cnpj}, {city}/{state})
Campo pesquisado: {focus_field}
{instruction_block}

Resultado encontrado:
{result_block}

Escreva um resumo em pt-BR (1-3 parágrafos) descrevendo:
1. O que foi buscado e como
2. O resultado encontrado (ou ausência dele) e nível de confiança
3. Recomendações ou observações para o time comercial"""


async def generate_focused_summary(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    focus = state.get("focus_field", "") or ""
    instruction = state.get("custom_instruction", "") or ""
    forced_updates = state.get("forced_updates") or {}
    new_contacts = state.get("new_contacts") or []
    trace = state.get("_trace")

    if forced_updates:
        result_block = f"Encontrado: {json.dumps(forced_updates, ensure_ascii=False)}"
    elif new_contacts:
        result_block = f"Contatos encontrados ({len(new_contacts)}): {json.dumps(new_contacts, ensure_ascii=False, indent=2)}"
    else:
        result_block = "Nenhum resultado encontrado com confiança suficiente."

    instruction_block = f"Instrução: {instruction}" if instruction else ""

    gen = trace.generation(name="generate_focused_summary", model="claude-haiku-4-5-20251001", input=[]) if trace else None
    try:
        raw = await _call_claude(
            system=_FOCUSED_SUMMARY_SYSTEM,
            user=_FOCUSED_SUMMARY_USER.format(
                business_name=lead.get("businessName", "N/A"),
                cnpj=lead.get("companyRegistrationID", "N/A"),
                city=lead.get("city", "N/A"),
                state=lead.get("state", "N/A"),
                focus_field=focus,
                instruction_block=instruction_block,
                result_block=result_block,
            ),
            model="claude-haiku-4-5-20251001",
            max_tokens=512,
        )
        summary = raw.content[0].text.strip() if raw.content else result_block
        if gen:
            gen.end(output={"length": len(summary)})
    except Exception:
        logger.exception("[DeepResearch] generate_focused_summary error")
        summary = result_block
        if gen:
            gen.end(output={"error": "llm_failed"})

    logger.info("[DeepResearch] focused summary field=%s found=%s", focus, bool(forced_updates or new_contacts))
    return {"summary": summary}


# ── Node: generate_summary ─────────────────────────────────────────────────


_SUMMARY_SYSTEM = """\
Você é um analista de CRM. Escreva resumos de pesquisa de leads em português brasileiro,
de forma profissional, objetiva e útil para o time comercial."""

_SUMMARY_USER = """\
Lead pesquisado:
Nome: {business_name}
CNPJ: {cnpj}
Cidade: {city} / {state}

Campos novos preenchidos ({update_count} campos que estavam vazios e foram encontrados):
{updates_json}

Campos encontrados mas que já tinham valor no CRM (para auditoria):
{proposed_json}

Novos contatos encontrados ({contact_count}):
{contacts_json}

Análise de redes sociais:
{instagram_block}

Alertas automáticos (DEVEM aparecer no resumo):
{alerts_block}

Campos que permanecem vazios:
{still_missing}

{previous_context}

Escreva um resumo em pt-BR (3-5 parágrafos) cobrindo:
1. O que foi encontrado e preenchido (campos que estavam vazios)
2. Qualidade e confiabilidade dos dados (mencione se veio do CNPJ oficial)
3. Destaques sobre a empresa (porte, segmento, presença digital)
4. Redes sociais: Instagram, TikTok, frequência de posts, anúncios ativos na Meta
5. Contatos encontrados e seus cargos
6. Alertas e lacunas que permanecem — pontos de atenção para o time comercial
{reresearch_instruction}"""


def _build_social_block(insights: dict | None, updates: dict, lead: dict) -> tuple[str, str]:
    """Return (ig_handle, formatted block) for the summary prompt."""
    instagram_val = updates.get("instagram") or lead.get("instagram")
    tiktok_val = updates.get("tiktok") or lead.get("tiktok")

    if not instagram_val and not tiktok_val:
        return "N/A", "Nenhum perfil Instagram ou TikTok encontrado."

    ig_handle = _extract_instagram_handle(str(instagram_val)) if instagram_val else None
    tt_handle = _extract_tiktok_handle(str(tiktok_val)) if tiktok_val else None

    parts: list[str] = []

    # Instagram
    if ig_handle:
        parts.append(f"**Instagram @{ig_handle}**")
        if not insights:
            parts.append("  Perfil encontrado mas sem métricas disponíveis.")
        else:
            if insights.get("followers") is not None:
                parts.append(f"  Seguidores: {insights['followers']:,}")
            if insights.get("posts") is not None:
                parts.append(f"  Posts: {insights['posts']}")
            if insights.get("frequency"):
                freq = insights["frequency"].capitalize()
                note = f" — {insights['frequencyNote']}" if insights.get("frequencyNote") else ""
                parts.append(f"  Frequência: {freq}{note}")
            if insights.get("lastPostDays") is not None:
                parts.append(f"  Último post: há {insights['lastPostDays']} dias")
            if not any(insights.get(k) for k in ("followers", "posts", "frequency")):
                parts.append("  Sem métricas disponíveis (perfil possivelmente bloqueado para bots)")

    # TikTok
    tt_data = (insights or {}).get("tiktok") if insights else None
    if tt_handle:
        parts.append(f"**TikTok @{tt_handle}**")
        if tt_data:
            if tt_data.get("followers") is not None:
                parts.append(f"  Seguidores: {tt_data['followers']:,}")
            if tt_data.get("videos") is not None:
                parts.append(f"  Vídeos: {tt_data['videos']}")
            if tt_data.get("frequency"):
                parts.append(f"  Frequência: {tt_data['frequency'].capitalize()}")
            if not any(tt_data.get(k) for k in ("followers", "videos", "frequency")):
                parts.append("  Sem métricas disponíveis")
        else:
            parts.append("  Perfil encontrado mas sem métricas disponíveis.")

    # Meta Ads
    meta = updates.get("metaAds")
    if meta:
        if meta.get("hasAds"):
            parts.append(f"**Meta Ads:** {meta['activeCount']} anúncio(s) ativo(s) ✓")
        else:
            parts.append("**Meta Ads:** sem anúncios ativos encontrados")

    display_handle = ig_handle or tt_handle or "N/A"
    return display_handle, "\n".join(parts) if parts else "Perfis encontrados mas sem métricas."


async def generate_summary(state: DeepResearchState) -> dict:
    lead = state.get("lead", {})
    updates = state.get("updates", {})
    proposed_fields = state.get("proposed_fields", {})
    new_contacts = state.get("new_contacts", [])
    missing = state.get("missing_fields", [])
    instagram_insights = state.get("instagram_insights")
    previous_summary = state.get("previous_summary")
    previous_research_at = state.get("previous_research_at")
    research_note = state.get("research_note")
    trace = state.get("_trace")

    # Fields still missing after research
    all_nullable = [
        "registeredName", "companyOwner", "foundationDate", "legalNature",
        "segment", "companySize", "employeesCount", "revenueRange", "description",
        "website", "email", "phone", "whatsapp", "instagram", "linkedin", "facebook", "tiktok",
    ]
    still_missing = [
        f for f in all_nullable
        if _is_empty(lead.get(f)) and _is_empty(updates.get(f))
    ]

    ig_handle, ig_block = _build_social_block(instagram_insights, updates, lead)

    # Alerts for the summary
    alerts: list[str] = []
    existing_email = lead.get("email") or ""
    if existing_email and _is_third_party_email(existing_email):
        alerts.append(f"⚠️ Email no CRM ({existing_email}) parece ser de terceiro (contabilidade/escritório) — recomenda-se buscar email direto da empresa.")
    website_val = updates.get("website") or lead.get("website") or ""
    if website_val and _is_link_aggregator(website_val):
        alerts.append(f"⚠️ Website registrado ({website_val}) é um agregador de links (Linktree/similar) — empresa pode não ter site próprio. Verificar se há domínio próprio.")

    # Build previous research context block
    if previous_summary:
        previous_context = (
            f"Resumo da pesquisa anterior ({previous_research_at or 'data desconhecida'}):\n"
            f"{previous_summary[:2000]}"
        )
        reresearch_instruction = (
            "IMPORTANTE: Este é um resumo de RE-PESQUISA. Escreva um resumo COMPLETO e ATUALIZADO "
            "que incorpore tanto o que já era conhecido (pesquisa anterior) quanto os novos achados desta rodada. "
            "O novo resumo substitui o anterior — deve ser autocontido. "
            "Se nenhuma informação nova foi encontrada, diga explicitamente: "
            f"\"Re-pesquisa realizada em {previous_research_at or 'hoje'}. "
            "Nenhuma informação nova identificada além do já registrado.\""
        )
    else:
        previous_context = ""
        reresearch_instruction = ""

    user = _SUMMARY_USER.format(
        business_name=lead.get("businessName", "N/A"),
        cnpj=lead.get("companyRegistrationID", "N/A"),
        city=lead.get("city", "N/A"),
        state=lead.get("state", "N/A"),
        update_count=len(updates),
        updates_json=json.dumps(updates, ensure_ascii=False, indent=2) if updates else "Nenhum campo novo preenchido.",
        proposed_json=json.dumps(proposed_fields, ensure_ascii=False, indent=2) if proposed_fields else "Nenhum.",
        contact_count=len(new_contacts),
        contacts_json=json.dumps(new_contacts, ensure_ascii=False, indent=2) if new_contacts else "Nenhum contato novo.",
        instagram_block=ig_block,
        alerts_block="\n".join(alerts) if alerts else "Nenhum alerta.",
        still_missing=", ".join(still_missing) if still_missing else "Nenhum — pesquisa completa!",
        previous_context=previous_context,
        reresearch_instruction=reresearch_instruction,
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
