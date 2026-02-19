import subprocess
from functools import lru_cache
from importlib import metadata
from pathlib import Path

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict

_SENSITIVE_SETTING_KEYS = {
    "database_url",
    "app_session_secret",
    "ui_demo_password",
    "ui_demo_password_hash",
    "openai_api_key",
    "anthropic_api_key",
    "aws_access_key_id",
    "aws_secret_access_key",
    "aws_session_token",
    "langfuse_public_key",
    "langfuse_secret_key",
    "mcp_token_pepper",
    "oauth_client_secret_pepper",
}

_DEV_ENV_NAMES = {"dev", "development", "local"}


def _default_semantic_version() -> str:
    try:
        return metadata.version("engram-vault-api")
    except metadata.PackageNotFoundError:
        return "0.1.0"


def _default_commit_sha() -> str:
    # Best-effort only: local/dev usually has .git, containers may not.
    try:
        repo_root = Path(__file__).resolve().parents[2]
        result = subprocess.run(
            ["git", "rev-parse", "--short", "HEAD"],
            cwd=repo_root,
            check=True,
            capture_output=True,
            text=True,
        )
        commit = result.stdout.strip()
        if commit:
            return commit
    except Exception:
        pass
    return "unknown"


class Settings(BaseSettings):
    app_env: str = "development"
    log_config_in_dev: bool = True
    app_semantic_version: str = Field(default_factory=_default_semantic_version)
    app_commit_sha: str = Field(default_factory=_default_commit_sha)
    database_url: str = "postgresql://engram:engram@localhost:5432/engram_vault"
    embedding_dim: int = 256
    embedding_provider: str = "local"
    embedding_model: str = "text-embedding-3-small"
    embedding_fallback_to_local: bool = True
    embedding_timeout_seconds: float = 20.0
    ingestion_max_file_bytes: int = 2_000_000
    ingestion_max_text_chars: int = 200_000
    api_host: str = "0.0.0.0"
    api_port: int = 8000
    app_session_secret: str = "engram-local-dev-session-secret"
    ui_demo_username: str = "admin"
    ui_demo_password: str = "admin123"
    ui_demo_password_hash: str | None = None
    langgraph_checkpoint_path: str = "./data/langgraph_checkpoints.sqlite"
    audit_log_path: str = "./data/audit_events.jsonl"
    login_rate_limit_window_seconds: int = 300
    login_rate_limit_max_attempts: int = 5
    login_lockout_seconds: int = 900
    default_chat_provider: str = "openai"
    default_chat_model: str = "gpt-4o-mini"
    chat_debug_enabled: bool = True
    chat_debug_log_console: bool = True
    chat_debug_include_raw_text: bool = True
    langfuse_enabled: bool = False
    langfuse_host: str = "https://cloud.langfuse.com"
    langfuse_public_key: str | None = None
    langfuse_secret_key: str | None = None
    openai_api_key: str | None = None
    openai_base_url: str = "https://api.openai.com"
    anthropic_api_key: str | None = None
    anthropic_base_url: str = "https://api.anthropic.com"
    anthropic_version: str = "2023-06-01"
    aws_region: str = "us-east-1"
    aws_access_key_id: str | None = None
    aws_secret_access_key: str | None = None
    aws_session_token: str | None = None
    mcp_token_pepper: str = "engram-local-dev-mcp-token-pepper"
    oauth_enabled: bool = True
    oauth_issuer_url: str | None = None
    oauth_access_token_ttl_seconds: int = 3600
    oauth_authorization_code_ttl_seconds: int = 300
    oauth_client_secret_pepper: str = "engram-local-dev-oauth-client-pepper"

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )


@lru_cache
def get_settings() -> Settings:
    return Settings()


def build_debug_settings_snapshot(settings: Settings) -> dict[str, object]:
    snapshot = settings.model_dump(mode="python")
    for key in _SENSITIVE_SETTING_KEYS:
        if key in snapshot and snapshot[key]:
            snapshot[key] = "<redacted>"
    return snapshot


def should_log_settings(settings: Settings) -> bool:
    return settings.log_config_in_dev and settings.app_env.strip().lower() in _DEV_ENV_NAMES
