package config

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"engram/internal/governance"

	"github.com/kelseyhightower/envconfig"
)

var (
	sensitiveSettingKeys = map[string]struct{}{
		"database_url":               {},
		"app_session_secret":         {},
		"ui_demo_password":           {},
		"ui_demo_password_hash":      {},
		"oidc_client_secret":         {},
		"openai_api_key":             {},
		"anthropic_api_key":          {},
		"aws_access_key_id":          {},
		"aws_secret_access_key":      {},
		"aws_session_token":          {},
		"langfuse_public_key":        {},
		"langfuse_secret_key":        {},
		"mcp_token_pepper":           {},
		"oauth_client_secret_pepper": {},
	}

	devEnvNames = map[string]struct{}{
		"dev":         {},
		"development": {},
		"local":       {},
	}

	productionEnvNames = map[string]struct{}{
		"prod":       {},
		"production": {},
	}

	weakPasswordValues = map[string]struct{}{
		"admin":    {},
		"admin123": {},
		"changeme": {},
		"password": {},
	}
)

const (
	minSecretLength                = 32
	minPasswordLength              = 12
	defaultSessionSecret           = "engram-local-dev-session-secret"
	defaultMCPTokenPepper          = "engram-local-dev-mcp-token-pepper"
	defaultOAuthClientSecret       = "engram-local-dev-oauth-client-pepper"
	defaultUIDemoPassword          = "admin123"
	defaultDatabaseURL             = "postgresql://engram:engram@localhost:5432/engram_vault"
	defaultEmbeddingModel          = "text-embedding-3-small"
	defaultOpenAIBaseURL           = "https://api.openai.com"
	defaultAnthropicBaseURL        = "https://api.anthropic.com"
	defaultAnthropicVersion        = "2023-06-01"
	defaultLangfuseHost            = "https://cloud.langfuse.com"
	defaultAppSemanticVersion      = "0.1.0"
	defaultAppCommitSHA            = "unknown"
	defaultAPIServerHost           = "0.0.0.0"
	defaultDefaultChatProvider     = "openai"
	defaultDefaultChatModel        = "gpt-4o-mini"
	defaultAuditLogPath            = "./data/audit_events.jsonl"
	defaultLanggraphCheckpointSQL  = "./data/langgraph_checkpoints.sqlite"
	defaultChatPromptPolicyVersion = governance.DefaultChatPromptPolicyVersion
	defaultMCPToolPolicyVersion    = governance.DefaultMCPToolPolicyVersion
)

// Settings stores backend runtime configuration.
type Settings struct {
	AppEnv                             string  `envconfig:"APP_ENV" default:"development"`
	LogConfigInDev                     bool    `envconfig:"LOG_CONFIG_IN_DEV" default:"true"`
	AppSemanticVersion                 string  `envconfig:"APP_SEMANTIC_VERSION" default:"0.1.0"`
	AppCommitSHA                       string  `envconfig:"APP_COMMIT_SHA" default:"unknown"`
	DatabaseURL                        string  `envconfig:"DATABASE_URL" default:"postgresql://engram:engram@localhost:5432/engram_vault"`
	EmbeddingDim                       int     `envconfig:"EMBEDDING_DIM" default:"256"`
	EmbeddingProvider                  string  `envconfig:"EMBEDDING_PROVIDER" default:"local"`
	EmbeddingModel                     string  `envconfig:"EMBEDDING_MODEL" default:"text-embedding-3-small"`
	EmbeddingFallbackToLocal           bool    `envconfig:"EMBEDDING_FALLBACK_TO_LOCAL" default:"true"`
	EmbeddingTimeoutSeconds            float64 `envconfig:"EMBEDDING_TIMEOUT_SECONDS" default:"20.0"`
	IngestionMaxFileBytes              int     `envconfig:"INGESTION_MAX_FILE_BYTES" default:"2000000"`
	IngestionMaxTextChars              int     `envconfig:"INGESTION_MAX_TEXT_CHARS" default:"200000"`
	IngestionMaxMetadataJSONBytes      int     `envconfig:"INGESTION_MAX_METADATA_JSON_BYTES" default:"20000"`
	APIHost                            string  `envconfig:"API_HOST" default:"0.0.0.0"`
	APIPort                            int     `envconfig:"API_PORT" default:"8000"`
	APIRequestLogEnabled               bool    `envconfig:"API_REQUEST_LOG_ENABLED" default:"true"`
	APIMetricsEnabled                  bool    `envconfig:"API_METRICS_ENABLED" default:"true"`
	AppSessionSecret                   string  `envconfig:"APP_SESSION_SECRET" default:"engram-local-dev-session-secret"`
	UIDemoUsername                     string  `envconfig:"UI_DEMO_USERNAME" default:"admin"`
	UIDemoPassword                     string  `envconfig:"UI_DEMO_PASSWORD" default:"admin123"`
	UIDemoPasswordHash                 string  `envconfig:"UI_DEMO_PASSWORD_HASH"`
	LanggraphCheckpointPath            string  `envconfig:"LANGGRAPH_CHECKPOINT_PATH" default:"./data/langgraph_checkpoints.sqlite"`
	AuditLogPath                       string  `envconfig:"AUDIT_LOG_PATH" default:"./data/audit_events.jsonl"`
	AuditLogStdoutEnabled              bool    `envconfig:"AUDIT_LOG_STDOUT_ENABLED" default:"false"`
	AuditLogMaxEventBytes              int     `envconfig:"AUDIT_LOG_MAX_EVENT_BYTES" default:"32768"`
	LoginRateLimitWindowSeconds        int     `envconfig:"LOGIN_RATE_LIMIT_WINDOW_SECONDS" default:"300"`
	LoginRateLimitMaxAttempts          int     `envconfig:"LOGIN_RATE_LIMIT_MAX_ATTEMPTS" default:"5"`
	LoginLockoutSeconds                int     `envconfig:"LOGIN_LOCKOUT_SECONDS" default:"900"`
	MCPTransportRateLimitWindowSeconds int     `envconfig:"MCP_TRANSPORT_RATE_LIMIT_WINDOW_SECONDS" default:"60"`
	MCPTransportRateLimitMaxRequests   int     `envconfig:"MCP_TRANSPORT_RATE_LIMIT_MAX_REQUESTS" default:"120"`
	MCPTransportRateLimitBlockSeconds  int     `envconfig:"MCP_TRANSPORT_RATE_LIMIT_BLOCK_SECONDS" default:"30"`
	DefaultChatProvider                string  `envconfig:"DEFAULT_CHAT_PROVIDER" default:"openai"`
	DefaultChatModel                   string  `envconfig:"DEFAULT_CHAT_MODEL" default:"gpt-4o-mini"`
	ChatPromptPolicyVersion            string  `envconfig:"CHAT_PROMPT_POLICY_VERSION" default:"chat-prompt-policy-v1"`
	MCPToolPolicyVersion               string  `envconfig:"MCP_TOOL_POLICY_VERSION" default:"mcp-tool-policy-v1"`
	ChatDebugEnabled                   bool    `envconfig:"CHAT_DEBUG_ENABLED" default:"true"`
	ChatDebugLogConsole                bool    `envconfig:"CHAT_DEBUG_LOG_CONSOLE" default:"true"`
	ChatDebugIncludeRawText            bool    `envconfig:"CHAT_DEBUG_INCLUDE_RAW_TEXT" default:"true"`
	LangfuseEnabled                    bool    `envconfig:"LANGFUSE_ENABLED" default:"false"`
	LangfuseHost                       string  `envconfig:"LANGFUSE_HOST" default:"https://cloud.langfuse.com"`
	LangfusePublicKey                  string  `envconfig:"LANGFUSE_PUBLIC_KEY"`
	LangfuseSecretKey                  string  `envconfig:"LANGFUSE_SECRET_KEY"`
	OpenAIAPIKey                       string  `envconfig:"OPENAI_API_KEY"`
	OpenAIBaseURL                      string  `envconfig:"OPENAI_BASE_URL" default:"https://api.openai.com"`
	AnthropicAPIKey                    string  `envconfig:"ANTHROPIC_API_KEY"`
	AnthropicBaseURL                   string  `envconfig:"ANTHROPIC_BASE_URL" default:"https://api.anthropic.com"`
	AnthropicVersion                   string  `envconfig:"ANTHROPIC_VERSION" default:"2023-06-01"`
	AWSRegion                          string  `envconfig:"AWS_REGION" default:"us-east-1"`
	AWSAccessKeyID                     string  `envconfig:"AWS_ACCESS_KEY_ID"`
	AWSSecretAccessKey                 string  `envconfig:"AWS_SECRET_ACCESS_KEY"`
	AWSSessionToken                    string  `envconfig:"AWS_SESSION_TOKEN"`
	MCPTokenPepper                     string  `envconfig:"MCP_TOKEN_PEPPER" default:"engram-local-dev-mcp-token-pepper"`
	OIDCEnabled                        bool    `envconfig:"OIDC_ENABLED" default:"false"`
	OIDCIssuerURL                      string  `envconfig:"OIDC_ISSUER_URL"`
	OIDCClientID                       string  `envconfig:"OIDC_CLIENT_ID"`
	OIDCClientSecret                   string  `envconfig:"OIDC_CLIENT_SECRET"`
	OIDCRedirectURL                    string  `envconfig:"OIDC_REDIRECT_URL"`
	OIDCScopes                         string  `envconfig:"OIDC_SCOPES" default:"openid profile email"`
	OIDCUsernameClaim                  string  `envconfig:"OIDC_USERNAME_CLAIM" default:"email"`
	OAuthEnabled                       bool    `envconfig:"OAUTH_ENABLED" default:"true"`
	OAuthIssuerURL                     string  `envconfig:"OAUTH_ISSUER_URL"`
	OAuthRequireProtectedRegistration  bool    `envconfig:"OAUTH_REQUIRE_PROTECTED_REGISTRATION" default:"false"`
	OAuthAccessTokenTTLSeconds         int     `envconfig:"OAUTH_ACCESS_TOKEN_TTL_SECONDS" default:"3600"`
	OAuthAuthorizationCodeTTLSeconds   int     `envconfig:"OAUTH_AUTHORIZATION_CODE_TTL_SECONDS" default:"300"`
	OAuthClientSecretPepper            string  `envconfig:"OAUTH_CLIENT_SECRET_PEPPER" default:"engram-local-dev-oauth-client-pepper"`
}

func applyDefaults(settings *Settings) {
	setStringDefault(&settings.AppSemanticVersion, defaultAppSemanticVersion)
	setStringDefaultWithFunc(&settings.AppCommitSHA, defaultCommitSHA)
	setStringDefault(&settings.DatabaseURL, defaultDatabaseURL)
	setStringDefault(&settings.EmbeddingModel, defaultEmbeddingModel)
	setStringDefault(&settings.OpenAIBaseURL, defaultOpenAIBaseURL)
	setStringDefault(&settings.AnthropicBaseURL, defaultAnthropicBaseURL)
	setStringDefault(&settings.AnthropicVersion, defaultAnthropicVersion)
	setStringDefault(&settings.LangfuseHost, defaultLangfuseHost)
	setStringDefault(&settings.APIHost, defaultAPIServerHost)
	setStringDefault(&settings.DefaultChatProvider, defaultDefaultChatProvider)
	setStringDefault(&settings.DefaultChatModel, defaultDefaultChatModel)
	setStringDefault(&settings.ChatPromptPolicyVersion, defaultChatPromptPolicyVersion)
	setStringDefault(&settings.MCPToolPolicyVersion, defaultMCPToolPolicyVersion)
	setStringDefault(&settings.AuditLogPath, defaultAuditLogPath)
	setStringDefault(&settings.LanggraphCheckpointPath, defaultLanggraphCheckpointSQL)
}

func setStringDefault(target *string, defaultValue string) {
	if strings.TrimSpace(*target) == "" {
		*target = defaultValue
	}
}

func setStringDefaultWithFunc(target *string, defaultValue func() string) {
	if strings.TrimSpace(*target) == "" {
		*target = defaultValue()
	}
}

// LoadSettings loads environment-backed configuration and validates production security requirements.
func LoadSettings() (Settings, error) {
	var settings Settings
	if err := envconfig.Process("", &settings); err != nil {
		return Settings{}, fmt.Errorf("load settings: %w", err)
	}
	applyDefaults(&settings)
	if err := ValidateOIDCSettings(settings); err != nil {
		return Settings{}, err
	}
	if err := ValidateProductionSecurity(settings); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

// ValidateOIDCSettings enforces required OIDC fields when oidc login is enabled.
func ValidateOIDCSettings(settings Settings) error {
	if !settings.OIDCEnabled {
		return nil
	}
	missing := make([]string, 0, 4)
	if strings.TrimSpace(settings.OIDCIssuerURL) == "" {
		missing = append(missing, "OIDC_ISSUER_URL")
	}
	if strings.TrimSpace(settings.OIDCClientID) == "" {
		missing = append(missing, "OIDC_CLIENT_ID")
	}
	if strings.TrimSpace(settings.OIDCClientSecret) == "" {
		missing = append(missing, "OIDC_CLIENT_SECRET")
	}
	if strings.TrimSpace(settings.OIDCRedirectURL) == "" {
		missing = append(missing, "OIDC_REDIRECT_URL")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("oidc is enabled but missing required settings: %s", strings.Join(missing, ", "))
}

func appendSecretViolations(violations []string, name, value, insecureDefault string) []string {
	normalized := strings.TrimSpace(value)
	if normalized == insecureDefault {
		violations = append(violations, fmt.Sprintf("%s cannot use the development default in production", name))
	}
	if len(normalized) < minSecretLength {
		violations = append(violations, fmt.Sprintf("%s must be at least %d characters in production", name, minSecretLength))
	}
	return violations
}

func demoPasswordViolations(password string) []string {
	normalized := strings.TrimSpace(password)
	violations := make([]string, 0)
	if len(normalized) < minPasswordLength {
		violations = append(violations, fmt.Sprintf("UI_DEMO_PASSWORD must be at least %d characters in production", minPasswordLength))
	}
	if _, weak := weakPasswordValues[strings.ToLower(normalized)]; weak {
		violations = append(violations, "UI_DEMO_PASSWORD uses a weak value in production")
	}
	return violations
}

// ValidateProductionSecurity enforces production-only secret and auth hardening.
func ValidateProductionSecurity(settings Settings) error {
	if !IsProductionEnv(settings) {
		return nil
	}

	violations := make([]string, 0)
	violations = appendSecretViolations(violations, "APP_SESSION_SECRET", settings.AppSessionSecret, defaultSessionSecret)
	violations = appendSecretViolations(violations, "MCP_TOKEN_PEPPER", settings.MCPTokenPepper, defaultMCPTokenPepper)
	violations = appendSecretViolations(violations, "OAUTH_CLIENT_SECRET_PEPPER", settings.OAuthClientSecretPepper, defaultOAuthClientSecret)
	violations = append(violations, demoPasswordViolations(settings.UIDemoPassword)...)

	if !settings.OAuthRequireProtectedRegistration {
		violations = append(violations, "OAUTH_REQUIRE_PROTECTED_REGISTRATION must be enabled in production")
	}

	if len(violations) == 0 {
		return nil
	}
	return errors.New(strings.Join(violations, "; "))
}

// BuildDebugSettingsSnapshot returns a map suitable for dev logging with secrets redacted.
func BuildDebugSettingsSnapshot(settings Settings) map[string]any {
	snapshot := settingsMap(settings)
	for key := range sensitiveSettingKeys {
		redactIfSensitive(snapshot, key)
	}
	return snapshot
}

func redactIfSensitive(snapshot map[string]any, key string) {
	value, exists := snapshot[key]
	if !exists {
		return
	}
	stringValue, isString := value.(string)
	if !isString {
		return
	}
	if strings.TrimSpace(stringValue) == "" {
		return
	}
	snapshot[key] = "<redacted>"
}

func settingsMap(settings Settings) map[string]any {
	return map[string]any{
		"app_env":                                 settings.AppEnv,
		"log_config_in_dev":                       settings.LogConfigInDev,
		"app_semantic_version":                    settings.AppSemanticVersion,
		"app_commit_sha":                          settings.AppCommitSHA,
		"database_url":                            settings.DatabaseURL,
		"embedding_dim":                           settings.EmbeddingDim,
		"embedding_provider":                      settings.EmbeddingProvider,
		"embedding_model":                         settings.EmbeddingModel,
		"embedding_fallback_to_local":             settings.EmbeddingFallbackToLocal,
		"embedding_timeout_seconds":               settings.EmbeddingTimeoutSeconds,
		"ingestion_max_file_bytes":                settings.IngestionMaxFileBytes,
		"ingestion_max_text_chars":                settings.IngestionMaxTextChars,
		"ingestion_max_metadata_json_bytes":       settings.IngestionMaxMetadataJSONBytes,
		"api_host":                                settings.APIHost,
		"api_port":                                settings.APIPort,
		"api_request_log_enabled":                 settings.APIRequestLogEnabled,
		"api_metrics_enabled":                     settings.APIMetricsEnabled,
		"app_session_secret":                      settings.AppSessionSecret,
		"ui_demo_username":                        settings.UIDemoUsername,
		"ui_demo_password":                        settings.UIDemoPassword,
		"ui_demo_password_hash":                   settings.UIDemoPasswordHash,
		"langgraph_checkpoint_path":               settings.LanggraphCheckpointPath,
		"audit_log_path":                          settings.AuditLogPath,
		"audit_log_stdout_enabled":                settings.AuditLogStdoutEnabled,
		"audit_log_max_event_bytes":               settings.AuditLogMaxEventBytes,
		"login_rate_limit_window_seconds":         settings.LoginRateLimitWindowSeconds,
		"login_rate_limit_max_attempts":           settings.LoginRateLimitMaxAttempts,
		"login_lockout_seconds":                   settings.LoginLockoutSeconds,
		"mcp_transport_rate_limit_window_seconds": settings.MCPTransportRateLimitWindowSeconds,
		"mcp_transport_rate_limit_max_requests":   settings.MCPTransportRateLimitMaxRequests,
		"mcp_transport_rate_limit_block_seconds":  settings.MCPTransportRateLimitBlockSeconds,
		"default_chat_provider":                   settings.DefaultChatProvider,
		"default_chat_model":                      settings.DefaultChatModel,
		"chat_prompt_policy_version":              settings.ChatPromptPolicyVersion,
		"mcp_tool_policy_version":                 settings.MCPToolPolicyVersion,
		"chat_debug_enabled":                      settings.ChatDebugEnabled,
		"chat_debug_log_console":                  settings.ChatDebugLogConsole,
		"chat_debug_include_raw_text":             settings.ChatDebugIncludeRawText,
		"langfuse_enabled":                        settings.LangfuseEnabled,
		"langfuse_host":                           settings.LangfuseHost,
		"langfuse_public_key":                     settings.LangfusePublicKey,
		"langfuse_secret_key":                     settings.LangfuseSecretKey,
		"openai_api_key":                          settings.OpenAIAPIKey,
		"openai_base_url":                         settings.OpenAIBaseURL,
		"anthropic_api_key":                       settings.AnthropicAPIKey,
		"anthropic_base_url":                      settings.AnthropicBaseURL,
		"anthropic_version":                       settings.AnthropicVersion,
		"aws_region":                              settings.AWSRegion,
		"aws_access_key_id":                       settings.AWSAccessKeyID,
		"aws_secret_access_key":                   settings.AWSSecretAccessKey,
		"aws_session_token":                       settings.AWSSessionToken,
		"mcp_token_pepper":                        settings.MCPTokenPepper,
		"oidc_enabled":                            settings.OIDCEnabled,
		"oidc_issuer_url":                         settings.OIDCIssuerURL,
		"oidc_client_id":                          settings.OIDCClientID,
		"oidc_client_secret":                      settings.OIDCClientSecret,
		"oidc_redirect_url":                       settings.OIDCRedirectURL,
		"oidc_scopes":                             settings.OIDCScopes,
		"oidc_username_claim":                     settings.OIDCUsernameClaim,
		"oauth_enabled":                           settings.OAuthEnabled,
		"oauth_issuer_url":                        settings.OAuthIssuerURL,
		"oauth_require_protected_registration":    settings.OAuthRequireProtectedRegistration,
		"oauth_access_token_ttl_seconds":          settings.OAuthAccessTokenTTLSeconds,
		"oauth_authorization_code_ttl_seconds":    settings.OAuthAuthorizationCodeTTLSeconds,
		"oauth_client_secret_pepper":              settings.OAuthClientSecretPepper,
	}
}

// ShouldLogSettings gates debug-config logging to development environments.
func ShouldLogSettings(settings Settings) bool {
	if !settings.LogConfigInDev {
		return false
	}
	_, isDev := devEnvNames[strings.ToLower(strings.TrimSpace(settings.AppEnv))]
	return isDev
}

// IsProductionEnv returns true for production-like environments.
func IsProductionEnv(settings Settings) bool {
	_, isProd := productionEnvNames[strings.ToLower(strings.TrimSpace(settings.AppEnv))]
	return isProd
}

func defaultCommitSHA() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return defaultAppCommitSHA
	}
	commit := strings.TrimSpace(string(output))
	if commit == "" {
		return defaultAppCommitSHA
	}
	return commit
}
