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
    prod_settings = Settings(
        app_env="production",
        log_config_in_dev=True,
        app_session_secret="p" * 48,
        ui_demo_password="P@ssword-for-production-123",
        mcp_token_pepper="m" * 48,
        oauth_client_secret_pepper="o" * 48,
    )
    disabled_settings = Settings(app_env="development", log_config_in_dev=False)

    assert should_log_settings(dev_settings) is True
    assert should_log_settings(local_settings) is True
    assert should_log_settings(prod_settings) is False
    assert should_log_settings(disabled_settings) is False


def test_production_settings_reject_insecure_defaults() -> None:
    try:
        Settings(app_env="production")
    except Exception as exc:  # noqa: BLE001
        message = str(exc)
    else:  # pragma: no cover - defensive
        raise AssertionError("Expected production settings validation to fail")

    assert "APP_SESSION_SECRET cannot use the development default in production" in message
    assert "UI_DEMO_PASSWORD uses a weak value in production" in message


def test_production_settings_accept_hardened_values() -> None:
    settings = Settings(
        app_env="production",
        app_session_secret="s" * 48,
        ui_demo_password="Strong-production-passphrase-123",
        mcp_token_pepper="m" * 48,
        oauth_client_secret_pepper="o" * 48,
        oauth_require_protected_registration=True,
    )

    assert settings.app_env == "production"
