from __future__ import annotations

from typing import Any

from typing_extensions import TypedDict


class ProposalState(TypedDict, total=False):
    # ── Input ─────────────────────────────────────────────────────────────
    job_id: str        # = proposalId sent by CRM (used as correlation key)
    proposal_id: str   # CRM's proposalId — echoed back in all webhook calls
    webhook_url: str
    brand: str          # "wb" | "salto"
    lead: dict
    contacts: list
    products: list
    user_instructions: str
    qa_history: list  # [{"question": "...", "answer": "..."}]
    _trace: Any

    # ── Node 1: analyze_completeness (Haiku) ──────────────────────────────
    is_sufficient: bool
    question: str | None

    # ── Nodes 2a/2b/2c/2d: parallel generation ───────────────────────────
    # 2a — write_narrative (Sonnet): texto personalizado
    narrative_sections: dict   # PROJECT_TITLE, OPENING_P_1/2, SCOPE_INTRO, CLOSING_P_1
    narrative_error: str | None

    # 2b — write_technical (Sonnet): stack técnica e funcionalidades
    technical_sections: dict   # TECH_SECTIONS, FEATURES_LIST
    technical_error: str | None

    # 2c — write_commercial (Haiku): tabelas, valores, considerações
    commercial_sections: dict  # INVESTMENT_TABLE, PAYMENT_SCHEDULE, ADDITIONAL_CONSIDERATIONS_ITEMS
    commercial_error: str | None

    # 2d — write_optional_sections (Haiku): seções opcionais — retorna HTML completo ou ""
    optional_sections: dict    # PLANNING_SECTION, METHODOLOGY_SECTION, POST_LAUNCH_SECTION,
    optional_error: str | None #   PENALTY_CLAUSE, LGPD_SECTION, FUTURE_EVOLUTION_SECTION

    # ── Node 3: merge_sections (determinístico) ────────────────────────────
    proposal_number: str
    project_title: str
    filled_html: str

    # ── Node 4: convert_to_pdf ────────────────────────────────────────────
    file_name: str
    file_size: int
    pdf_base64: str   # base64-encoded PDF sent to CRM in completed webhook

    # ── Error global ──────────────────────────────────────────────────────
    error: str | None
