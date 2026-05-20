"""Google Drive upload helper for proposal PDFs."""
from __future__ import annotations

import asyncio
import json
import logging
import os

logger = logging.getLogger("agents")

_SCOPES = ["https://www.googleapis.com/auth/drive.file"]


def _get_credentials():
    from google.oauth2 import service_account
    from app.config import settings

    if settings.google_service_account_json:
        info = json.loads(settings.google_service_account_json)
        return service_account.Credentials.from_service_account_info(info, scopes=_SCOPES)

    if settings.google_service_account_path and os.path.exists(settings.google_service_account_path):
        return service_account.Credentials.from_service_account_file(
            settings.google_service_account_path, scopes=_SCOPES
        )

    raise ValueError(
        "No Google service account configured. "
        "Set GOOGLE_SERVICE_ACCOUNT_JSON or GOOGLE_SERVICE_ACCOUNT_PATH."
    )


def _upload_sync(pdf_bytes: bytes, file_name: str, folder_id: str | None) -> dict:
    from googleapiclient.discovery import build
    from googleapiclient.http import MediaInMemoryUpload

    creds = _get_credentials()
    service = build("drive", "v3", credentials=creds)

    metadata: dict = {"name": file_name, "mimeType": "application/pdf"}
    if folder_id:
        metadata["parents"] = [folder_id]

    media = MediaInMemoryUpload(pdf_bytes, mimetype="application/pdf", resumable=False)
    file = service.files().create(
        body=metadata,
        media_body=media,
        fields="id,size,webViewLink",
    ).execute()

    service.permissions().create(
        fileId=file["id"],
        body={"type": "anyone", "role": "reader"},
    ).execute()

    file_id = file["id"]
    return {
        "file_id": file_id,
        "url": f"https://drive.google.com/file/d/{file_id}/view",
        "size": int(file.get("size", 0)),
    }


async def upload_pdf(pdf_bytes: bytes, file_name: str, folder_id: str | None = None) -> dict:
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _upload_sync, pdf_bytes, file_name, folder_id)
