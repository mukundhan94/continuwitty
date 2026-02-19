from app.config import Settings, build_debug_settings_snapshot, should_log_settings


def test_build_debug_settings_snapshot_redacts_secrets() -> None:
    settings = Settings(
        app_env="development",
        log_config_in_dev=True,
        database_url="postgresql://engram:secret@localhost:5432/engram_vault",
        app_session_secret="session-secret",
        ui_demo_password="admin123",
        ui_demo_password_hash="hash-value",
        openai_api_key="openai-key",
        anthropic_api_key="anthropic-key",
        aws_access_key_id="aws-access",
        aws_secret_access_key="aws-secret",
        aws_session_token="aws-session",
        oauth_client_secret_pepper="oauth-secret-pepper",
        default_chat_provider="openai",
    )

    snapshot = build_debug_settings_snapshot(settings)

    assert snapshot["database_url"] == "<redacted>"
    assert snapshot["app_session_secret"] == "<redacted>"
    assert snapshot["ui_demo_password"] == "<redacted>"
    assert snapshot["ui_demo_password_hash"] == "<redacted>"
    assert snapshot["openai_api_key"] == "<redacted>"
    assert snapshot["anthropic_api_key"] == "<redacted>"
    assert snapshot["aws_access_key_id"] == "<redacted>"
    assert snapshot["aws_secret_access_key"] == "<redacted>"
    assert snapshot["aws_session_token"] == "<redacted>"
    assert snapshot["oauth_client_secret_pepper"] == "<redacted>"
    assert snapshot["default_chat_provider"] == "openai"


def test_should_log_settings_only_for_dev_modes() -> None:
    dev_settings = Settings(app_env="development", log_config_in_dev=True)
    local_settings = Settings(app_env="local", log_config_in_dev=True)
    prod_settings = Settings(app_env="production", log_config_in_dev=True)
    disabled_settings = Settings(app_env="development", log_config_in_dev=False)

    assert should_log_settings(dev_settings) is True
    assert should_log_settings(local_settings) is True
    assert should_log_settings(prod_settings) is False
    assert should_log_settings(disabled_settings) is False
