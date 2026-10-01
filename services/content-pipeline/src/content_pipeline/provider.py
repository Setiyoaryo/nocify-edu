"""AI provider abstraction (M6).

One interface for embeddings and chat completions, implemented by:

* :class:`OpenAICompatibleProvider` -- any endpoint that speaks the OpenAI REST
  shape (OpenAI, OpenRouter, a local vLLM/Ollama server). Configured by env.
* :class:`MockProvider` -- deterministic, offline, no API key. Mirrors the Go
  side's ``PAYMENT_PROVIDER=mock`` so tests and dev runs never need a key.

The HTTP client is injectable (``client=``) so tests pass an
``httpx.MockTransport`` and never touch the network, exactly like ``Fetcher``.
"""

from __future__ import annotations

import hashlib
import json
import math
from collections.abc import Sequence
from dataclasses import dataclass
from typing import Any, Protocol, runtime_checkable

import httpx

from content_pipeline.config import Settings


class ProviderError(RuntimeError):
    """Raised when a provider call fails or returns an unusable payload."""


@dataclass(frozen=True)
class ChatResult:
    text: str
    model: str
    prompt_tokens: int
    completion_tokens: int
    total_tokens: int


@runtime_checkable
class Provider(Protocol):
    @property
    def name(self) -> str: ...

    @property
    def embedding_model(self) -> str: ...

    @property
    def chat_model(self) -> str: ...

    def embed(self, texts: Sequence[str]) -> list[list[float]]: ...

    def complete(self, *, system: str, user: str, max_tokens: int) -> ChatResult: ...

    def close(self) -> None: ...


def _as_vector(value: object, *, dim_hint: int = 0) -> list[float]:
    if not isinstance(value, list) or not value:
        raise ProviderError("embedding vector is not a non-empty list")
    vector: list[float] = []
    for item in value:
        if isinstance(item, bool) or not isinstance(item, (int, float)):
            raise ProviderError("embedding vector contains a non-numeric value")
        vector.append(float(item))
    if dim_hint and len(vector) != dim_hint:
        raise ProviderError(f"embedding dimension {len(vector)} != expected {dim_hint}")
    return vector


class OpenAICompatibleProvider:
    """OpenAI-compatible embeddings + chat completions over httpx."""

    def __init__(
        self,
        *,
        base_url: str,
        api_key: str,
        embedding_model: str,
        chat_model: str,
        timeout_seconds: float = 30.0,
        client: httpx.Client | None = None,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._api_key = api_key
        self._embedding_model = embedding_model
        self._chat_model = chat_model
        self._owns_client = client is None
        self._client = client or httpx.Client(timeout=timeout_seconds)

    @property
    def name(self) -> str:
        return "openai"

    @property
    def embedding_model(self) -> str:
        return self._embedding_model

    @property
    def chat_model(self) -> str:
        return self._chat_model

    def close(self) -> None:
        if self._owns_client:
            self._client.close()

    def __enter__(self) -> OpenAICompatibleProvider:
        return self

    def __exit__(self, *_: object) -> None:
        self.close()

    def _post(self, path: str, body: dict[str, object]) -> dict[str, Any]:
        url = f"{self._base_url}{path}"
        try:
            response = self._client.post(
                url, json=body, headers={"Authorization": f"Bearer {self._api_key}"}
            )
        except httpx.HTTPError as exc:  # network/DNS/timeout
            raise ProviderError(f"{path}: {exc}") from exc
        if response.status_code >= 400:
            raise ProviderError(f"{path}: HTTP {response.status_code}: {response.text[:200]}")
        raw: object = response.json()
        if not isinstance(raw, dict):
            raise ProviderError(f"{path}: response is not a JSON object")
        return raw

    def embed(self, texts: Sequence[str]) -> list[list[float]]:
        if not texts:
            return []
        payload = self._post("/embeddings", {"model": self._embedding_model, "input": list(texts)})
        data = payload.get("data")
        if not isinstance(data, list) or len(data) != len(texts):
            raise ProviderError("embeddings: unexpected 'data' payload")
        ordered: list[tuple[int, list[float]]] = []
        for position, entry in enumerate(data):
            if not isinstance(entry, dict):
                raise ProviderError("embeddings: entry is not an object")
            index = entry.get("index", position)
            if not isinstance(index, int):
                index = position
            ordered.append((index, _as_vector(entry.get("embedding"))))
        ordered.sort(key=lambda pair: pair[0])
        return [vector for _, vector in ordered]

    def complete(self, *, system: str, user: str, max_tokens: int) -> ChatResult:
        payload = self._post(
            "/chat/completions",
            {
                "model": self._chat_model,
                "messages": [
                    {"role": "system", "content": system},
                    {"role": "user", "content": user},
                ],
                "max_tokens": max_tokens,
                "temperature": 0,
                "response_format": {"type": "json_object"},
            },
        )
        choices = payload.get("choices")
        if not isinstance(choices, list) or not choices:
            raise ProviderError("chat: no choices returned")
        first = choices[0]
        if not isinstance(first, dict):
            raise ProviderError("chat: choice is not an object")
        message = first.get("message")
        if not isinstance(message, dict):
            raise ProviderError("chat: message missing")
        content = message.get("content")
        if not isinstance(content, str) or not content.strip():
            raise ProviderError("chat: empty content")
        usage = payload.get("usage")
        prompt_tokens = response_tokens = 0
        if isinstance(usage, dict):
            prompt_tokens = _as_int(usage.get("prompt_tokens"))
            response_tokens = _as_int(usage.get("completion_tokens"))
        total = _as_int(usage.get("total_tokens")) if isinstance(usage, dict) else 0
        return ChatResult(
            text=content,
            model=self._chat_model,
            prompt_tokens=prompt_tokens,
            completion_tokens=response_tokens,
            total_tokens=total or (prompt_tokens + response_tokens),
        )


def _as_int(value: object) -> int:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        return 0
    return int(value)


class MockProvider:
    """Deterministic, offline provider for tests and dev.

    Embeddings come from a sha256 of the text (identical text -> cosine 1.0,
    different text -> ~0). Completions return a grounded JSON question built from
    the longest word of the prompt, so the assisted pipeline can be exercised
    end-to-end without a key.
    """

    def __init__(self, *, dim: int = 64, embedding_model: str = "mock-embed", chat_model: str = "mock-chat") -> None:
        self._dim = dim
        self._embedding_model = embedding_model
        self._chat_model = chat_model

    @property
    def name(self) -> str:
        return "mock"

    @property
    def embedding_model(self) -> str:
        return self._embedding_model

    @property
    def chat_model(self) -> str:
        return self._chat_model

    def close(self) -> None:
        return None

    def embed(self, texts: Sequence[str]) -> list[list[float]]:
        return [self._vector(text) for text in texts]

    def _vector(self, text: str) -> list[float]:
        # Expand a sha256 digest into `dim` floats in [-1, 1], then L2-normalise.
        seed = hashlib.sha256(text.encode("utf-8")).digest()
        raw = [((seed[i % len(seed)] / 255.0) * 2.0) - 1.0 for i in range(self._dim)]
        norm = math.sqrt(sum(value * value for value in raw)) or 1.0
        return [value / norm for value in raw]

    def complete(self, *, system: str, user: str, max_tokens: int) -> ChatResult:
        words = [w.strip(".,;:()[]\"'").lower() for w in user.split()]
        content_words = [w for w in words if len(w) >= 5 and w.isalpha()]
        term = sorted(content_words)[0] if content_words else "materi"
        body = {
            "type": "short_answer",
            "difficulty": 3,
            "learning_objective": f"Peserta didik dapat menjelaskan {term}.",
            "content": {"prompt": f"Menurut sumber, apa yang dimaksud dengan {term}?"},
            "answer": {"accepted": [term]},
            "explanation": f"{term} dijelaskan pada bagian sumber yang dirujuk.",
        }
        text = json.dumps(body, ensure_ascii=False)
        return ChatResult(
            text=text,
            model=self._chat_model,
            prompt_tokens=max(1, len(user) // 4),
            completion_tokens=max(1, len(text) // 4),
            total_tokens=max(1, (len(user) + len(text)) // 4),
        )


def build_provider(settings: Settings, *, client: httpx.Client | None = None) -> Provider | None:
    """Build the configured provider, or None when AI is disabled."""
    provider = settings.ai_provider
    if not provider:
        return None
    if provider == "mock":
        return MockProvider()
    if provider == "openai":
        if not settings.ai_base_url or not settings.ai_api_key:
            raise ValueError("AI_PROVIDER=openai requires AI_BASE_URL and AI_API_KEY")
        return OpenAICompatibleProvider(
            base_url=settings.ai_base_url,
            api_key=settings.ai_api_key,
            embedding_model=settings.ai_embedding_model,
            chat_model=settings.ai_chat_model,
            timeout_seconds=settings.ai_timeout_seconds,
            client=client,
        )
    raise ValueError(f"unknown AI_PROVIDER {provider!r} (expected 'mock' or 'openai')")
