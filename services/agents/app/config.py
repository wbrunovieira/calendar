from __future__ import annotations

import os


class Settings:
    # DeepSeek (OpenAI-compatible)
    deepseek_api_key: str = os.getenv("DEEPSEEK_API_KEY", "")
    deepseek_base_url: str = os.getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com")
    deepseek_model: str = os.getenv("DEEPSEEK_MODEL", "deepseek-chat")

    # calendar-finances internal URL (Docker network)
    finances_base_url: str = os.getenv("FINANCES_BASE_URL", "http://calendar-finances:3335")

    # Finance profile IDs
    finance_personal_profile_id: str = os.getenv("FINANCE_PERSONAL_PROFILE_ID", "")

    # Anthropic (Claude)
    anthropic_api_key: str = os.getenv("ANTHROPIC_API_KEY", "")
    anthropic_model: str = os.getenv("ANTHROPIC_MODEL", "claude-opus-4-6")
    anthropic_supervisor_model: str = os.getenv("ANTHROPIC_SUPERVISOR_MODEL", "claude-haiku-4-5-20251001")

    # WB-CRM (internal network)
    crm_base_url: str = os.getenv("CRM_BASE_URL", "http://host.docker.internal:3000")

    # Langfuse
    langfuse_public_key: str = os.getenv("LANGFUSE_PUBLIC_KEY", "")
    langfuse_secret_key: str = os.getenv("LANGFUSE_SECRET_KEY", "")
    langfuse_host: str = os.getenv("LANGFUSE_HOST", "http://langfuse-web:3000")


settings = Settings()
