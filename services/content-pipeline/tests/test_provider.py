from __future__ import annotations

import dataclasses
import json
from collections.abc import Callable
from typing import Any

import httpx
import pytest

from content_pipeline.config import Settings
from content_pipeline.provider import (
    MockProvider,
    OpenAICompatibleProvider,
    ProviderError,
    build_provider,
)
from content_pipeline.similarity import cosine


def _settings(**overrides: Any) -> Settings:
    base = Settings(
        database_url="postgres://x",
        user_agent="ua",
        http_timeout_seconds=20.0,
        max_response_bytes=1,
        max_redirects=5,
        default_rate_limit_ms=1000,
        s3_endpoint="",
        s3_region="us-east-1",
        s3_bucket="",
        s3_use_ssl=True,
        storage_access_key="",
        storage_secret_key="",
        ai_provider="",
        ai_base_url="",
        ai_api_key="",
        ai_embedding_model="",
        ai_chat_model="",
        ai_timeout_seconds=30.0,
        ai_max_output_tokens=700,
        ai_generation_cap=50,
        ai_embedding_batch_size=64,
        ai_prompt_version="scenario-v1",
        worker_poll_seconds=5.0,
        scheduler_seconds=60.0,
    )
    return dataclasses.replace(base, **overrides)


def _provider(handler: Callable[[httpx.Request], httpx.Response]) -> OpenAICompatibleProvider:
    client = httpx.Client(transport=httpx.MockTransport(handler))
    return OpenAICompatibleProvider(
        base_url="https://ai.test/v1",
        api_key="k",
        embedding_model="emb",
        chat_model="chat",
        client=client,
    )


def test_embed_posts_to_embeddings_and_orders_by_index() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/embeddings"
        assert request.headers["authorization"] == "Bearer k"
        body = json.loads(request.content)
        assert body["model"] == "emb"
        assert body["input"] == ["a", "b"]
        return httpx.Response(
            200,
            json={
                "data": [
                    {"index": 1, "embedding": [0.0, 1.0]},
                    {"index": 0, "embedding": [1.0, 0.0]},
                ]
            },
        )

    assert _provider(handler).embed(["a", "b"]) == [[1.0, 0.0], [0.0, 1.0]]


def test_embed_returns_empty_without_calling_the_api() -> None:
    def handler(request: httpx.Request) -> httpx.Response:  # pragma: no cover - must not run
        raise AssertionError("no call expected")

    assert _provider(handler).embed([]) == []


def test_complete_posts_to_chat_and_reads_tokens() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/chat/completions"
        body = json.loads(request.content)
        assert body["model"] == "chat"
        assert body["temperature"] == 0
        assert body["messages"][0]["role"] == "system"
        assert body["messages"][1]["content"] == "user text"
        return httpx.Response(
            200,
            json={
                "choices": [{"message": {"content": '{"ok": true}'}}],
                "usage": {"prompt_tokens": 3, "completion_tokens": 4, "total_tokens": 7},
            },
        )

    result = _provider(handler).complete(system="sys", user="user text", max_tokens=10)
    assert result.text == '{"ok": true}'
    assert (result.prompt_tokens, result.completion_tokens, result.total_tokens) == (3, 4, 7)


def test_non_2xx_raises_provider_error() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(500, text="boom")

    with pytest.raises(ProviderError):
        _provider(handler).embed(["a"])


def test_malformed_embedding_payload_raises_provider_error() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"data": [{"embedding": "nope"}]})

    with pytest.raises(ProviderError):
        _provider(handler).embed(["a"])


def test_mock_provider_is_deterministic_and_normalised() -> None:
    mock = MockProvider()
    first = mock.embed(["hello"])[0]
    assert first == mock.embed(["hello"])[0]
    assert cosine(first, mock.embed(["hello"])[0]) == pytest.approx(1.0)
    assert first != mock.embed(["other"])[0]
    assert sum(value * value for value in first) == pytest.approx(1.0)


def test_mock_completion_is_grounded_in_the_prompt() -> None:
    user = "Penjumlahan bilangan sampai dua puluh"
    result = MockProvider().complete(system="sys", user=user, max_tokens=10)
    body = json.loads(result.text)
    answer = body["answer"]["accepted"][0]
    assert answer in user.lower()
    assert body["content"]["prompt"]


def test_build_provider_disabled_when_unset() -> None:
    assert build_provider(_settings()) is None


def test_build_provider_returns_mock() -> None:
    provider = build_provider(_settings(ai_provider="mock"))
    assert isinstance(provider, MockProvider)
    assert provider.embedding_model == "mock-embed"


def test_build_provider_openai_requires_base_url_and_key() -> None:
    with pytest.raises(ValueError):
        build_provider(_settings(ai_provider="openai"))


def test_build_provider_rejects_unknown_provider() -> None:
    with pytest.raises(ValueError):
        build_provider(_settings(ai_provider="wat"))
