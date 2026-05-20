"""Job context persistence — stores state between /proposal-agent and /answer calls."""
from __future__ import annotations

import asyncio
import json
import logging
import os

logger = logging.getLogger("agents")

_JOBS_DIR = "/tmp/proposal_jobs"
_lock = asyncio.Lock()


def _job_path(job_id: str) -> str:
    return os.path.join(_JOBS_DIR, f"{job_id}.json")


async def save_job(job_id: str, data: dict) -> None:
    os.makedirs(_JOBS_DIR, exist_ok=True)
    async with _lock:
        with open(_job_path(job_id), "w", encoding="utf-8") as f:
            json.dump(data, f, ensure_ascii=False, indent=2)


async def load_job(job_id: str) -> dict | None:
    path = _job_path(job_id)
    if not os.path.exists(path):
        return None
    async with _lock:
        with open(path, encoding="utf-8") as f:
            return json.load(f)


async def delete_job(job_id: str) -> None:
    path = _job_path(job_id)
    if os.path.exists(path):
        async with _lock:
            try:
                os.remove(path)
            except FileNotFoundError:
                pass
