from __future__ import annotations

import json
import logging

from langchain_core.messages import HumanMessage, SystemMessage
from langchain_openai import ChatOpenAI

from app.agents.finances.nodes import langfuse
from app.config import settings

logger = logging.getLogger("agents")

EVAL_PROMPT = """\
You are a quality evaluator for a financial transaction parser.

You received a WhatsApp message and the parsing result. Evaluate 3 criteria with a score of 0 or 1:

1. **correctness**: Does the extracted JSON match the intent of the original message?
   - Are the description, amount, and account (if mentioned) correct?
   - Typo correction is EXPECTED and CORRECT: "almso" → "Almoço", "nubak" → "Nubank" = 1
   - The parser should normalize names to their correct form, not copy errors
   - 1 = fields reflect the correct intent, 0 = some field was extracted incorrectly

2. **category_match**: Does the inferred category make sense for the description?
   - "Almoço" → "Alimentação" = 1
   - "Uber" → "Renda" = 0
   - If category is null, score 0

3. **completeness**: Were all possible fields filled in?
   - Fields: description, amount, type, account_name, category_name
   - 1 = all fields that could be inferred were filled
   - 0 = some inferrable field was left null

Respond ONLY with valid JSON:
{{"correctness": 0 | 1, "category_match": 0 | 1, "completeness": 0 | 1}}
"""

EVAL_USER = """\
Original message: "{raw_text}"

Parsing result:
{parsed_json}
"""


async def evaluate_trace(trace, raw_text: str, parsed: dict | None) -> None:
    """Run LLM-as-judge evaluation and submit scores to Langfuse trace."""
    if not trace or not parsed:
        return

    trace_id = trace.id

    try:
        llm = ChatOpenAI(
            api_key=settings.deepseek_api_key,
            base_url=settings.deepseek_base_url,
            model=settings.deepseek_model,
            temperature=0,
        )

        user_msg = EVAL_USER.format(
            raw_text=raw_text,
            parsed_json=json.dumps(parsed, ensure_ascii=False, indent=2),
        )

        generation = trace.generation(
            name="evaluator",
            model=settings.deepseek_model,
            input=[{"role": "system", "content": EVAL_PROMPT}, {"role": "user", "content": user_msg}],
            metadata={"eval_version": "v1"},
        )

        response = await llm.ainvoke([
            SystemMessage(content=EVAL_PROMPT),
            HumanMessage(content=user_msg),
        ])

        raw = response.content.strip()
        scores = json.loads(raw)

        generation.end(output=scores, metadata={"eval_status": "ok"})

        # Submit scores via langfuse client (not trace.score) for reliability
        for name in ("correctness", "category_match", "completeness"):
            value = scores.get(name)
            if value is not None:
                langfuse.score(
                    trace_id=trace_id,
                    name=name,
                    value=float(value),
                )

        logger.info("Eval scores for trace %s: %s", trace_id, scores)

    except Exception:
        logger.exception("Evaluator failed for trace %s", trace_id)
