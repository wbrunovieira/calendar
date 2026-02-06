from __future__ import annotations

import logging
from contextlib import asynccontextmanager

from dotenv import load_dotenv
from fastapi import FastAPI

load_dotenv()

logging.basicConfig(level=logging.INFO, format="%(levelname)s:%(name)s: %(message)s")
logger = logging.getLogger("agents")


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: import and compile graph, but never execute it
    try:
        from app.graph import build_graph

        app.state.compiled_graph = build_graph()
        app.state.langgraph_available = True
        logger.info("LangGraph compiled successfully")
    except Exception:
        app.state.compiled_graph = None
        app.state.langgraph_available = False
        logger.exception("Failed to compile LangGraph")

    yield


app = FastAPI(title="agents", lifespan=lifespan)

from app.health import router as health_router  # noqa: E402
from app.agents.transaction.router import router as transaction_router  # noqa: E402

app.include_router(health_router)
app.include_router(transaction_router)
