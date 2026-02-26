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

// ChatTimelineEvent models timeline activity rows derived from linked engrams.
type ChatTimelineEvent struct {
	EventID                  uuid.UUID `json:"event_id"`
	SessionID                uuid.UUID `json:"session_id"`
	EventType                string    `json:"event_type"`
	Title                    string    `json:"title"`
	Abstract                 string    `json:"abstract"`
	Tags                     []string  `json:"tags"`
	ConsolidationGroupKey    *string   `json:"consolidation_group_key,omitempty"`
	ConsolidationMergedCount *int      `json:"consolidation_merged_count,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
}

// AdminChatSessionRecord models admin-level chat session rows including soft-delete metadata.
type AdminChatSessionRecord struct {
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
	DeletedAt               *time.Time           `json:"deleted_at,omitempty"`
	DeletedByUserID         *uuid.UUID           `json:"deleted_by_user_id,omitempty"`
	DeleteReason            *string              `json:"delete_reason,omitempty"`
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

// SaveSessionAsEngramRequest models chat-session save payload.
type SaveSessionAsEngramRequest struct {
	Title           string          `json:"title"`
	Abstract        string          `json:"abstract"`
	VisibilityScope VisibilityScope `json:"visibility_scope"`
	Tags            []string        `json:"tags"`
	Keywords        []string        `json:"keywords"`
}

// SaveSessionAsEngramResponse models save-session result payload.
type SaveSessionAsEngramResponse struct {
	EngramID  uuid.UUID `json:"engram_id"`
	SessionID uuid.UUID `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ContinueSessionRequest models continue-session payload.
type ContinueSessionRequest struct {
	Title *string `json:"title,omitempty"`
}

// ContinueSessionResponse models continue-session result payload.
type ContinueSessionResponse struct {
	Session          ChatSessionRecord `json:"session"`
	CarriedEngramIDs []uuid.UUID       `json:"carried_engram_ids"`
}

// PinEngramRequest models engram pin payloads.
type PinEngramRequest struct {
	EngramID uuid.UUID `json:"engram_id"`
}

// PinDocumentRequest models document pin payloads.
type PinDocumentRequest struct {
	DocumentID uuid.UUID `json:"document_id"`
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
