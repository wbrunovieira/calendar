from __future__ import annotations

import json
import logging

from langchain_core.messages import HumanMessage, SystemMessage
from langchain_openai import ChatOpenAI

from app.agents.transaction.nodes import langfuse
from app.config import settings

logger = logging.getLogger("calendar-agents")

EVAL_PROMPT = """\
Você é um avaliador de qualidade para um parser de lançamentos financeiros.

Recebeu uma mensagem de WhatsApp e o resultado do parsing. Avalie 3 critérios com nota 0 ou 1:

1. **correctness**: O JSON extraído corresponde fielmente à mensagem original?
   - A descrição, valor e conta (se mencionada) estão corretos?
   - 1 = correto, 0 = algum campo foi extraído errado

2. **category_match**: A categoria inferida faz sentido para a descrição?
   - "Almoço" → "Alimentação" = 1
   - "Uber" → "Renda" = 0
   - Se categoria é null, pontue 0

3. **completeness**: Todos os campos possíveis foram preenchidos?
   - Campos: description, amount, type, account_name, category_name
   - 1 = todos os campos que poderiam ser inferidos foram preenchidos
   - 0 = algum campo inferível ficou null

Responda APENAS com JSON válido:
{{"correctness": 0 | 1, "category_match": 0 | 1, "completeness": 0 | 1}}
"""

EVAL_USER = """\
Mensagem original: "{raw_text}"

Resultado do parsing:
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
