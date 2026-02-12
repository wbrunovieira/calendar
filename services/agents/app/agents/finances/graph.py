from __future__ import annotations

from langgraph.graph import END, START, StateGraph

from app.agents.finances.nodes import (
    create_transaction,
    format_reply,
    load_context,
    parse_message,
    resolve_entities,
)
from app.agents.finances.state import TransactionState


def _after_parse(state: TransactionState) -> str:
    if state.get("error"):
        return END
    return "resolve_entities"


def _after_resolve(state: TransactionState) -> str:
    if state.get("error"):
        return END
    return "create_transaction"


def _after_create(state: TransactionState) -> str:
    if state.get("error"):
        return END
    return "format_reply"


def build_transaction_graph():
    builder = StateGraph(TransactionState)

    builder.add_node("load_context", load_context)
    builder.add_node("parse_message", parse_message)
    builder.add_node("resolve_entities", resolve_entities)
    builder.add_node("create_transaction", create_transaction)
    builder.add_node("format_reply", format_reply)

    builder.add_edge(START, "load_context")
    builder.add_edge("load_context", "parse_message")
    builder.add_conditional_edges("parse_message", _after_parse)
    builder.add_conditional_edges("resolve_entities", _after_resolve)
    builder.add_conditional_edges("create_transaction", _after_create)
    builder.add_edge("format_reply", END)

    return builder.compile()
