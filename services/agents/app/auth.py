"""Shared-token auth for every route, mirroring calendar-core's ApiTokenGuard.

Accepted forms: ``Authorization: Bearer <token>`` or ``X-API-Key: <token>``.

Exceptions:
- ``/health`` stays public for the health checks;
- CORS preflights pass (the CORS middleware sits outside this one and answers them);
- a blank ``AGENTS_API_TOKEN`` leaves the API open (logged as a warning at startup). That is deliberate: this ships before
  the callers (n8n, the CRM) send the token, and enforcement starts once it is set.

The token must be ASCII (e.g. ``openssl rand -hex 32``): headers are read as latin-1.
"""

from __future__ import annotations

import hmac

from starlette.responses import JSONResponse
from starlette.types import ASGIApp, Receive, Scope, Send

from app.config import settings

PUBLIC_PATHS = frozenset({"/health"})


def _extract_token(headers: dict[bytes, bytes]) -> str:
    auth = headers.get(b"authorization", b"").decode("latin-1")
    scheme, _, value = auth.partition(" ")
    if scheme == "Bearer" and value.strip():
        return value.strip()
    return headers.get(b"x-api-key", b"").decode("latin-1")


class ApiTokenMiddleware:
    def __init__(self, app: ASGIApp) -> None:
        self.app = app

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        # Everything but lifespan is checked, so a websocket route added later is covered too.
        if scope["type"] == "lifespan":
            await self.app(scope, receive, send)
            return

        # Read per request so the token can be rotated in tests without re-importing the app.
        expected = settings.agents_api_token.strip()
        if not expected or scope["path"] in PUBLIC_PATHS:
            await self.app(scope, receive, send)
            return

        provided = _extract_token(dict(scope["headers"]))
        # compare_digest runs in constant time, so a wrong token can't be guessed from timing.
        if not provided or not hmac.compare_digest(provided.encode(), expected.encode()):
            if scope["type"] == "websocket":
                # Closing before accept makes the server answer the handshake with 403.
                await send({"type": "websocket.close", "code": 1008})
                return
            response = JSONResponse(
                {"detail": "Invalid or missing API token"},
                status_code=401,
                headers={"WWW-Authenticate": 'Bearer realm="agents"'},
            )
            await response(scope, receive, send)
            return

        await self.app(scope, receive, send)
