# Content pipeline (M1). Build context: repository root.
# Runs as a one-shot job, not a server: `docker compose --profile pipeline run --rm pipeline crawl`
FROM python:3.12-slim

ENV PYTHONUNBUFFERED=1 \
    UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy \
    UV_PROJECT_ENVIRONMENT=/app/.venv

RUN pip install --no-cache-dir uv

WORKDIR /app
COPY services/content-pipeline/pyproject.toml services/content-pipeline/uv.lock ./
RUN uv sync --frozen --no-dev --no-install-project

COPY services/content-pipeline/ ./
RUN uv sync --frozen --no-dev

ENV PATH="/app/.venv/bin:$PATH"
RUN useradd -u 10001 -m pipeline
USER pipeline

ENTRYPOINT ["content-pipeline"]
CMD ["--help"]
