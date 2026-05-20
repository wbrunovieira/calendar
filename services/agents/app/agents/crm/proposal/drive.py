"""Google Drive upload using OAuth2 refresh token."""
from __future__ import annotations

import asyncio
import io
import logging

logger = logging.getLogger("agents")

_SCOPES = ["https://www.googleapis.com/auth/drive.file"]


def _get_credentials():
    from google.auth.transport.requests import Request
    from google.oauth2.credentials import Credentials
    from app.config import settings

    if not settings.google_refresh_token:
        raise ValueError(
            "No Google refresh token configured. Set GOOGLE_REFRESH_TOKEN."
        )

    creds = Credentials(
        token=None,
        refresh_token=settings.google_refresh_token,
        client_id=settings.google_client_id,
        client_secret=settings.google_client_secret,
        token_uri="https://oauth2.googleapis.com/token",
        scopes=_SCOPES,
    )
    creds.refresh(Request())
    return creds


def _find_or_create_folder(service, folder_name: str) -> str:
    result = service.files().list(
        q=f"mimeType='application/vnd.google-apps.folder' and name='{folder_name}' and trashed=false",
        fields="files(id)",
        pageSize=1,
    ).execute()
    files = result.get("files", [])
    if files:
        return files[0]["id"]

    folder = service.files().create(
        body={"name": folder_name, "mimeType": "application/vnd.google-apps.folder"},
        fields="id",
    ).execute()
    logger.info("[Drive] created folder '%s' id=%s", folder_name, folder["id"])
    return folder["id"]


def _upload_sync(pdf_bytes: bytes, file_name: str, folder_name: str) -> dict:
    from googleapiclient.discovery import build
    from googleapiclient.http import MediaIoBaseUpload

    creds = _get_credentials()
    service = build("drive", "v3", credentials=creds)

    folder_id = _find_or_create_folder(service, folder_name)

    media = MediaIoBaseUpload(
        io.BytesIO(pdf_bytes),
        mimetype="application/pdf",
        resumable=False,
    )
    file = service.files().create(
        body={"name": file_name, "parents": [folder_id]},
        media_body=media,
        fields="id,webViewLink,size",
    ).execute()

    service.permissions().create(
        fileId=file["id"],
        body={"role": "reader", "type": "anyone"},
    ).execute()

    return {
        "file_id": file["id"],
        "url": file["webViewLink"],
        "size": int(file.get("size") or 0),
    }


async def upload_pdf(pdf_bytes: bytes, file_name: str, folder_name: str | None = None) -> dict:
    """Upload PDF to Google Drive folder. Returns {file_id, url, size}."""
    from app.config import settings
    folder = folder_name or settings.google_drive_folder_name or "Propostas CRM"
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _upload_sync, pdf_bytes, file_name, folder)
