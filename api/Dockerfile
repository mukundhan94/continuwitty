FROM ghcr.io/astral-sh/uv:python3.13-bookworm-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    UV_LINK_MODE=copy

WORKDIR /srv/api

COPY api/pyproject.toml api/uv.lock ./
RUN uv sync --frozen

COPY api/app ./app
COPY api/evals ./evals
COPY db/init /srv/db/init

RUN mkdir -p /srv/data

EXPOSE 8000

CMD ["uv", "run", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
