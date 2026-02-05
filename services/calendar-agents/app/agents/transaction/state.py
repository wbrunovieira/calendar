from __future__ import annotations

from typing import Any, TypedDict


class TransactionState(TypedDict, total=False):
    # Input
    phone: str
    raw_text: str
    profile_id: str

    # Langfuse trace (passed through, not serialized)
    _trace: Any

    # Context (loaded from calendar-finances)
    accounts: list[dict]
    categories: list[dict]

    # Parsed by LLM
    parsed: dict | None

    # Resolved IDs
    bank_account_id: str | None
    category_id: str | None

    # Result
    transaction: dict | None
    reply: str
    error: str | None
