from __future__ import annotations

import logging

import httpx

from app.config import settings

logger = logging.getLogger("agents")


def _auth_headers() -> dict[str, str]:
    """Build auth headers for CRM API calls."""
    headers: dict[str, str] = {}
    if settings.crm_webhook_secret:
        headers["X-Webhook-Secret"] = settings.crm_webhook_secret
    return headers


async def get_icp(icp_id: str) -> dict:
    """GET /api/icps/{icp_id} — Fetch ICP from WB-CRM."""
    async with httpx.AsyncClient(base_url=settings.crm_base_url, timeout=10) as client:
        resp = await client.get(f"/api/icps/{icp_id}", headers=_auth_headers())
        if resp.status_code == 404:
            raise LookupError(f"ICP '{icp_id}' not found")
        resp.raise_for_status()
        return resp.json()


async def get_leads_by_icp(icp_id: str) -> list[dict]:
    """GET /api/leads/by-icp/{icp_id} — Fetch existing leads for an ICP."""
    async with httpx.AsyncClient(base_url=settings.crm_base_url, timeout=10) as client:
        resp = await client.get(f"/api/leads/by-icp/{icp_id}", headers=_auth_headers())
        if resp.status_code == 404:
            return []
        resp.raise_for_status()
        return resp.json()


async def create_lead(payload: dict) -> dict:
    """POST /api/leads — Create a lead in WB-CRM."""
    async with httpx.AsyncClient(base_url=settings.crm_base_url, timeout=15) as client:
        resp = await client.post("/api/leads", json=payload, headers=_auth_headers())
        if resp.status_code >= 400:
            error_body = resp.text.strip()
            raise RuntimeError(error_body or f"HTTP {resp.status_code}")
        return resp.json()


async def link_lead_to_icp(lead_id: str, icp_id: str, notes: str = "Created via AI Research") -> dict:
    """POST /api/leads/{lead_id}/icps — Link lead to ICP."""
    async with httpx.AsyncClient(base_url=settings.crm_base_url, timeout=10) as client:
        resp = await client.post(
            f"/api/leads/{lead_id}/icps",
            json={"icpId": icp_id, "notes": notes},
            headers=_auth_headers(),
        )
        if resp.status_code >= 400:
            logger.warning("Failed to link lead %s to ICP %s: %s", lead_id, icp_id, resp.text.strip())
            return {}
        return resp.json()


async def create_lead_contact(lead_id: str, payload: dict) -> dict:
    """POST /api/leads/{lead_id}/contacts — Add contact to a lead."""
    async with httpx.AsyncClient(base_url=settings.crm_base_url, timeout=10) as client:
        resp = await client.post(f"/api/leads/{lead_id}/contacts", json=payload, headers=_auth_headers())
        if resp.status_code >= 400:
            error_body = resp.text.strip()
            raise RuntimeError(error_body or f"HTTP {resp.status_code}")
        return resp.json()


async def send_webhook(payload: dict) -> None:
    """POST /api/webhooks/lead-research — Notify CRM that research is done."""
    url = f"{settings.crm_base_url}/api/webhooks/lead-research"
    async with httpx.AsyncClient(timeout=10) as client:
        try:
            resp = await client.post(url, json=payload, headers=_auth_headers())
            resp.raise_for_status()
        except Exception:
            logger.warning("Webhook callback failed (CRM may not have the endpoint yet): %s", url)
