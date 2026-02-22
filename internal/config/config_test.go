package config

import (
	"strings"
	"testing"
)

func baseSettings() Settings {
	return Settings{
		AppEnv:                            "development",
		LogConfigInDev:                    true,
		DatabaseURL:                       "postgresql://engram:engram@localhost:5432/engram_vault",
		AppSessionSecret:                  defaultSessionSecret,
		UIDemoUsername:                    "admin",
		UIDemoPassword:                    defaultUIDemoPassword,
		MCPTokenPepper:                    defaultMCPTokenPepper,
		OAuthClientSecretPepper:           defaultOAuthClientSecret,
		OAuthRequireProtectedRegistration: true,
		DefaultChatProvider:               "openai",
	}
}

func TestBuildDebugSettingsSnapshotRedactsSecrets(t *testing.T) {
	settings := baseSettings()
	settings.DatabaseURL = "postgresql://engram:secret@localhost:5432/engram_vault"
	settings.AppSessionSecret = "session-secret"
	settings.UIDemoPassword = "admin123"
	settings.UIDemoPasswordHash = "hash-value"
	settings.OpenAIAPIKey = "openai-key"
	settings.AnthropicAPIKey = "anthropic-key"
	settings.AWSAccessKeyID = "aws-access"
	settings.AWSSecretAccessKey = "aws-secret"
	settings.AWSSessionToken = "aws-session"
	settings.OAuthClientSecretPepper = "oauth-secret-pepper"
	settings.DefaultChatProvider = "openai"

	snapshot := BuildDebugSettingsSnapshot(settings)

	assertEqualString(t, snapshot["database_url"], "<redacted>")
	assertEqualString(t, snapshot["app_session_secret"], "<redacted>")
	assertEqualString(t, snapshot["ui_demo_password"], "<redacted>")
	assertEqualString(t, snapshot["ui_demo_password_hash"], "<redacted>")
	assertEqualString(t, snapshot["openai_api_key"], "<redacted>")
	assertEqualString(t, snapshot["anthropic_api_key"], "<redacted>")
	assertEqualString(t, snapshot["aws_access_key_id"], "<redacted>")
	assertEqualString(t, snapshot["aws_secret_access_key"], "<redacted>")
	assertEqualString(t, snapshot["aws_session_token"], "<redacted>")
	assertEqualString(t, snapshot["oauth_client_secret_pepper"], "<redacted>")
	assertEqualString(t, snapshot["default_chat_provider"], "openai")
}

func TestShouldLogSettingsOnlyForDevModes(t *testing.T) {
	devSettings := baseSettings()
	devSettings.AppEnv = "development"
	devSettings.LogConfigInDev = true

	localSettings := baseSettings()
	localSettings.AppEnv = "local"
	localSettings.LogConfigInDev = true

	prodSettings := baseSettings()
	prodSettings.AppEnv = "production"
	prodSettings.LogConfigInDev = true
	prodSettings.AppSessionSecret = strings.Repeat("p", 48)
	prodSettings.UIDemoPassword = "P@ssword-for-production-123"
	prodSettings.MCPTokenPepper = strings.Repeat("m", 48)
	prodSettings.OAuthClientSecretPepper = strings.Repeat("o", 48)

	disabledSettings := baseSettings()
	disabledSettings.AppEnv = "development"
	disabledSettings.LogConfigInDev = false

	if !ShouldLogSettings(devSettings) {
		t.Fatalf("development settings should be loggable")
	}
	if !ShouldLogSettings(localSettings) {
		t.Fatalf("local settings should be loggable")
	}
	if ShouldLogSettings(prodSettings) {
		t.Fatalf("production settings should not be loggable")
	}
	if ShouldLogSettings(disabledSettings) {
		t.Fatalf("log disabled settings should not be loggable")
	}
}

func TestProductionSettingsRejectInsecureDefaults(t *testing.T) {
	settings := baseSettings()
	settings.AppEnv = "production"

	err := ValidateProductionSecurity(settings)
	if err == nil {
		t.Fatalf("expected production settings validation to fail")
	}
	message := err.Error()
	if !strings.Contains(message, "APP_SESSION_SECRET cannot use the development default in production") {
		t.Fatalf("expected APP_SESSION_SECRET violation, got %q", message)
	}
	if !strings.Contains(message, "UI_DEMO_PASSWORD uses a weak value in production") {
		t.Fatalf("expected weak password violation, got %q", message)
	}
}

func TestProductionSettingsAcceptHardenedValues(t *testing.T) {
	settings := baseSettings()
	settings.AppEnv = "production"
	settings.AppSessionSecret = strings.Repeat("s", 48)
	settings.UIDemoPassword = "Strong-production-passphrase-123"
	settings.MCPTokenPepper = strings.Repeat("m", 48)
	settings.OAuthClientSecretPepper = strings.Repeat("o", 48)
	settings.OAuthRequireProtectedRegistration = true

	err := ValidateProductionSecurity(settings)
	if err != nil {
		t.Fatalf("expected hardened production settings to pass validation: %v", err)
	}
}

func assertEqualString(t *testing.T, actual any, expected string) {
	t.Helper()
	value, ok := actual.(string)
	if !ok {
		t.Fatalf("expected %q to be string, got %T", expected, actual)
	}
	if value != expected {
		t.Fatalf("expected %q, got %q", expected, value)
	}
}
