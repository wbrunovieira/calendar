from __future__ import annotations

import logging
from contextlib import asynccontextmanager

from dotenv import load_dotenv
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

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

app.add_middleware(
    CORSMiddleware,
    allow_origins=[
        "https://crm.wbdigitalsolutions.com",
        "https://calendar.wbdigitalsolutions.com",
        "https://finances.wbdigitalsolutions.com",
        "https://projects.wbdigitalsolutions.com",
        "http://localhost:3000",
        "http://localhost:3003",
    ],
    allow_methods=["*"],
    allow_headers=["*"],
)

from app.health import router as health_router  # noqa: E402
from app.agents.finances.router import router as transaction_router  # noqa: E402
from app.agents.crm.router import router as crm_router  # noqa: E402
from app.agents.crm.deep_research.router import router as deep_research_router  # noqa: E402
from app.agents.crm.call_analysis.router import router as call_analysis_router  # noqa: E402
from app.agents.crm.meet_analysis.router import router as meet_analysis_router  # noqa: E402

app.include_router(health_router)
app.include_router(transaction_router)
app.include_router(crm_router)
app.include_router(deep_research_router)
app.include_router(call_analysis_router)
app.include_router(meet_analysis_router)
