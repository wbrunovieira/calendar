from __future__ import annotations

import asyncio
import json
import logging
import os
import re
import unicodedata

import httpx

from app.agents.crm.nodes import _call_claude
from app.agents.crm.proposal.state import ProposalState
from app.agents.crm.proposal.counter import next_proposal_number
from app.agents.finances.nodes import langfuse
from app.config import settings

logger = logging.getLogger("agents")

_HAIKU  = "claude-haiku-4-5-20251001"
_SONNET = "claude-sonnet-4-6"


# ── Brand configuration ───────────────────────────────────────────────────

_TEMPLATE_PATHS = {
    "wb":    settings.proposal_template_wb,
    "salto": settings.proposal_template_salto,
}

# Internal key → template placeholder name, per brand.
# Nodes always output the WB internal keys; merge maps them to the
# correct template placeholder for the active brand.
_BRAND_PLACEHOLDER_MAP: dict[str, dict[str, str]] = {
    "wb": {
        "TECH_SECTIONS":          "TECH_SECTIONS",
        "PLANNING_SECTION":       "PLANNING_SECTION",
        "FEATURES_LIST":          "FEATURES_LIST",
        "METHODOLOGY_SECTION":    "METHODOLOGY_SECTION",
        "POST_LAUNCH_SECTION":    "POST_LAUNCH_SECTION",
        "LGPD_SECTION":           "LGPD_SECTION",
        "FUTURE_EVOLUTION_SECTION": "FUTURE_EVOLUTION_SECTION",
    },
    "salto": {
        "TECH_SECTIONS":          "SERVICE_SECTIONS",
        "PLANNING_SECTION":       "DIAGNOSIS_SECTION",
        "FEATURES_LIST":          "DELIVERABLES_LIST",
        "METHODOLOGY_SECTION":    "TIMELINE_SECTION",
        "POST_LAUNCH_SECTION":    "CONTINUITY_SECTION",
        "LGPD_SECTION":           "CONFIDENTIALITY_SECTION",
        "FUTURE_EVOLUTION_SECTION": "RESULTS_DISCLAIMER",
    },
}


# ══════════════════════════════════════════════════════════════════════════
# GREETING — lógica determinística, sem LLM
# ══════════════════════════════════════════════════════════════════════════

_FEMALE_NAMES = frozenset({
    "maria", "ana", "fernanda", "claudia", "patricia", "juliana", "sandra",
    "cristina", "carla", "isabela", "camila", "beatriz", "larissa", "leticia",
    "amanda", "priscila", "vanessa", "raquel", "monica", "luciana", "gabriela",
    "mariana", "tatiana", "roberta", "simone", "renata", "silvia", "bruna",
    "alice", "julia", "laura", "sofia", "melissa", "giovanna", "carolina",
    "bianca", "aline", "andrea", "fabiana", "flavia", "gisele", "helena",
    "janaina", "jessica", "joana", "karina", "lara", "lilian", "lorena",
    "lucia", "luisa", "natalia", "natasha", "nicole", "paola", "paula",
    "rebeca", "rosa", "rosana", "sabrina", "samantha", "sara", "sarah",
    "silvana", "sonia", "stephanie", "stefany", "sueli", "suzana", "tais",
    "tamara", "tatiane", "tereza", "thais", "thalita", "valeria", "vera",
    "veronica", "viviane", "yasmin", "nathalia", "margareth", "irene",
    "keila", "kelly", "ligia", "lilian", "liliana", "magda", "margaret",
    "regiane", "rita", "rosangela", "ruth", "stefania", "wendy",
})

_FEMALE_NAMES_NORMALIZED = frozenset(
    unicodedata.normalize("NFD", n).encode("ascii", "ignore").decode()
    for n in _FEMALE_NAMES
)


def _infer_gender(full_name: str) -> str:
    """Infer gender from Brazilian first name. Defaults to 'male' when ambiguous."""
    first = full_name.strip().split()[0]
    normalized = unicodedata.normalize("NFD", first).encode("ascii", "ignore").decode().lower()
    return "female" if normalized in _FEMALE_NAMES_NORMALIZED else "male"


def _build_greeting(contacts: list) -> str:
    """
    Build Portuguese greeting following saudação rules.
    Examples:
      1 homem  → "Prezado Sr. João Silva"
      1 mulher → "Prezada Sra. Maria Santos"
      2 homens → "Prezados Srs. João Silva e Carlos Oliveira"
      misto    → "Prezados Sr. João Silva e Sra. Maria Santos"
    """
    if not contacts:
        return "Prezados"

    resolved = [
        {
            "name": c.get("name", ""),
            "gender": c.get("gender") or _infer_gender(c.get("name", "")),
        }
        for c in contacts
    ]

    def _join(parts: list) -> str:
        if len(parts) == 1:
            return parts[0]
        return ", ".join(parts[:-1]) + f" e {parts[-1]}"

    all_female = all(r["gender"] == "female" for r in resolved)
    all_male   = all(r["gender"] == "male"   for r in resolved)

    if len(resolved) == 1:
        r = resolved[0]
        if r["gender"] == "female":
            return f"Prezada Sra. {r['name']}"
        return f"Prezado Sr. {r['name']}"

    if all_female:
        return f"Prezadas Sras. {_join([r['name'] for r in resolved])}"

    if all_male:
        return f"Prezados Srs. {_join([r['name'] for r in resolved])}"

    parts = [
        f"{'Sra.' if r['gender'] == 'female' else 'Sr.'} {r['name']}"
        for r in resolved
    ]
    return f"Prezados {_join(parts)}"


# ── Helpers ───────────────────────────────────────────────────────────────

def _parse_json(text: str) -> dict:
    text = re.sub(r"^```(?:json)?\s*", "", text.strip())
    text = re.sub(r"\s*```$", "", text.strip())
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        from json_repair import repair_json
        return json.loads(repair_json(text))


def _auth_headers() -> dict[str, str]:
    headers: dict[str, str] = {"Content-Type": "application/json"}
    if settings.crm_webhook_secret:
        headers["X-Webhook-Secret"] = settings.crm_webhook_secret
    return headers


def _resolve_webhook_url(url: str) -> str:
    """Replace localhost URLs with the configured CRM base URL.

    The CRM may send http://localhost:PORT/... which is unreachable from inside
    the agents Docker container. Rewrite to the external CRM URL instead.
    """
    if not url or "localhost" not in url:
        return url
    import re as _re
    rewritten = _re.sub(r"https?://localhost(:\d+)?", settings.crm_base_url.rstrip("/"), url)
    logger.info("[Proposal] rewrote webhook url: %s → %s", url, rewritten)
    return rewritten


def _build_context_block(state: ProposalState) -> str:
    lead = state.get("lead", {})
    contacts = state.get("contacts", [])
    products = state.get("products", [])
    instructions = state.get("user_instructions", "")
    qa = state.get("qa_history", [])
    brand = state.get("brand", "wb")

    primary = next((c for c in contacts if c.get("isPrimary")), contacts[0] if contacts else {})

    products_text = "\n".join(
        f"  - {p.get('name', '')} | Linha: {p.get('businessLine', '')} "
        f"| Valor estimado: R$ {p.get('estimatedValue', 'N/A')} "
        f"| Interesse: {p.get('interestLevel', '')} "
        f"| Notas: {p.get('notes', '')}"
        for p in products
    )

    qa_text = ""
    if qa:
        qa_text = "\nINFORMAÇÕES ADICIONAIS (P&R com o vendedor):\n"
        for item in qa:
            qa_text += f"  P: {item.get('question', '')}\n  R: {item.get('answer', '')}\n"

    brand_label = "WB Digital Solutions" if brand == "wb" else "Salto"

    return (
        f"EMPRESA DO CLIENTE: {lead.get('businessName', 'N/A')}\n"
        f"Razão Social: {lead.get('registeredName', '')}\n"
        f"Segmento: {lead.get('segment', '')} | Cidade: {lead.get('city', '')} | País: {lead.get('country', '')}\n"
        f"Website: {lead.get('website', '')}\n"
        f"Descrição: {lead.get('description', '')}\n\n"
        f"CONTATO PRINCIPAL: {primary.get('name', 'N/A')} | {primary.get('role', '')}\n"
        f"Email: {primary.get('email', '')} | Tel: {primary.get('phone', '')}\n\n"
        f"PRODUTOS/SERVIÇOS:\n{products_text}\n\n"
        f"INSTRUÇÕES DO VENDEDOR:\n{instructions}"
        f"{qa_text}\n\n"
        f"MARCA EMISSORA DA PROPOSTA: {brand_label}"
    )


# ══════════════════════════════════════════════════════════════════════════
# NODE 1 — analyze_completeness (Haiku)
# ══════════════════════════════════════════════════════════════════════════

_ANALYZE_SYSTEM = """\
Você avalia se há informações suficientes para gerar uma proposta comercial completa.

Itens necessários:
1. Empresa e descrição do negócio ✓ (sempre presente)
2. Nome do contato principal ✓ (sempre presente)
3. Serviço/produto de interesse ✓ (sempre presente)
4. Valor total — pode estar em products[].estimatedValue ou nas instruções
5. Condições de pagamento — ex: "30% na assinatura, 70% na entrega"
6. Prazo de entrega — ex: "3 meses", "12 semanas"
7. Escopo mínimo — o que está incluso

Se faltarem itens críticos, formule UMA única pergunta clara em português.
Priorize perguntar sobre: valor > pagamento > prazo > escopo.

Retorne APENAS JSON válido sem markdown:
{"is_sufficient": true/false, "question": "pergunta ou null", "missing": ["campos ou []"]}"""


async def analyze_completeness(state: ProposalState) -> dict:
    trace = state.get("_trace")
    ctx = _build_context_block(state)

    gen = (
        trace.generation(name="proposal_analyze", model=_HAIKU, input=[{"role": "user", "content": ctx}])
        if trace else None
    )
    try:
        resp = await _call_claude(_ANALYZE_SYSTEM, ctx, model=_HAIKU, max_tokens=512)
        raw = resp.content[0].text
        if gen:
            gen.end(output=raw, usage={"input": resp.usage.input_tokens, "output": resp.usage.output_tokens})
        data = _parse_json(raw)
        return {
            "is_sufficient": bool(data.get("is_sufficient", False)),
            "question": data.get("question"),
        }
    except Exception as exc:
        if gen:
            gen.end(output={"error": str(exc)})
        logger.exception("[Proposal] analyze_completeness failed job=%s", state.get("job_id"))
        return {"error": f"Falha ao analisar contexto: {exc}"}


# ══════════════════════════════════════════════════════════════════════════
# NODE 2a — write_narrative (Sonnet) — comum às duas marcas
# ══════════════════════════════════════════════════════════════════════════

_NARRATIVE_SYSTEM_WB = """\
Você escreve o conteúdo narrativo de uma proposta comercial da WB Digital Solutions.
Tom: profissional, parceiro de tecnologia, confiante. Foco em solução técnica e resultados.

Retorne APENAS JSON válido sem markdown:
{
  "PROJECT_TITLE": "Título do projeto (4-10 palavras, ex: 'Desenvolvimento de Sistema de Gestão Web com ERP')",
  "OPENING_PARAGRAPH_1": "1-2 frases. Mencione 'conforme nosso alinhamento' e o projeto específico.",
  "OPENING_PARAGRAPH_2": "1-2 frases demonstrando entendimento do negócio e do problema técnico a resolver.",
  "SCOPE_INTRO": "1-2 frases introduzindo o escopo do MVP.",
  "CLOSING_PARAGRAPH_1": "1-2 frases de fechamento reafirmando compromisso com a entrega técnica."
}"""

_NARRATIVE_SYSTEM_SALTO = """\
Você escreve o conteúdo narrativo de uma proposta comercial da Salto.
Tom: consultivo, direto, focado em resultados comerciais e crescimento. Sem jargões técnicos.

Retorne APENAS JSON válido sem markdown:
{
  "PROJECT_TITLE": "Título do serviço (4-10 palavras, ex: 'Consultoria Comercial e Gestão de Tráfego Pago')",
  "OPENING_PARAGRAPH_1": "1-2 frases contextualizando a conversa/diagnóstico que levou à proposta.",
  "OPENING_PARAGRAPH_2": "1-2 frases demonstrando entendimento do desafio comercial e do momento do negócio.",
  "SCOPE_INTRO": "1-2 frases introduzindo o que está incluso no escopo.",
  "CLOSING_PARAGRAPH_1": "1-2 frases de fechamento focadas em resultados e crescimento do negócio do cliente."
}"""


async def write_narrative(state: ProposalState) -> dict:
    brand = state.get("brand", "wb")
    trace = state.get("_trace")
    ctx = _build_context_block(state)
    system = _NARRATIVE_SYSTEM_SALTO if brand == "salto" else _NARRATIVE_SYSTEM_WB

    gen = (
        trace.generation(name="proposal_narrative", model=_SONNET, input=[{"role": "user", "content": ctx}])
        if trace else None
    )
    try:
        resp = await _call_claude(system, ctx, model=_SONNET, max_tokens=1024)
        raw = resp.content[0].text
        if gen:
            gen.end(output={"tokens": resp.usage.output_tokens})
        return {"narrative_sections": _parse_json(raw), "narrative_error": None}
    except Exception as exc:
        if gen:
            gen.end(output={"error": str(exc)})
        logger.exception("[Proposal] write_narrative failed job=%s", state.get("job_id"))
        return {"narrative_error": f"Falha ao escrever narrativa: {exc}"}


# ══════════════════════════════════════════════════════════════════════════
# NODE 2b — write_technical (Sonnet) — brand-aware
# Gera sempre com chaves internas unificadas; merge faz o mapeamento.
# ══════════════════════════════════════════════════════════════════════════

_TECHNICAL_SYSTEM_WB = """\
Você escreve o conteúdo técnico de uma proposta da WB Digital Solutions.

PORTFÓLIO:
- Plataformas web: Next.js + NestJS + PostgreSQL + DDD + AWS
- E-commerce / EAD: Stripe, Asaas, Panda, Vimeo, Zoom
- Apps mobile: React Native
- DevOps: Docker, Terraform, Ansible, CI/CD, AWS
- Design: Figma + Tailwind CSS
- Automação: n8n, Make, Zapier
- Websites: Next.js, WordPress

Identifique o tipo de projeto pelo businessLine e adapte a stack.

Retorne APENAS JSON válido sem markdown. Valores são HTML:
{
  "TECH_SECTIONS": "(HTML) <h3 class='subsection-title'> + <ul> com a stack técnica. Seções: Frontend, Backend, Infraestrutura (adapte ao tipo de projeto).",
  "FEATURES_LIST": "(HTML) <ul> com 5-8 <li>. Formato: <li><strong>Nome:</strong> Descrição breve.</li>"
}"""

_TECHNICAL_SYSTEM_SALTO = """\
Você escreve o conteúdo de serviços de uma proposta da Salto.
Tom: comercial, direto, sem jargões técnicos. Foco em resultados de negócio.

SERVIÇOS SALTO:
- Consultoria Comercial: funil de vendas, ICP, scripts, posicionamento
- Gestão de Tráfego Pago: Meta Ads, Google Ads, relatórios de performance
- Treinamento de Equipe Comercial: técnicas de vendas, role-plays, playbook
- Automação Comercial: n8n, Make, Zapier (fluxos de vendas e atendimento)
- Websites e Landing Pages: páginas de conversão e presença digital

Identifique os serviços pelos campos products[].businessLine e name.

Retorne APENAS JSON válido sem markdown. Valores são HTML:
{
  "TECH_SECTIONS": "(HTML) <h3 class='subsection-title'> + <ul> descrevendo CADA serviço contratado em bullets práticos. Sem linguagem técnica.",
  "FEATURES_LIST": "(HTML) <ul> com 4-7 <li> listando as entregas concretas. Formato: <li><strong>Entrega:</strong> Descrição do que o cliente receberá.</li>"
}"""


async def write_technical(state: ProposalState) -> dict:
    brand = state.get("brand", "wb")
    trace = state.get("_trace")
    ctx = _build_context_block(state)
    system = _TECHNICAL_SYSTEM_SALTO if brand == "salto" else _TECHNICAL_SYSTEM_WB

    gen = (
        trace.generation(name="proposal_technical", model=_SONNET, input=[{"role": "user", "content": ctx}])
        if trace else None
    )
    try:
        resp = await _call_claude(system, ctx, model=_SONNET, max_tokens=3072)
        raw = resp.content[0].text
        if gen:
            gen.end(output={"tokens": resp.usage.output_tokens})
        return {"technical_sections": _parse_json(raw), "technical_error": None}
    except Exception as exc:
        if gen:
            gen.end(output={"error": str(exc)})
        logger.exception("[Proposal] write_technical failed job=%s", state.get("job_id"))
        return {"technical_error": f"Falha ao escrever seções técnicas/de serviço: {exc}"}


# ══════════════════════════════════════════════════════════════════════════
# NODE 2c — write_commercial (Haiku) — comum às duas marcas
# ══════════════════════════════════════════════════════════════════════════

_COMMERCIAL_SYSTEM_WB = """\
Você escreve o conteúdo financeiro de uma proposta da WB Digital Solutions.

Regras:
- Valores em formato brasileiro: R$ X.XXX,00
- INVESTMENT_TABLE: divida o total em componentes (Frontend, Backend, Design, Infra, etc.)
- Custos de terceiros (AWS, domínios, APIs) NÃO entram na tabela
- ADDITIONAL_CONSIDERATIONS_ITEMS: sempre inclua aprovações/disponibilidade. Para projetos com infra cloud, adicione item sobre custos externos. Adapte a linguagem ao perfil do cliente.

Retorne APENAS JSON válido sem markdown. Valores são HTML:
{
  "INVESTMENT_TABLE": "<table class='investment-table'>. Cada linha: <tr><td><strong>Item</strong><br/><small>Descrição</small></td><td>R$ X.XXX,00</td></tr>. Total: <tr class='total-row'><td>Total do Investimento</td><td>R$ XX.XXX,00</td></tr>",
  "PAYMENT_SCHEDULE": "<ol class='payment-list'> com parcelas. Cada: <li><strong>Parcela — Gatilho:</strong> R$ X.XXX,00 — descrição.</li>",
  "ADDITIONAL_CONSIDERATIONS_ITEMS": "Um ou mais <li> com considerações. Sempre inclua o item de aprovações."
}"""

_COMMERCIAL_SYSTEM_SALTO = """\
Você escreve o conteúdo financeiro de uma proposta da Salto.
Tom: simples, direto, sem linguagem técnica. Foco em custo-benefício.

Regras:
- Valores em formato brasileiro: R$ X.XXX,00
- INVESTMENT_TABLE: divida o total por serviço/fase (Diagnóstico, Gestão de Tráfego, Treinamento, etc.)
- Verbas de mídia (anúncios) NÃO entram na tabela — são do cliente
- ADDITIONAL_CONSIDERATIONS_ITEMS: adapte ao serviço. Para tráfego pago, inclua item sobre verba de mídia. Sempre inclua item de engajamento/disponibilidade.

Retorne APENAS JSON válido sem markdown. Valores são HTML:
{
  "INVESTMENT_TABLE": "<table class='investment-table'>. Cada linha: <tr><td><strong>Serviço</strong><br/><small>Descrição breve</small></td><td>R$ X.XXX,00</td></tr>. Total: <tr class='total-row'><td>Total do Investimento</td><td>R$ XX.XXX,00</td></tr>",
  "PAYMENT_SCHEDULE": "<ol class='payment-list'> com parcelas. Cada: <li><strong>Parcela — Gatilho:</strong> R$ X.XXX,00 — descrição.</li>",
  "ADDITIONAL_CONSIDERATIONS_ITEMS": "Um ou mais <li> com considerações relevantes ao serviço. Linguagem simples."
}"""


async def write_commercial(state: ProposalState) -> dict:
    brand = state.get("brand", "wb")
    trace = state.get("_trace")
    ctx = _build_context_block(state)
    system = _COMMERCIAL_SYSTEM_SALTO if brand == "salto" else _COMMERCIAL_SYSTEM_WB

    gen = (
        trace.generation(name="proposal_commercial", model=_HAIKU, input=[{"role": "user", "content": ctx}])
        if trace else None
    )
    try:
        resp = await _call_claude(system, ctx, model=_HAIKU, max_tokens=2048)
        raw = resp.content[0].text
        if gen:
            gen.end(output={"tokens": resp.usage.output_tokens})
        return {"commercial_sections": _parse_json(raw), "commercial_error": None}
    except Exception as exc:
        if gen:
            gen.end(output={"error": str(exc)})
        logger.exception("[Proposal] write_commercial failed job=%s", state.get("job_id"))
        return {"commercial_error": f"Falha ao escrever seções comerciais: {exc}"}


# ══════════════════════════════════════════════════════════════════════════
# NODE 2d — write_optional_sections (Haiku) — brand-aware
# Retorna chaves internas unificadas (WB names); merge faz mapeamento.
# ══════════════════════════════════════════════════════════════════════════

_OPTIONAL_SYSTEM_WB = """\
Você decide e gera as seções opcionais de uma proposta da WB Digital Solutions.
Para cada placeholder, retorne o BLOCO HTML COMPLETO (incluindo o <h2>) OU "" se não se aplicar.

CRITÉRIOS:
- PLANNING_SECTION: incluir para sistemas/plataformas complexas com arquitetura. Omitir para websites simples, landing pages, automações pontuais.
- METHODOLOGY_SECTION: incluir para projetos iterativos com sprints. Omitir para entrega única.
- POST_LAUNCH_SECTION: incluir para plataformas/sistemas que precisam de manutenção. Omitir para entregas fechadas.
- PENALTY_CLAUSE: incluir para projetos com prazo crítico ou valor > R$ 15.000. Omitir para projetos simples.
- LGPD_SECTION: incluir se coleta dados pessoais (login, e-commerce, CRM). Omitir para sites institucionais, landing pages.
- FUTURE_EVOLUTION_SECTION: incluir para produtos digitais com roadmap. Omitir para entregas pontuais.

REGRA: Se todos os opcionais seriam omitidos, inclua ao menos POST_LAUNCH_SECTION.

FORMATOS OBRIGATÓRIOS:

PLANNING_SECTION quando incluído:
<h3 class="subsection-title">Planejamento Inicial</h3>
<ul>
  <li><strong>Estudo Inicial e Arquitetura:</strong> [descrição da fase de arquitetura adaptada ao projeto]</li>
</ul>

METHODOLOGY_SECTION quando incluído — use .sprint-list/.sprint-badge:
<h2 class="section-title">Metodologia e Cronograma</h2>
<p>Para garantir flexibilidade e alinhamento contínuo, adotaremos metodologia ágil com sprints quinzenais de 2 semanas, permitindo entregas incrementais e feedback contínuo.</p>
<div class="sprint-list">
  <div class="sprint-item"><div class="sprint-badge">1</div><div class="sprint-content"><strong>Sprint 1 (Semanas 1–2):</strong> [descrição]</div></div>
  [demais sprints baseados no prazo total]
  <div class="sprint-item"><div class="sprint-badge">✓</div><div class="sprint-content"><strong>Sprint Final:</strong> Testes, implantação e entrega.</div></div>
</div>

POST_LAUNCH_SECTION quando incluído:
<h2 class="section-title">Suporte Pós-Lançamento</h2>
<p>Após o go-live, oferecemos suporte técnico mensal no valor fixo de <strong>[R$ X.XXX,00 — ~8% do valor total/mês se não especificado]</strong>. Cobre correções, manutenção preventiva e pequenas melhorias contínuas.</p>

PENALTY_CLAUSE quando incluído — COPIE ESTE TEXTO EXATAMENTE:
<h2 class="section-title">Cláusula de Multa por Atraso</h2>
<p>Fica acordado que, em caso de atraso na entrega do projeto, será aplicada uma multa de 5% do valor total por semana, limitada a 20%. A aplicação dessa penalidade ocorrerá apenas após 4 semanas de atraso além do prazo acordado, contados a partir do início oficial do projeto.</p>

LGPD_SECTION quando incluído — COPIE ESTE TEXTO EXATAMENTE:
<h2 class="section-title">Proteção de Dados e Conformidade com a LGPD</h2>
<p>Todos os processos de desenvolvimento e operação da plataforma respeitarão as diretrizes da Lei Geral de Proteção de Dados (LGPD), assegurando a confidencialidade, integridade e segurança das informações dos usuários. Implementaremos boas práticas de governança de dados, autenticação segura, controle de acesso e armazenamento protegido, garantindo o cumprimento das obrigações legais referentes ao tratamento de dados pessoais.</p>

FUTURE_EVOLUTION_SECTION quando incluído:
<h2 class="section-title">Evolução Futura e Uso de Inteligência Artificial</h2>
<p>Nossa empresa estará preparada para continuar a evoluir o projeto após a entrega do MVP, oferecendo melhorias contínuas em aspectos estratégicos.</p>
<p>Podemos implementar:</p>
<ul>
  <li><strong>[Item relevante ao projeto]:</strong> [Descrição]</li>
</ul>

Retorne APENAS JSON válido sem markdown:
{"PLANNING_SECTION":"HTML ou ''","METHODOLOGY_SECTION":"HTML ou ''","POST_LAUNCH_SECTION":"HTML ou ''","PENALTY_CLAUSE":"HTML ou ''","LGPD_SECTION":"HTML ou ''","FUTURE_EVOLUTION_SECTION":"HTML ou ''"}"""

_OPTIONAL_SYSTEM_SALTO = """\
Você decide e gera as seções opcionais de uma proposta da Salto.
Tom: comercial, direto, sem linguagem técnica.
Para cada placeholder, retorne o BLOCO HTML COMPLETO (incluindo o <h2>) OU "" se não se aplicar.

CRITÉRIOS:
- PLANNING_SECTION (→ Diagnóstico Inicial): incluir quando há etapa de imersão/diagnóstico antes da execução. Omitir se escopo já totalmente definido.
- METHODOLOGY_SECTION (→ Cronograma de Execução): incluir para projetos com fases progressivas. Omitir para contratos mensais sem fases definidas.
- POST_LAUNCH_SECTION (→ Acompanhamento Contínuo): incluir quando há possibilidade de continuidade mensal. Omitir para projetos de entrega única sem renovação.
- PENALTY_CLAUSE: incluir para contratos de maior valor ou prazo crítico. Omitir quando prejudica o fechamento.
- LGPD_SECTION (→ Confidencialidade): incluir quando há acesso a dados sensíveis do negócio (faturamento, base de clientes).
- FUTURE_EVOLUTION_SECTION (→ Sobre Resultados): RECOMENDADO sempre que houver expectativa de ROI/metas quantitativas. Protege a Salto.

REGRA: Se todos os opcionais seriam omitidos, inclua ao menos POST_LAUNCH_SECTION ou FUTURE_EVOLUTION_SECTION.

FORMATOS OBRIGATÓRIOS (use os estilos do template Salto):

PLANNING_SECTION quando incluído:
<h3 class="subsection-title">Diagnóstico Inicial</h3>
<ul>
  <li><strong>Imersão no Negócio:</strong> [descrição do diagnóstico comercial adaptado ao serviço]</li>
</ul>

METHODOLOGY_SECTION quando incluído — use .phase-list/.phase-badge (NÃO .sprint-list):
<h2 class="section-title">Cronograma de Execução</h2>
<p>A execução será organizada em fases progressivas, garantindo evolução contínua e entregas tangíveis a cada etapa:</p>
<div class="phase-list">
  <div class="phase-item"><div class="phase-badge">1</div><div class="phase-content"><strong>Fase 1 (Semanas 1–2):</strong> [descrição]</div></div>
  [demais fases baseadas no prazo total]
  <div class="phase-item"><div class="phase-badge">✓</div><div class="phase-content"><strong>Fase Final:</strong> Entrega do playbook, relatório e reunião de fechamento.</div></div>
</div>

POST_LAUNCH_SECTION quando incluído:
<h2 class="section-title">Acompanhamento Contínuo</h2>
<p>Após a conclusão do escopo inicial, oferecemos plano de acompanhamento mensal com valor fixo de <strong>[R$ X.XXX,00 — ~10% do valor total/mês se não especificado]</strong>. Inclui gestão contínua, relatórios e reuniões mensais de alinhamento.</p>

PENALTY_CLAUSE quando incluído — COPIE ESTE TEXTO EXATAMENTE:
<h2 class="section-title">Cláusula de Multa por Atraso</h2>
<p>Fica acordado que, em caso de atraso na entrega do projeto, será aplicada uma multa de 5% do valor total por semana, limitada a 20%. A aplicação dessa penalidade ocorrerá apenas após 4 semanas de atraso além do prazo acordado, contados a partir do início oficial do projeto.</p>

LGPD_SECTION quando incluído:
<h2 class="section-title">Confidencialidade</h2>
<p>Todas as informações compartilhadas pelo cliente durante a execução deste trabalho serão tratadas com total sigilo. A Salto compromete-se a não divulgar dados estratégicos, financeiros ou operacionais a terceiros, garantindo a proteção das informações do seu negócio.</p>

FUTURE_EVOLUTION_SECTION quando incluído:
<h2 class="section-title">Sobre Resultados e Metas</h2>
<p>Os resultados de estratégias de marketing e vendas dependem de variáveis externas como sazonalidade, mercado e engajamento do cliente. A Salto compromete-se com a excelência na execução e na otimização contínua das ações, mas não garante métricas específicas de retorno. Metas e indicadores serão definidos em conjunto no início do projeto e revisados periodicamente.</p>

Retorne APENAS JSON válido sem markdown:
{"PLANNING_SECTION":"HTML ou ''","METHODOLOGY_SECTION":"HTML ou ''","POST_LAUNCH_SECTION":"HTML ou ''","PENALTY_CLAUSE":"HTML ou ''","LGPD_SECTION":"HTML ou ''","FUTURE_EVOLUTION_SECTION":"HTML ou ''"}"""


async def write_optional_sections(state: ProposalState) -> dict:
    brand = state.get("brand", "wb")
    trace = state.get("_trace")
    ctx = _build_context_block(state)
    system = _OPTIONAL_SYSTEM_SALTO if brand == "salto" else _OPTIONAL_SYSTEM_WB

    gen = (
        trace.generation(name="proposal_optional", model=_HAIKU, input=[{"role": "user", "content": ctx}])
        if trace else None
    )
    try:
        resp = await _call_claude(system, ctx, model=_HAIKU, max_tokens=3072)
        raw = resp.content[0].text
        if gen:
            gen.end(output={"tokens": resp.usage.output_tokens})
        return {"optional_sections": _parse_json(raw), "optional_error": None}
    except Exception as exc:
        if gen:
            gen.end(output={"error": str(exc)})
        logger.exception("[Proposal] write_optional_sections failed job=%s", state.get("job_id"))
        return {"optional_error": f"Falha ao gerar seções opcionais: {exc}"}


# ══════════════════════════════════════════════════════════════════════════
# NODE 3 — merge_sections (determinístico)
# Combina os 4 dicts, mapeia chaves internas → placeholders do template correto
# ══════════════════════════════════════════════════════════════════════════

async def merge_sections(state: ProposalState) -> dict:
    errors = [e for e in [
        state.get("narrative_error"),
        state.get("technical_error"),
        state.get("commercial_error"),
        state.get("optional_error"),
    ] if e]
    if errors:
        return {"error": " | ".join(errors)}

    brand = state.get("brand", "wb")
    proposal_number = await next_proposal_number(brand)

    # Merge all node outputs using unified internal keys
    all_sections: dict = {}
    all_sections.update(state.get("narrative_sections") or {})
    all_sections.update(state.get("technical_sections") or {})
    all_sections.update(state.get("commercial_sections") or {})
    all_sections.update(state.get("optional_sections") or {})

    # Add deterministic sections
    all_sections["PROPOSAL_NUMBER"] = proposal_number
    all_sections["GREETING"] = _build_greeting(state.get("contacts") or [])

    # Load the correct brand template
    template_path = _TEMPLATE_PATHS.get(brand, _TEMPLATE_PATHS["wb"])
    try:
        with open(template_path, encoding="utf-8") as f:
            template = f.read()
    except Exception as exc:
        return {"error": f"Template não encontrado ({brand}): {exc}"}

    # Substitute: map internal key → brand-specific placeholder name, then replace
    placeholder_map = _BRAND_PLACEHOLDER_MAP.get(brand, _BRAND_PLACEHOLDER_MAP["wb"])
    filled = template
    for internal_key, value in all_sections.items():
        template_key = placeholder_map.get(internal_key, internal_key)
        filled = filled.replace(f"{{{{{template_key}}}}}", str(value) if value else "")

    logger.info("[Proposal] sections merged job=%s brand=%s number=%s",
                state.get("job_id"), brand, proposal_number)
    return {
        "proposal_number": proposal_number,
        "project_title": all_sections.get("PROJECT_TITLE", "Proposta Comercial"),
        "filled_html": filled,
    }


# ══════════════════════════════════════════════════════════════════════════
# NODE 4 — convert_to_pdf (WeasyPrint, executor thread)
# ══════════════════════════════════════════════════════════════════════════

_WEASYPRINT_OVERRIDES = """<style>
  /* WeasyPrint PDF rendering overrides */
  @page       { margin: 15mm 15mm 15mm 15mm; }
  @page :first { margin: 15mm 0   15mm 0;   }  /* cover page: zero side margins for full-bleed hero */
  body { background: #fff !important; }
  .page-block { box-shadow: none !important; padding-left: 15mm !important; padding-right: 15mm !important; }
  .document-wrapper { padding: 0 !important; }
  .page-break-visual { display: none !important; }
  /* Hide fixed running header/footer (causes duplicate logo and content overlap) */
  .print-header { display: none !important; }
  .print-footer { display: none !important; }
  /* Show per-section logo at top of each section page */
  .preview-header { display: flex !important; align-items: center; padding-bottom: 4mm; margin-bottom: 6mm; border-bottom: 1px solid #e0e0e0; }
  /* Cover page: full-bleed hero — zero out page-block side padding, hero fills full width */
  .page-block-cover { padding-left: 0 !important; padding-right: 0 !important; }
  .cover-hero { margin-left: 0 !important; margin-right: 0 !important; padding-left: 15mm !important; padding-right: 15mm !important; }
  /* Re-add side spacing to cover content below the hero */
  .proposal-pill  { margin-left: 15mm !important; }
  .opening-letter { padding-left: 15mm !important; padding-right: 15mm !important; }
</style>"""


def _html_to_pdf_sync(html: str) -> bytes:
    from weasyprint import HTML
    html = html.replace("</head>", _WEASYPRINT_OVERRIDES + "</head>", 1)
    return HTML(string=html, base_url=None).write_pdf()


async def convert_to_pdf(state: ProposalState) -> dict:
    import base64
    job_id = state.get("job_id", "unknown")
    lead = state.get("lead", {})
    brand = state.get("brand", "wb")
    company = re.sub(r"[^\w]", "", lead.get("businessName", "Proposta"))[:20]

    from datetime import datetime, timezone
    now = datetime.now(timezone.utc)
    prefix = "Proposta_Salto" if brand == "salto" else "Proposta_WB"
    file_name = f"{prefix}_{company}_{now.strftime('%m%y')}.pdf"

    try:
        loop = asyncio.get_event_loop()
        pdf_bytes = await loop.run_in_executor(None, _html_to_pdf_sync, state["filled_html"])
        pdf_base64 = base64.b64encode(pdf_bytes).decode("utf-8")
        logger.info("[Proposal] PDF generated job=%s size=%d bytes", job_id, len(pdf_bytes))
        return {
            "file_name": file_name,
            "file_size": len(pdf_bytes),
            "pdf_base64": pdf_base64,
        }
    except Exception as exc:
        logger.exception("[Proposal] convert_to_pdf failed job=%s", job_id)
        return {"error": f"Falha ao gerar PDF: {exc}"}


# ══════════════════════════════════════════════════════════════════════════
# Webhook nodes
# ══════════════════════════════════════════════════════════════════════════

async def send_asking_webhook(state: ProposalState) -> dict:
    job_id = state.get("job_id", "")
    payload = {
        "jobId": job_id,
        "proposalId": state.get("proposal_id", job_id),
        "status": "question",
        "question": state.get("question", "Poderia fornecer mais detalhes sobre o projeto?"),
    }
    try:
        async with httpx.AsyncClient(timeout=15) as client:
            await client.post(_resolve_webhook_url(state.get("webhook_url", "")), json=payload, headers=_auth_headers())
        logger.info("[Proposal] question webhook sent job=%s", job_id)
    except Exception as exc:
        logger.exception("[Proposal] question webhook failed job=%s", job_id)
        return {"error": f"Falha ao enviar pergunta ao CRM: {exc}"}
    return {}


async def send_completed_webhook(state: ProposalState) -> dict:
    job_id = state.get("job_id", "")
    webhook_url = _resolve_webhook_url(state.get("webhook_url", ""))
    logger.info("[Proposal] sending completed webhook job=%s url=%s size=%d",
                job_id, webhook_url, state.get("file_size", 0))
    payload = {
        "jobId": job_id,
        "proposalId": state.get("proposal_id", job_id),
        "status": "completed",
        "fileName": state.get("file_name", ""),
        "fileSize": state.get("file_size", 0),
        "fileBase64": state.get("pdf_base64", ""),
    }
    try:
        async with httpx.AsyncClient(timeout=30) as client:
            await client.post(webhook_url, json=payload, headers=_auth_headers())
        logger.info("[Proposal] completed job=%s number=%s", job_id, state.get("proposal_number"))
    except Exception as exc:
        logger.exception("[Proposal] completed webhook failed job=%s", job_id)
        return {"error": f"Proposta gerada mas falha ao notificar CRM: {exc}"}
    return {}


async def send_error_webhook(state: ProposalState) -> dict:
    job_id = state.get("job_id", "")
    webhook_url = _resolve_webhook_url(state.get("webhook_url", ""))
    payload = {
        "jobId": job_id,
        "proposalId": state.get("proposal_id", job_id),
        "status": "error",
        "errorMessage": state.get("error", "Erro desconhecido"),
    }
    logger.info("[Proposal] sending error webhook job=%s url=%s", job_id, webhook_url)
    try:
        async with httpx.AsyncClient(timeout=15) as client:
            await client.post(webhook_url, json=payload, headers=_auth_headers())
    except Exception:
        logger.exception("[Proposal] error webhook also failed job=%s url=%s", job_id, webhook_url)
    return {}
