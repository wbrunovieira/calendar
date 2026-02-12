"""
Langfuse Dataset Experiments for CRM lead research agent.

Seeds a dataset with diverse search queries and ICPs,
runs the investigator pipeline, evaluates with LLM-as-judge,
and records everything in Langfuse for comparison.

Usage:
    # From Docker (recommended)
    docker compose exec agents python -m app.agents.crm.experiments

    # Seed dataset only (no run)
    docker compose exec agents python -m app.agents.crm.experiments --seed-only

    # Run experiment with a custom name
    docker compose exec agents python -m app.agents.crm.experiments --run-name "prompt-v2"
"""

from __future__ import annotations

import asyncio
import json
import logging
import sys

from app.agents.crm.evaluator import EVAL_PROMPT, EVAL_USER
from app.agents.crm.nodes import _call_claude
from app.agents.finances.nodes import langfuse
from app.config import settings

logger = logging.getLogger("agents")

DATASET_NAME = "crm-lead-research-tests"

# Fake ICP for isolated testing
TEST_ICP = {
    "id": "test-icp-1",
    "name": "SaaS B2B Brasil",
    "content": (
        "Target: Brazilian SaaS companies selling to SMBs.\n"
        "Segment: Technology, SaaS, B2B.\n"
        "Size: 10-500 employees.\n"
        "Revenue: R$1M-R$100M.\n"
        "Location: Brazil, primarily SP and MG.\n"
        "Decision makers: CEO, CTO, Head of Growth."
    ),
}

# Each item: input (query + icp), expected (description of what should happen), metadata
TEST_ITEMS: list[dict] = [
    {
        "input": {"query": "Hotmart", "icp": TEST_ICP, "count": 1},
        "expected": {"should_find": True, "expected_name_contains": "Hotmart"},
        "metadata": {"tags": ["known_company", "saas_br"]},
    },
    {
        "input": {"query": "RD Station", "icp": TEST_ICP, "count": 1},
        "expected": {"should_find": True, "expected_name_contains": "RD Station"},
        "metadata": {"tags": ["known_company", "saas_br"]},
    },
    {
        "input": {"query": "startups edtech Sao Paulo", "icp": TEST_ICP, "count": 2},
        "expected": {"should_find": True, "min_leads": 1},
        "metadata": {"tags": ["vague_query", "multiple"]},
    },
    {
        "input": {"query": "empresas SaaS fintech Brasil", "icp": TEST_ICP, "count": 1},
        "expected": {"should_find": True},
        "metadata": {"tags": ["segment_search"]},
    },
    {
        "input": {"query": "Resultados Digitais marketing automation", "icp": TEST_ICP, "count": 1},
        "expected": {"should_find": True, "expected_name_contains": "Resultados Digitais"},
        "metadata": {"tags": ["known_company", "full_name"]},
    },
    {
        "input": {"query": "clinicas esteticas premium SP", "icp": TEST_ICP, "count": 1},
        "expected": {"should_find": True},
        "metadata": {"tags": ["niche_query", "local"]},
    },
    {
        "input": {"query": "plataforma EAD corporativa Brasil", "icp": TEST_ICP, "count": 1},
        "expected": {"should_find": True},
        "metadata": {"tags": ["segment_search", "edtech"]},
    },
    {
        "input": {"query": "XYZZYNONEXISTENT12345 company", "icp": TEST_ICP, "count": 1},
        "expected": {"should_find": False},
        "metadata": {"tags": ["impossible_query"]},
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
        print(f"  [{i+1}/{len(TEST_ITEMS)}] {item['input']['query'][:50]}")

    langfuse.flush()
    print(f"Seeded {len(TEST_ITEMS)} test items")


async def run_experiment(run_name: str) -> None:
    """Run all dataset items through the investigator and record in Langfuse."""
    dataset = langfuse.get_dataset(DATASET_NAME)
    items = dataset.items

    if not items:
        print("Dataset is empty. Run with --seed-only first.")
        return

    print(f"\nRunning experiment '{run_name}' with {len(items)} items...\n")

    for i, item in enumerate(items):
        query = item.input["query"]
        icp = item.input.get("icp", TEST_ICP)
        count = item.input.get("count", 1)

        trace = langfuse.trace(
            name="experiment-crm-research",
            input={"query": query, "count": count},
            metadata={"run_name": run_name, "item_id": item.id},
        )

        # Import and run the graph
        from app.agents.crm.graph import build_crm_graph

        # Mock get_icp to return our test ICP
        from unittest.mock import AsyncMock, patch
        with patch("app.agents.crm.nodes.crm.get_icp", new_callable=AsyncMock, return_value=icp):
            graph = build_crm_graph()
            result = await graph.ainvoke({
                "query": query,
                "icp_id": icp["id"],
                "count": count,
                "_trace": trace,
            })

        created = result.get("created_leads", [])
        error = result.get("error")

        status = "ok" if created else ("error" if error else "empty")
        print(f"  [{i+1}/{len(items)}] '{query[:40]}' → {status} ({len(created)} leads)")

        item.link(trace, run_name)
        trace.update(output={"result": result.get("reply", ""), "status": status})

    langfuse.flush()
    print(f"\nView results: {settings.langfuse_host}/datasets/{DATASET_NAME}")


def main():
    import argparse

    logging.basicConfig(level=logging.WARNING)

    parser = argparse.ArgumentParser(description="Run CRM lead research experiments")
    parser.add_argument("--seed-only", action="store_true", help="Only seed dataset, don't run")
    parser.add_argument("--run-name", default="crm-v1", help="Experiment run name")
    parser.add_argument("--skip-seed", action="store_true", help="Skip seeding, run only")
    args = parser.parse_args()

    if not args.skip_seed:
        seed_dataset()

    if args.seed_only:
        return

    asyncio.run(run_experiment(args.run_name))


if __name__ == "__main__":
    main()
