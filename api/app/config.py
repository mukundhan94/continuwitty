from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    database_url: str = "postgresql://engram:engram@localhost:5432/engram_vault"
    embedding_dim: int = 256
    api_host: str = "0.0.0.0"
    api_port: int = 8000
    app_session_secret: str = "engram-local-dev-session-secret"
    ui_demo_username: str = "admin"
    ui_demo_password: str = "admin123"
    ui_demo_password_hash: str | None = None
    langgraph_checkpoint_path: str = "./data/langgraph_checkpoints.sqlite"

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )


@lru_cache
def get_settings() -> Settings:
    return Settings()
