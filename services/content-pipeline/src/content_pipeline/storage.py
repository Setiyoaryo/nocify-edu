"""Raw document archive in S3-compatible object storage (§11).

Raw bytes are kept so extraction can be re-run without re-fetching (§51). The
key layout mirrors the planning document:

    raw/<source-code>/<YYYY>/<MM>/<sha256>.<ext>

Storage is optional: when no bucket is configured, archiving is a no-op and the
crawl item's ``storage_key`` stays empty.
"""

from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path

import boto3

from content_pipeline.config import Settings

_EXTENSIONS = {
    "text/html": ".html",
    "text/plain": ".txt",
    "text/markdown": ".md",
    "application/pdf": ".pdf",
    "application/xml": ".xml",
    "application/json": ".json",
}


def extension_for(content_type: str) -> str:
    base = content_type.split(";", 1)[0].strip().lower()
    for known, ext in _EXTENSIONS.items():
        if base.startswith(known):
            return ext
    return ".bin"


def normalise_endpoint(endpoint: str, use_ssl: bool) -> str | None:
    """boto3 needs a URL scheme; the compose env passes ``s3:8333`` like the Go API."""
    if not endpoint:
        return None
    if "://" in endpoint:
        return endpoint
    return f"{'https' if use_ssl else 'http'}://{endpoint}"


def raw_key(source_code: str, content_hash: str, content_type: str, when: datetime | None = None) -> str:
    when = when or datetime.now(UTC)
    ext = extension_for(content_type)
    return f"raw/{source_code}/{when:%Y}/{when:%m}/{content_hash}{ext}"


class RawArchive:
    def __init__(self, settings: Settings) -> None:
        self._settings = settings
        self._client: object | None = None

    @property
    def enabled(self) -> bool:
        return self._settings.raw_storage_enabled

    def _s3(self) -> object:
        if self._client is None:
            self._client = boto3.client(
                "s3",
                endpoint_url=normalise_endpoint(self._settings.s3_endpoint, self._settings.s3_use_ssl),
                region_name=self._settings.s3_region,
                use_ssl=self._settings.s3_use_ssl,
                aws_access_key_id=self._settings.storage_access_key or None,
                aws_secret_access_key=self._settings.storage_secret_key or None,
            )
        return self._client

    def store(self, source_code: str, body: bytes, content_hash: str, content_type: str) -> str:
        """Archive *body* and return its storage key ('' when disabled)."""
        if not self.enabled:
            return ""
        key = raw_key(source_code, content_hash, content_type)
        self._s3().put_object(  # type: ignore[attr-defined]
            Bucket=self._settings.s3_bucket,
            Key=key,
            Body=body,
            ContentType=content_type.split(";", 1)[0].strip() or "application/octet-stream",
        )
        return key

    def load(self, storage_key: str) -> bytes:
        """Read an archived document back, so extraction can be re-run (§51)."""
        response = self._s3().get_object(  # type: ignore[attr-defined]
            Bucket=self._settings.s3_bucket, Key=storage_key
        )
        return bytes(response["Body"].read())


def local_filename(key: str) -> str:
    """Basename of a storage key, for tests and debugging."""
    return Path(key).name
