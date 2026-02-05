from __future__ import annotations

import json
import logging
from datetime import date

from langchain_core.messages import HumanMessage, SystemMessage
from langchain_openai import ChatOpenAI
from langfuse import Langfuse

from app.agents.transaction.prompts import FALLBACK_SYSTEM, FALLBACK_USER
from app.agents.transaction.state import TransactionState
from app.clients import finances
from app.config import settings

logger = logging.getLogger("calendar-agents")

langfuse = Langfuse(
    public_key=settings.langfuse_public_key,
    secret_key=settings.langfuse_secret_key,
    host=settings.langfuse_host,
    enabled=bool(settings.langfuse_public_key and settings.langfuse_secret_key),
)

# ── Prompt helpers ───────────────────────────────────────────

_prompt_cache: dict | None = None


def _get_langfuse_prompt() -> list[dict] | None:
    """Fetch prompt from Langfuse with in-memory cache. Returns None on failure."""
    global _prompt_cache
    if _prompt_cache is not None:
        return _prompt_cache
    if not langfuse.enabled:
        return None
    try:
        prompt = langfuse.get_prompt("transaction-parser", type="chat", label="production")
        _prompt_cache = prompt
        logger.info("Loaded prompt 'transaction-parser' v%s from Langfuse", prompt.version)
        return prompt
    except Exception:
        logger.warning("Failed to fetch prompt from Langfuse, using local fallback")
        return None


def _build_messages(
    raw_text: str, today: str, accounts_list: str, categories_list: str,
) -> tuple[str, str, dict | None]:
    """Build system + user messages. Returns (system, user, langfuse_prompt_or_None)."""
    prompt = _get_langfuse_prompt()
    if prompt is not None:
        compiled = prompt.compile(
            today=today,
            accounts_list=accounts_list,
            categories_list=categories_list,
            raw_text=raw_text,
        )
        system = compiled[0]["content"]
        user = compiled[1]["content"]
        return system, user, prompt
    # Local fallback — same structure, no Langfuse link
    system = FALLBACK_SYSTEM
    user = FALLBACK_USER.format(
        today=today,
        accounts_list=accounts_list,
        categories_list=categories_list,
        raw_text=raw_text,
    )
    return system, user, None


async def load_context(state: TransactionState) -> dict:
    """Fetch accounts and categories from calendar-finances."""
    trace = state.get("_trace")
    profile_id = state["profile_id"]

    span = trace.span(name="load_context", input={"profile_id": profile_id}) if trace else None

    try:
        accounts = await finances.get_bank_accounts(profile_id)
        categories = await finances.get_categories(profile_id)
    except Exception as exc:
        logger.exception("Failed to load context from calendar-finances")
        if span:
            span.end(output={"error": str(exc)})
        return {
            "accounts": [],
            "categories": [],
            "error": f"Erro ao carregar dados: {exc}",
            "reply": "Erro interno ao carregar contas e categorias.",
        }

    if span:
        span.end(output={"accounts": len(accounts), "categories": len(categories)})

    return {"accounts": accounts, "categories": categories}


async def parse_message(state: TransactionState) -> dict:
    """Use LLM to extract structured data from the raw text."""
    if state.get("error"):
        return {}

    trace = state.get("_trace")

    accounts_list = "\n".join(
        f"- {a['name']}" for a in state["accounts"]
    ) or "Nenhuma conta cadastrada"

    categories_list = "\n".join(
        f"- {c['name']} ({c['type']})" for c in state["categories"]
    ) or "Nenhuma categoria cadastrada"

    today = date.today().isoformat()

    system, user, lf_prompt = _build_messages(
        raw_text=state["raw_text"],
        today=today,
        accounts_list=accounts_list,
        categories_list=categories_list,
    )

    gen_kwargs: dict = {
        "name": "parse_message",
        "model": settings.deepseek_model,
        "input": [{"role": "system", "content": system}, {"role": "user", "content": user}],
        "metadata": {"prompt_source": "langfuse" if lf_prompt else "local"},
    }
    if lf_prompt is not None:
        gen_kwargs["prompt"] = lf_prompt

    generation = trace.generation(**gen_kwargs) if trace else None

    llm = ChatOpenAI(
        api_key=settings.deepseek_api_key,
        base_url=settings.deepseek_base_url,
        model=settings.deepseek_model,
        temperature=0,
    )

    response = await llm.ainvoke([SystemMessage(content=system), HumanMessage(content=user)])
    raw = response.content.strip()

    if raw.lower() == "null" or not raw:
        if generation:
            generation.end(output="null", metadata={"parse_status": "failed"})
        return {
            "parsed": None,
            "error": "parse_failed",
            "reply": "Não entendi. Tente: 'descrição valor conta'. Ex: 'almoco 32 nubank'",
        }

    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        if generation:
            generation.end(output=raw, metadata={"parse_status": "invalid_json"})
        return {
            "parsed": None,
            "error": "parse_failed",
            "reply": "Não entendi. Tente: 'descrição valor conta'. Ex: 'almoco 32 nubank'",
        }

    if generation:
        generation.end(output=parsed, metadata={"parse_status": "ok"})

    return {"parsed": parsed}


async def resolve_entities(state: TransactionState) -> dict:
    """Match account_name and category_name to their IDs."""
    if state.get("error") or not state.get("parsed"):
        return {}

    trace = state.get("_trace")
    parsed = state["parsed"]
    span = trace.span(name="resolve_entities", input=parsed) if trace else None

    # Resolve bank account
    bank_account_id = None
    account_name = parsed.get("account_name")
    if account_name:
        for acc in state["accounts"]:
            if acc["name"].lower() == account_name.lower():
                bank_account_id = acc["id"]
                break
        if not bank_account_id:
            available = ", ".join(a["name"] for a in state["accounts"])
            if span:
                span.end(output={"error": "account_not_found", "account_name": account_name})
            return {
                "error": "account_not_found",
                "reply": f"Conta '{account_name}' não encontrada. Contas disponíveis: {available}",
            }

    # Resolve category
    category_id = None
    category_name = parsed.get("category_name")
    if category_name:
        for cat in state["categories"]:
            if cat["name"].lower() == category_name.lower():
                category_id = cat["id"]
                break

    if span:
        span.end(output={"bank_account_id": bank_account_id, "category_id": category_id})

    return {"bank_account_id": bank_account_id, "category_id": category_id}


async def create_transaction(state: TransactionState) -> dict:
    """POST the transaction to calendar-finances."""
    if state.get("error"):
        return {}

    trace = state.get("_trace")
    parsed = state["parsed"]

    if not state.get("bank_account_id"):
        return {
            "error": "account_required",
            "reply": "Conta bancária é obrigatória. Tente: 'almoco 32 nubank'",
        }

    payload = {
        "profileId": state["profile_id"],
        "bankAccountId": state["bank_account_id"],
        "type": parsed.get("type", "EXPENSE"),
        "amount": parsed["amount"],
        "currency": "BRL",
        "description": parsed["description"],
        "occurredOn": parsed.get("occurred_on", date.today().isoformat()),
        "status": parsed.get("status", "CONFIRMED"),
    }
    if state.get("category_id"):
        payload["categoryId"] = state["category_id"]

    span = trace.span(name="create_transaction", input=payload) if trace else None

    try:
        result = await finances.create_transaction(payload)
    except Exception as exc:
        logger.exception("Failed to create transaction")
        if span:
            span.end(output={"error": str(exc)})
        return {
            "error": "api_error",
            "reply": f"Erro ao criar lançamento: {exc}",
        }

    if span:
        span.end(output=result)

    return {"transaction": result}


async def format_reply(state: TransactionState) -> dict:
    """Build the final reply message."""
    if state.get("error"):
        return {}

    parsed = state["parsed"]
    tx = state.get("transaction", {})

    amount_fmt = f"R$ {parsed['amount']:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")
    parts = [f"✓ {parsed['description']} {amount_fmt}"]

    account_name = parsed.get("account_name", "")
    category_name = parsed.get("category_name", "")
    if account_name and category_name:
        parts.append(f"({account_name} → {category_name})")
    elif account_name:
        parts.append(f"({account_name})")
    elif category_name:
        parts.append(f"({category_name})")

    reply = " ".join(parts)

    return {"reply": reply, "transaction": tx}
