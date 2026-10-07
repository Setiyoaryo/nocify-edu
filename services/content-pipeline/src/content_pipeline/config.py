"""Environment-driven configuration for the content pipeline."""

from __future__ import annotations

import os
from dataclasses import dataclass

DEFAULT_USER_AGENT = "NocifyEduPlayBot/0.1 (+https://github.com/Setiyoaryo/nocify-edu)"


def _env(name: str, default: str = "") -> str:
    return os.environ.get(name, default).strip()


def _env_int(name: str, default: int) -> int:
    raw = _env(name)
    if not raw:
        return default
    try:
        return int(raw)
    except ValueError as exc:
        raise ValueError(f"{name} must be an integer, got {raw!r}") from exc


def _env_float(name: str, default: float) -> float:
    raw = _env(name)
    if not raw:
        return default
    try:
        return float(raw)
    except ValueError as exc:
        raise ValueError(f"{name} must be a number, got {raw!r}") from exc


def _env_bool(name: str, default: bool) -> bool:
    raw = _env(name).lower()
    if not raw:
        return default
    return raw in {"1", "true", "yes", "on"}


@dataclass(frozen=True)
class Settings:
    database_url: str
    user_agent: str
    http_timeout_seconds: float
    max_response_bytes: int
    max_redirects: int
    default_rate_limit_ms: int
    s3_endpoint: str
    s3_region: str
    s3_bucket: str
    s3_use_ssl: bool
    storage_access_key: str
    storage_secret_key: str
    # AI provider (M6). Empty ai_provider disables every AI-backed command.
    ai_provider: str
    ai_base_url: str
    ai_api_key: str
    ai_embedding_model: str
    ai_chat_model: str
    ai_timeout_seconds: float
    ai_max_output_tokens: int
    ai_generation_cap: int
    ai_embedding_batch_size: int
    ai_prompt_version: str
    # Worker daemon (M11). The poll interval is how often an idle worker looks
    # for a queued job; the scheduler interval is how often it checks sources
    # whose crawl cadence is due.
    worker_poll_seconds: float
    scheduler_seconds: float

    @classmethod
    def from_env(cls) -> Settings:
        database_url = _env("CONTENT_PIPELINE_DATABASE_URL") or _env("DATABASE_URL")
        if not database_url:
            raise ValueError("DATABASE_URL is required")
        return cls(
            database_url=database_url,
            user_agent=_env("PIPELINE_USER_AGENT", DEFAULT_USER_AGENT),
            http_timeout_seconds=_env_float("HTTP_TIMEOUT_SECONDS", 20.0),
            max_response_bytes=_env_int("MAX_RESPONSE_BYTES", 10 * 1024 * 1024),
            max_redirects=_env_int("MAX_REDIRECTS", 5),
            default_rate_limit_ms=_env_int("DEFAULT_RATE_LIMIT_MS", 1000),
            s3_endpoint=_env("S3_ENDPOINT"),
            s3_region=_env("S3_REGION", "us-east-1"),
            s3_bucket=_env("S3_BUCKET"),
            s3_use_ssl=_env_bool("S3_USE_SSL", True),
            storage_access_key=_env("STORAGE_ACCESS_KEY"),
            storage_secret_key=_env("STORAGE_SECRET_KEY"),
            ai_provider=_env("AI_PROVIDER"),
            ai_base_url=_env("AI_BASE_URL"),
            ai_api_key=_env("AI_API_KEY"),
            ai_embedding_model=_env("AI_EMBEDDING_MODEL"),
            ai_chat_model=_env("AI_CHAT_MODEL"),
            ai_timeout_seconds=_env_float("AI_TIMEOUT_SECONDS", 30.0),
            ai_max_output_tokens=_env_int("AI_MAX_OUTPUT_TOKENS", 700),
            ai_generation_cap=_env_int("AI_GENERATION_CAP", 50),
            ai_embedding_batch_size=_env_int("AI_EMBEDDING_BATCH_SIZE", 64),
            ai_prompt_version=_env("AI_PROMPT_VERSION", "scenario-v1"),
            worker_poll_seconds=_env_float("PIPELINE_WORKER_POLL_SECONDS", 5.0),
            scheduler_seconds=_env_float("PIPELINE_SCHEDULER_SECONDS", 60.0),
        )

    @property
    def raw_storage_enabled(self) -> bool:
        """Raw archive is optional so the crawler can run without object storage."""
        return bool(self.s3_bucket)

    @property
    def ai_enabled(self) -> bool:
        """True when any provider is selected (mock or openai)."""
        return bool(self.ai_provider)

    @property
    def ai_configured(self) -> bool:
        """True when the selected provider can actually be reached."""
        if self.ai_provider == "mock":
            return True
        return self.ai_provider == "openai" and bool(self.ai_base_url) and bool(self.ai_api_key)
