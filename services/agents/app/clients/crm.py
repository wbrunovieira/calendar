from __future__ import annotations

import httpx

from app.config import settings


async def get_icp(icp_id: str) -> dict:
    """GET /api/icps/{icp_id} — Fetch ICP from WB-CRM."""
    async with httpx.AsyncClient(base_url=settings.crm_base_url, timeout=10) as client:
        resp = await client.get(f"/api/icps/{icp_id}")
        if resp.status_code == 404:
            raise LookupError(f"ICP '{icp_id}' not found")
        resp.raise_for_status()
        return resp.json()


async def create_lead(payload: dict) -> dict:
    """POST /api/leads — Create a lead in WB-CRM."""
    async with httpx.AsyncClient(base_url=settings.crm_base_url, timeout=15) as client:
        resp = await client.post("/api/leads", json=payload)
        if resp.status_code >= 400:
            error_body = resp.text.strip()
            raise RuntimeError(error_body or f"HTTP {resp.status_code}")
        return resp.json()


async def create_lead_contact(lead_id: str, payload: dict) -> dict:
    """POST /api/leads/{lead_id}/contacts — Add contact to a lead."""
    async with httpx.AsyncClient(base_url=settings.crm_base_url, timeout=10) as client:
        resp = await client.post(f"/api/leads/{lead_id}/contacts", json=payload)
        if resp.status_code >= 400:
            error_body = resp.text.strip()
            raise RuntimeError(error_body or f"HTTP {resp.status_code}")
        return resp.json()


async def send_webhook(payload: dict) -> None:
    """POST /api/webhooks/lead-research — Notify CRM that research is done."""
    url = f"{settings.crm_base_url}/api/webhooks/lead-research"
    async with httpx.AsyncClient(timeout=10) as client:
        try:
            resp = await client.post(url, json=payload)
            resp.raise_for_status()
        except Exception:
            import logging
            logging.getLogger("agents").warning(
                "Webhook callback failed (CRM may not have the endpoint yet): %s", url,
            )
