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

    # Langfuse
    langfuse_public_key: str = os.getenv("LANGFUSE_PUBLIC_KEY", "")
    langfuse_secret_key: str = os.getenv("LANGFUSE_SECRET_KEY", "")
    langfuse_host: str = os.getenv("LANGFUSE_HOST", "http://langfuse-web:3000")


settings = Settings()
