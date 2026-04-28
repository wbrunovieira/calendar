from __future__ import annotations

from typing import Any, TypedDict


class DeepResearchState(TypedDict, total=False):
    # Input
    lead_id: str
    requester_id: str
    lead: dict          # full lead data as received from CRM
    contacts: list[dict]
    _trace: Any

    # Research plan
    missing_fields: list[str]   # categories decided for research
    search_queries: list[str]   # Tavily queries to run

    # Raw results
    cnpj_raw: dict | None       # BrasilAPI response
    web_results: list[dict]     # Tavily combined results

    # Extracted & output
    updates: dict               # fields to patch on lead (only truly empty fields)
    proposed_fields: dict       # fields agent found but lead already had a value (for audit)
    new_contacts: list[dict]
    instagram_insights: dict | None   # followers, posts, frequency, meta ads — for summary only
    summary: str
    error: str | None
