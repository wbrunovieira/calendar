from __future__ import annotations

from typing import Any, TypedDict


class DeepResearchState(TypedDict, total=False):
    # Input
    lead_id: str
    requester_id: str
    lead: dict          # full lead data as received from CRM
    contacts: list[dict]
    previous_summary: str | None      # summary from a prior research run (re-research mode)
    previous_research_at: str | None  # ISO timestamp of the prior run
    focus_field: str | None           # focused research mode: target field
    custom_instruction: str | None    # extra instruction for focused mode
    _trace: Any

    # Research plan
    missing_fields: list[str]   # categories decided for research
    search_queries: list[str]   # Tavily queries to run
    research_note: str | None   # context note from assess_previous_research
    _skip_categories: list[str] # categories to skip (set by assess_previous_research)
    _extra_queries: list[str]   # extra queries from assess_previous_research

    # Raw results
    cnpj_raw: dict | None       # BrasilAPI response
    web_results: list[dict]     # Tavily combined results

    # Extracted & output
    updates: dict               # fields to patch on lead (only truly empty fields)
    forced_updates: dict        # focused mode: always overwrites regardless of existing value
    proposed_fields: dict       # fields agent found but lead already had a value (for audit)
    new_contacts: list[dict]
    instagram_insights: dict | None   # followers, posts, frequency, meta ads — for summary only
    summary: str
    error: str | None
