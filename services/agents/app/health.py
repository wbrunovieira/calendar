from __future__ import annotations

from fastapi import APIRouter, Query, Request

router = APIRouter()


@router.get("/health")
async def health(request: Request, deep: int | None = Query(default=None)):
    langgraph_available: bool = request.app.state.langgraph_available
    compiled_graph = request.app.state.compiled_graph

    if deep == 1 and langgraph_available and compiled_graph is not None:
        from app.graph import verify_graph

        messages = verify_graph(compiled_graph)
        return {
            "status": "healthy",
            "service": "agents",
            "langgraph": "verified",
            "graph_result": messages,
        }

    return {
        "status": "healthy",
        "service": "agents",
        "langgraph": "available" if langgraph_available else "unavailable",
    }
