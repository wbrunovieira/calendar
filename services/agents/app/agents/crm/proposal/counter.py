"""Sequential proposal number counter — persists in /app/data/. Per-brand counters."""
from __future__ import annotations

import asyncio
import json
import logging
import os
from datetime import datetime, timezone

logger = logging.getLogger("agents")

_DATA_DIR = "/app/data"
_lock = asyncio.Lock()

_COUNTER_FILES = {
    "wb":    os.path.join(_DATA_DIR, "proposal_counter_wb.json"),
    "salto": os.path.join(_DATA_DIR, "proposal_counter_salto.json"),
}

_FORMATS = {
    "wb":    "WB-P{num:03d}-{mm}{yy}-REV1",
    "salto": "SALTO-P{num:03d}-{mm}{yy}-REV1",
}


async def next_proposal_number(brand: str = "wb") -> str:
    """Returns next proposal number for the given brand.

    WB:    WB-P027-0526-REV1
    Salto: SALTO-P001-0526-REV1
    """
    counter_file = _COUNTER_FILES.get(brand, _COUNTER_FILES["wb"])
    fmt = _FORMATS.get(brand, _FORMATS["wb"])

    os.makedirs(_DATA_DIR, exist_ok=True)
    async with _lock:
        current = 0
        if os.path.exists(counter_file):
            try:
                with open(counter_file, encoding="utf-8") as f:
                    current = json.load(f).get("last_number", 0)
            except Exception:
                logger.warning("Could not read proposal counter for brand=%s, starting from 1", brand)

        next_num = current + 1
        with open(counter_file, "w", encoding="utf-8") as f:
            json.dump({"last_number": next_num}, f)

    now = datetime.now(timezone.utc)
    return fmt.format(num=next_num, mm=now.strftime("%m"), yy=now.strftime("%y"))
