package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ChatProvider identifies the configured LLM provider.
type ChatProvider string

const (
	ChatProviderOpenAI    ChatProvider = "openai"
	ChatProviderAnthropic ChatProvider = "anthropic"
	ChatProviderBedrock   ChatProvider = "bedrock"
)

// ParseChatProvider validates provider values from storage and inputs.
func ParseChatProvider(value string) (ChatProvider, error) {
	switch ChatProvider(value) {
	case ChatProviderOpenAI, ChatProviderAnthropic, ChatProviderBedrock:
		return ChatProvider(value), nil
	default:
		return "", fmt.Errorf("unsupported chat provider %q", value)
	}
}

// VisibilityScope controls whether resources are private or project-visible.
type VisibilityScope string

const (
	VisibilityScopePrivate VisibilityScope = "private"
	VisibilityScopeProject VisibilityScope = "project"
)

// ParseVisibilityScope validates visibility values from storage and inputs.
func ParseVisibilityScope(value string) (VisibilityScope, error) {
	switch VisibilityScope(value) {
	case VisibilityScopePrivate, VisibilityScopeProject:
		return VisibilityScope(value), nil
	default:
		return "", fmt.Errorf("unsupported visibility scope %q", value)
	}
}

// ChatAutosaveStrategy captures snapshot creation policy.
type ChatAutosaveStrategy string

const (
	ChatAutosaveStrategyOff          ChatAutosaveStrategy = "off"
	ChatAutosaveStrategyInterval     ChatAutosaveStrategy = "interval"
	ChatAutosaveStrategyMessageCount ChatAutosaveStrategy = "message_count"
)

// ParseChatAutosaveStrategy validates autosave strategy values.
func ParseChatAutosaveStrategy(value string) (ChatAutosaveStrategy, error) {
	switch ChatAutosaveStrategy(value) {
	case ChatAutosaveStrategyOff, ChatAutosaveStrategyInterval, ChatAutosaveStrategyMessageCount:
		return ChatAutosaveStrategy(value), nil
	default:
		return "", fmt.Errorf("unsupported autosave strategy %q", value)
	}
}

// ChatSessionCreateRequest models chat session create payload.
type ChatSessionCreateRequest struct {
	ProjectID               string               `json:"project_id"`
	Title                   string               `json:"title"`
	Provider                ChatProvider         `json:"provider"`
	ModelID                 string               `json:"model_id"`
	SystemPrompt            string               `json:"system_prompt"`
	VisibilityScope         VisibilityScope      `json:"visibility_scope"`
	AutosaveEnabled         bool                 `json:"autosave_enabled"`
	AutosaveStrategy        ChatAutosaveStrategy `json:"autosave_strategy"`
	AutosaveIntervalMinutes int                  `json:"autosave_interval_minutes"`
	AutosaveMinMessages     int                  `json:"autosave_min_messages"`
	RetentionDays           int                  `json:"retention_days"`
	RetentionMaxSnapshots   int                  `json:"retention_max_snapshots"`
}

// ChatSessionUpdateRequest models mutable chat session fields.
type ChatSessionUpdateRequest struct {
	Title                   *string               `json:"title,omitempty"`
	Provider                *ChatProvider         `json:"provider,omitempty"`
	ModelID                 *string               `json:"model_id,omitempty"`
	SystemPrompt            *string               `json:"system_prompt,omitempty"`
	VisibilityScope         *VisibilityScope      `json:"visibility_scope,omitempty"`
	AutosaveEnabled         *bool                 `json:"autosave_enabled,omitempty"`
	AutosaveStrategy        *ChatAutosaveStrategy `json:"autosave_strategy,omitempty"`
	AutosaveIntervalMinutes *int                  `json:"autosave_interval_minutes,omitempty"`
	AutosaveMinMessages     *int                  `json:"autosave_min_messages,omitempty"`
	RetentionDays           *int                  `json:"retention_days,omitempty"`
	RetentionMaxSnapshots   *int                  `json:"retention_max_snapshots,omitempty"`
}

// ChatSessionRecord models persisted chat session rows.
type ChatSessionRecord struct {
	SessionID               uuid.UUID            `json:"session_id"`
	OwnerUserID             uuid.UUID            `json:"owner_user_id"`
	ProjectID               string               `json:"project_id"`
	Title                   string               `json:"title"`
	Provider                ChatProvider         `json:"provider"`
	ModelID                 string               `json:"model_id"`
	SystemPrompt            string               `json:"system_prompt"`
	VisibilityScope         VisibilityScope      `json:"visibility_scope"`
	AutosaveEnabled         bool                 `json:"autosave_enabled"`
	AutosaveStrategy        ChatAutosaveStrategy `json:"autosave_strategy"`
	AutosaveIntervalMinutes int                  `json:"autosave_interval_minutes"`
	AutosaveMinMessages     int                  `json:"autosave_min_messages"`
	RetentionDays           int                  `json:"retention_days"`
	RetentionMaxSnapshots   int                  `json:"retention_max_snapshots"`
	CreatedAt               time.Time            `json:"created_at"`
	UpdatedAt               time.Time            `json:"updated_at"`
}

// ChatSessionAdminRecord models admin-level session lookup fields.
type ChatSessionAdminRecord struct {
	SessionID   uuid.UUID  `json:"session_id"`
	OwnerUserID uuid.UUID  `json:"owner_user_id"`
	ProjectID   string     `json:"project_id"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// ChatMessageRecord models persisted chat message rows.
type ChatMessageRecord struct {
	MessageID      uuid.UUID      `json:"message_id"`
	SessionID      uuid.UUID      `json:"session_id"`
	Role           string         `json:"role"`
	ContentText    string         `json:"content_text"`
	Provider       *string        `json:"provider,omitempty"`
	ModelID        *string        `json:"model_id,omitempty"`
	TokenUsageJSON map[string]any `json:"token_usage_json"`
	UsedEngramIDs  []uuid.UUID    `json:"used_engram_ids"`
	CreatedAt      time.Time      `json:"created_at"`
}

// PinnedEngramRecord models engram pins associated to chat sessions.
type PinnedEngramRecord struct {
	SessionID      uuid.UUID `json:"session_id"`
	EngramID       uuid.UUID `json:"engram_id"`
	PinnedByUserID uuid.UUID `json:"pinned_by_user_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// PinnedDocumentRecord models document pins associated to chat sessions.
type PinnedDocumentRecord struct {
	SessionID      uuid.UUID `json:"session_id"`
	DocumentID     uuid.UUID `json:"document_id"`
	PinnedByUserID uuid.UUID `json:"pinned_by_user_id"`
	CreatedAt      time.Time `json:"created_at"`
}
