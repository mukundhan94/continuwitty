from functools import lru_cache

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
}

_DEV_ENV_NAMES = {"dev", "development", "local"}


class Settings(BaseSettings):
    app_env: str = "development"
    log_config_in_dev: bool = True
    database_url: str = "postgresql://engram:engram@localhost:5432/engram_vault"
    embedding_dim: int = 256
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
    openai_api_key: str | None = None
    openai_base_url: str = "https://api.openai.com"
    anthropic_api_key: str | None = None
    anthropic_base_url: str = "https://api.anthropic.com"
    anthropic_version: str = "2023-06-01"
    aws_region: str = "us-east-1"
    aws_access_key_id: str | None = None
    aws_secret_access_key: str | None = None
    aws_session_token: str | None = None

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
