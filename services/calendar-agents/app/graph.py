from __future__ import annotations

from typing import TypedDict

from langgraph.graph import StateGraph, START, END


class PingPongState(TypedDict):
    messages: list[str]


def ping(state: PingPongState) -> PingPongState:
    return {"messages": state["messages"] + ["ping"]}


def pong(state: PingPongState) -> PingPongState:
    return {"messages": state["messages"] + ["pong"]}


def build_graph() -> StateGraph:
    """Build and return the compiled ping-pong graph without executing it."""
    builder = StateGraph(PingPongState)
    builder.add_node("ping", ping)
    builder.add_node("pong", pong)
    builder.add_edge(START, "ping")
    builder.add_edge("ping", "pong")
    builder.add_edge("pong", END)
    return builder.compile()


def verify_graph(graph) -> list[str]:
    """Execute the ping-pong graph and return the resulting messages."""
    result = graph.invoke({"messages": []})
    return result["messages"]
