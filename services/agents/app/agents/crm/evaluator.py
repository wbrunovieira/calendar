from __future__ import annotations

import json
import logging

from app.agents.crm.nodes import _call_claude
from app.agents.finances.nodes import langfuse
from app.config import settings

logger = logging.getLogger("agents")

EVAL_PROMPT = """\
You are a quality evaluator for a B2B lead research agent.

You received a lead research result. Evaluate 3 criteria with a score of 0 or 1:

1. **data_quality**: Is the data real and not fabricated?
   - Business name looks like a real company = 1
   - Contact info (email, phone, website) appears genuine = 1
   - Data that looks obviously invented or placeholder = 0

2. **completeness**: Does the lead have sufficient data?
   - businessName + at least one contact method (email OR phone OR website) + at least 1 contact with name = 1
   - Missing businessName or no contact info at all = 0

3. **icp_match**: Does the lead correspond to the ICP?
   - Company segment/industry matches ICP criteria = 1
   - Company is clearly outside the ICP target = 0

Respond ONLY with valid JSON:
{{"data_quality": 0 | 1, "completeness": 0 | 1, "icp_match": 0 | 1}}
"""

EVAL_USER = """\
## ICP Context
{icp_context}

## Lead
{lead_json}

## Contacts
{contacts_json}
"""


async def evaluate_lead_trace(trace, icp_context: dict, lead: dict, contacts: list[dict]) -> None:
    """Run LLM-as-judge evaluation and submit scores to Langfuse trace."""
    if not trace or not lead:
        return

    trace_id = trace.id
    icp_text = f"Name: {icp_context.get('name', '')}\n{icp_context.get('content', '')}"

    try:
        user_msg = EVAL_USER.format(
            icp_context=icp_text,
            lead_json=json.dumps(lead, ensure_ascii=False, indent=2),
            contacts_json=json.dumps(contacts, ensure_ascii=False, indent=2),
        )

        generation = trace.generation(
            name="evaluator",
            model=settings.anthropic_supervisor_model,
            input=[{"role": "system", "content": EVAL_PROMPT}, {"role": "user", "content": user_msg}],
            metadata={"eval_version": "v1"},
        )

        response = await _call_claude(
            EVAL_PROMPT, user_msg,
            model=settings.anthropic_supervisor_model,
        )

        raw = "".join(b.text for b in response.content if b.type == "text").strip()
        scores = json.loads(raw)

        generation.end(output=scores, metadata={"eval_status": "ok"})

        for name in ("data_quality", "completeness", "icp_match"):
            value = scores.get(name)
            if value is not None:
                langfuse.score(trace_id=trace_id, name=name, value=float(value))

        logger.info("CRM eval scores for trace %s: %s", trace_id, scores)

    except Exception:
        logger.exception("CRM evaluator failed for trace %s", trace_id)
