"""
Langfuse Dataset Experiments for transaction-parser prompt.

Seeds a dataset with diverse WhatsApp message variations,
runs the parser against each, evaluates with LLM-as-judge,
and records everything in Langfuse for comparison.

Usage:
    # From Docker (recommended)
    docker compose exec agents python -m app.agents.transaction.experiments

    # Seed dataset only (no run)
    docker compose exec agents python -m app.agents.transaction.experiments --seed-only

    # Run experiment with a custom name
    docker compose exec agents python -m app.agents.transaction.experiments --run-name "prompt-v2"
"""

from __future__ import annotations

import asyncio
import json
import logging
import sys
from datetime import date

from langchain_core.messages import HumanMessage, SystemMessage
from langchain_openai import ChatOpenAI

from app.agents.transaction.evaluator import EVAL_PROMPT, EVAL_USER
from app.agents.transaction.nodes import langfuse, _build_messages
from app.config import settings

logger = logging.getLogger("agents")

# ── Test dataset ─────────────────────────────────────────────

DATASET_NAME = "transaction-parser-tests"

# Fake accounts/categories for isolated testing (no API dependency)
TEST_ACCOUNTS = [
    {"id": "acc-1", "name": "Nubank"},
    {"id": "acc-2", "name": "Itaú"},
    {"id": "acc-3", "name": "C6 Bank"},
    {"id": "acc-4", "name": "Carteira"},
]

TEST_CATEGORIES = [
    {"id": "cat-1", "name": "Alimentação", "type": "EXPENSE"},
    {"id": "cat-2", "name": "Transporte", "type": "EXPENSE"},
    {"id": "cat-3", "name": "Saúde", "type": "EXPENSE"},
    {"id": "cat-4", "name": "Lazer", "type": "EXPENSE"},
    {"id": "cat-5", "name": "Moradia", "type": "EXPENSE"},
    {"id": "cat-6", "name": "Educação", "type": "EXPENSE"},
    {"id": "cat-7", "name": "Renda", "type": "INCOME"},
    {"id": "cat-8", "name": "Vestuário", "type": "EXPENSE"},
    {"id": "cat-9", "name": "Assinaturas", "type": "EXPENSE"},
]

# Each item: (raw_text, expected_output_or_null, tags)
# expected_output is for reference in Langfuse UI, scoring is done by evaluator
TEST_ITEMS: list[dict] = [
    # ── Basic cases ──────────────────────────────────────────
    {
        "input": {"text": "almoco 32 nubank"},
        "expected": {"description": "Almoço", "amount": 32.0, "type": "EXPENSE", "account_name": "Nubank", "category_name": "Alimentação"},
        "metadata": {"tags": ["basic", "with_account"]},
    },
    {
        "input": {"text": "uber 18.50"},
        "expected": {"description": "Uber", "amount": 18.5, "type": "EXPENSE", "account_name": None, "category_name": "Transporte"},
        "metadata": {"tags": ["basic", "no_account"]},
    },
    {
        "input": {"text": "salario 8000 itau"},
        "expected": {"description": "Salário", "amount": 8000.0, "type": "INCOME", "account_name": "Itaú", "category_name": "Renda"},
        "metadata": {"tags": ["income"]},
    },
    {
        "input": {"text": "mercado 230 nubank alimentacao"},
        "expected": {"description": "Mercado", "amount": 230.0, "type": "EXPENSE", "account_name": "Nubank", "category_name": "Alimentação"},
        "metadata": {"tags": ["basic", "with_category"]},
    },

    # ── Amount formats ───────────────────────────────────────
    {
        "input": {"text": "gasolina 150,00 c6 bank"},
        "expected": {"description": "Gasolina", "amount": 150.0, "type": "EXPENSE", "account_name": "C6 Bank"},
        "metadata": {"tags": ["amount_format", "comma"]},
    },
    {
        "input": {"text": "R$ 45,90 farmacia nubank"},
        "expected": {"description": "Farmácia", "amount": 45.9, "type": "EXPENSE", "account_name": "Nubank"},
        "metadata": {"tags": ["amount_format", "r$_prefix"]},
    },
    {
        "input": {"text": "lanche 12 reais nubank"},
        "expected": {"description": "Lanche", "amount": 12.0, "type": "EXPENSE", "account_name": "Nubank"},
        "metadata": {"tags": ["amount_format", "reais_suffix"]},
    },

    # ── Date variations ──────────────────────────────────────
    {
        "input": {"text": "almoco ontem 25 nubank"},
        "expected": {"description": "Almoço", "amount": 25.0, "type": "EXPENSE", "account_name": "Nubank"},
        "metadata": {"tags": ["date", "relative"]},
    },
    {
        "input": {"text": "mercado dia 3 230 nubank"},
        "expected": {"description": "Mercado", "amount": 230.0, "type": "EXPENSE", "account_name": "Nubank"},
        "metadata": {"tags": ["date", "day_reference"]},
    },

    # ── Description complexity ───────────────────────────────
    {
        "input": {"text": "cinema com amigos 45 nubank"},
        "expected": {"description": "Cinema com amigos", "amount": 45.0, "type": "EXPENSE", "account_name": "Nubank", "category_name": "Lazer"},
        "metadata": {"tags": ["description", "multi_word"]},
    },
    {
        "input": {"text": "presente aniversario mae 120 itau"},
        "expected": {"description": "Presente aniversário mãe", "amount": 120.0, "type": "EXPENSE", "account_name": "Itaú"},
        "metadata": {"tags": ["description", "multi_word"]},
    },
    {
        "input": {"text": "consulta medica 250 nubank saude"},
        "expected": {"description": "Consulta médica", "amount": 250.0, "type": "EXPENSE", "account_name": "Nubank", "category_name": "Saúde"},
        "metadata": {"tags": ["description", "with_category"]},
    },

    # ── Unparseable / edge cases ─────────────────────────────
    {
        "input": {"text": "bom dia"},
        "expected": None,
        "metadata": {"tags": ["unparseable", "greeting"]},
    },
    {
        "input": {"text": "ok"},
        "expected": None,
        "metadata": {"tags": ["unparseable", "short"]},
    },
    {
        "input": {"text": "quanto gastei esse mes?"},
        "expected": None,
        "metadata": {"tags": ["unparseable", "question"]},
    },

    # ── Income variations ────────────────────────────────────
    {
        "input": {"text": "recebi freelance 2500 nubank"},
        "expected": {"description": "Freelance", "amount": 2500.0, "type": "INCOME", "account_name": "Nubank", "category_name": "Renda"},
        "metadata": {"tags": ["income", "freelance"]},
    },
    {
        "input": {"text": "pix do joao 150 itau"},
        "expected": {"description": "Pix do João", "amount": 150.0, "type": "INCOME", "account_name": "Itaú"},
        "metadata": {"tags": ["income", "ambiguous"]},
    },

    # ── Typos / informal ─────────────────────────────────────
    {
        "input": {"text": "almso 32 nubak"},
        "expected": {"description": "Almoço", "amount": 32.0, "type": "EXPENSE", "account_name": "Nubank"},
        "metadata": {"tags": ["typo", "account_typo"]},
    },
    {
        "input": {"text": "ifood 28,90"},
        "expected": {"description": "iFood", "amount": 28.9, "type": "EXPENSE", "category_name": "Alimentação"},
        "metadata": {"tags": ["informal", "brand"]},
    },

    # ── Subscriptions / recurring ────────────────────────────
    {
        "input": {"text": "netflix 55.90 nubank"},
        "expected": {"description": "Netflix", "amount": 55.9, "type": "EXPENSE", "account_name": "Nubank", "category_name": "Assinaturas"},
        "metadata": {"tags": ["subscription"]},
    },
    {
        "input": {"text": "spotify 21.90 c6 bank"},
        "expected": {"description": "Spotify", "amount": 21.9, "type": "EXPENSE", "account_name": "C6 Bank", "category_name": "Assinaturas"},
        "metadata": {"tags": ["subscription"]},
    },
]


def seed_dataset() -> None:
    """Create or update the test dataset in Langfuse."""
    try:
        dataset = langfuse.get_dataset(DATASET_NAME)
        print(f"Dataset '{DATASET_NAME}' already exists ({len(dataset.items)} items)")
        return
    except Exception:
        pass

    langfuse.create_dataset(name=DATASET_NAME)
    print(f"Created dataset '{DATASET_NAME}'")

    for i, item in enumerate(TEST_ITEMS):
        langfuse.create_dataset_item(
            dataset_name=DATASET_NAME,
            input=item["input"],
            expected_output=item["expected"],
            metadata=item["metadata"],
        )
        print(f"  [{i+1}/{len(TEST_ITEMS)}] {item['input']['text'][:50]}")

    langfuse.flush()
    print(f"Seeded {len(TEST_ITEMS)} test items")


async def run_single(text: str, llm: ChatOpenAI) -> dict | None:
    """Run parse_message logic against a single text. Returns parsed dict or None."""
    accounts_list = "\n".join(f"- {a['name']}" for a in TEST_ACCOUNTS)
    categories_list = "\n".join(f"- {c['name']} ({c['type']})" for c in TEST_CATEGORIES)
    today = date.today().isoformat()

    system, user, lf_prompt = _build_messages(
        raw_text=text,
        today=today,
        accounts_list=accounts_list,
        categories_list=categories_list,
    )

    response = await llm.ainvoke([
        SystemMessage(content=system),
        HumanMessage(content=user),
    ])
    raw = response.content.strip()

    if raw.lower() == "null" or not raw:
        return None

    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return None


async def evaluate_single(llm: ChatOpenAI, raw_text: str, parsed: dict) -> dict:
    """Run LLM-as-judge on a single result. Returns scores dict."""
    user_msg = EVAL_USER.format(
        raw_text=raw_text,
        parsed_json=json.dumps(parsed, ensure_ascii=False, indent=2),
    )
    response = await llm.ainvoke([
        SystemMessage(content=EVAL_PROMPT),
        HumanMessage(content=user_msg),
    ])
    try:
        return json.loads(response.content.strip())
    except json.JSONDecodeError:
        return {"correctness": 0, "category_match": 0, "completeness": 0}


async def run_experiment(run_name: str) -> None:
    """Run all dataset items through the parser and record in Langfuse."""
    dataset = langfuse.get_dataset(DATASET_NAME)
    items = dataset.items

    if not items:
        print("Dataset is empty. Run with --seed-only first.")
        return

    llm = ChatOpenAI(
        api_key=settings.deepseek_api_key,
        base_url=settings.deepseek_base_url,
        model=settings.deepseek_model,
        temperature=0,
    )

    total_scores = {"correctness": 0.0, "category_match": 0.0, "completeness": 0.0}
    scored_count = 0
    null_correct = 0
    null_total = 0

    print(f"\nRunning experiment '{run_name}' with {len(items)} items...\n")

    for i, item in enumerate(items):
        text = item.input["text"]
        expected = item.expected_output

        # Create trace
        trace = langfuse.trace(
            name="experiment-parse",
            input={"text": text},
            metadata={"run_name": run_name, "item_id": item.id},
        )

        # Parse
        generation = trace.generation(
            name="parse_message",
            model=settings.deepseek_model,
            input=[{"role": "user", "content": text}],
        )

        parsed = await run_single(text, llm)

        generation.end(output=parsed or "null")

        # Handle null expected (unparseable messages)
        if expected is None:
            null_total += 1
            is_correct = parsed is None
            if is_correct:
                null_correct += 1
            trace.score(name="null_detection", value=1.0 if is_correct else 0.0)
            status = "PASS" if is_correct else "FAIL (should be null)"
            print(f"  [{i+1}/{len(items)}] '{text[:40]}' → null? {status}")
        elif parsed is not None:
            # Evaluate with LLM-as-judge
            scores = await evaluate_single(llm, text, parsed)
            for name, value in scores.items():
                langfuse.score(trace_id=trace.id, name=name, value=float(value))
                total_scores[name] += float(value)
            scored_count += 1

            score_str = " ".join(f"{k}={v}" for k, v in scores.items())
            print(f"  [{i+1}/{len(items)}] '{text[:40]}' → {score_str}")
        else:
            # Expected output but got null
            trace.score(name="correctness", value=0.0)
            trace.score(name="completeness", value=0.0)
            print(f"  [{i+1}/{len(items)}] '{text[:40]}' → FAIL (got null, expected output)")

        # Link to dataset item for Langfuse experiment UI
        item.link(trace, run_name)

        trace.update(output={"parsed": parsed, "expected": expected})

    langfuse.flush()

    # Summary
    print(f"\n{'='*50}")
    print(f"  Experiment: {run_name}")
    print(f"{'='*50}")
    if scored_count > 0:
        for name, total in total_scores.items():
            avg = total / scored_count
            print(f"  {name}: {avg:.0%} ({int(total)}/{scored_count})")
    if null_total > 0:
        print(f"  null_detection: {null_correct/null_total:.0%} ({null_correct}/{null_total})")
    print(f"{'='*50}")
    print(f"\nView results: {settings.langfuse_host}/datasets/{DATASET_NAME}")


def main():
    import argparse

    logging.basicConfig(level=logging.WARNING)

    parser = argparse.ArgumentParser(description="Run transaction-parser experiments")
    parser.add_argument("--seed-only", action="store_true", help="Only seed dataset, don't run")
    parser.add_argument("--run-name", default="prompt-v1", help="Experiment run name (default: prompt-v1)")
    parser.add_argument("--skip-seed", action="store_true", help="Skip seeding, run only")
    args = parser.parse_args()

    if not args.skip_seed:
        seed_dataset()

    if args.seed_only:
        return

    asyncio.run(run_experiment(args.run_name))


if __name__ == "__main__":
    main()
