from __future__ import annotations

import httpx

from app.config import settings


async def get_bank_accounts(profile_id: str) -> list[dict]:
    async with httpx.AsyncClient(base_url=settings.finances_base_url, timeout=10) as client:
        resp = await client.get("/api/v1/bank-accounts", params={"profileId": profile_id})
        resp.raise_for_status()
        body = resp.json()
        return body.get("data", [])


async def get_categories(profile_id: str, category_type: str | None = None) -> list[dict]:
    params: dict[str, str] = {"profileId": profile_id}
    if category_type:
        params["type"] = category_type
    async with httpx.AsyncClient(base_url=settings.finances_base_url, timeout=10) as client:
        resp = await client.get("/api/v1/categories", params=params)
        resp.raise_for_status()
        body = resp.json()
        return body.get("data", [])


async def create_transaction(payload: dict) -> dict:
    async with httpx.AsyncClient(base_url=settings.finances_base_url, timeout=10) as client:
        resp = await client.post("/api/v1/transactions", json=payload)
        if resp.status_code >= 400:
            error_body = resp.text.strip()
            raise RuntimeError(error_body or f"HTTP {resp.status_code}")
        return resp.json()
