from __future__ import annotations

from typing import Any, TypedDict


class CRMLeadState(TypedDict, total=False):
    # Input
    query: str
    icp_id: str
    count: int
    _trace: Any

    # Context
    icp_context: dict | None
    existing_leads: list[str]

    # Investigation
    raw_dossiers: list[str]
    tavily_data: list[dict] | None

    # Structured
    leads_data: list[dict]
    contacts_data: list[list[dict]]

    # Supervisor
    approved_indices: list[int]
    rejected: list[dict]

    # Result
    created_leads: list[dict]
    reply: str
    error: str | None
