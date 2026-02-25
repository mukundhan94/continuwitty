package chat

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestDeriveChatSnapshotAbstractPrefersLatestAssistant(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000001101")
	messages := []models.ChatMessageRecord{
		chatMessage(sessionID, "user", "First question"),
		chatMessage(sessionID, "assistant", "Older summary"),
		chatMessage(sessionID, "assistant", "Latest assistant summary"),
	}
	requireEqualStringSession(t, "Latest assistant summary", DeriveChatSnapshotAbstract(messages, 320))
}

func TestRunSessionLifecycleReturnsDisabledWithoutSideEffects(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000001102")
	session := sessionRecord(actorID)
	session.AutosaveEnabled = false
	session.AutosaveStrategy = models.ChatAutosaveStrategyOff
	calls := map[string]int{"linked": 0, "messages": 0, "count": 0, "delete": 0, "create": 0}
	dependencies := SessionLifecycleDependencies{
		ListSessionLinkedEngrams: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ int) ([]models.EngramSummary, error) {
			calls["linked"]++
			return []models.EngramSummary{}, nil
		},
		ListChatMessages: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ int) ([]models.ChatMessageRecord, error) {
			calls["messages"]++
			return []models.ChatMessageRecord{}, nil
		},
		CountSessionMessagesByRole: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (int, error) {
			calls["count"]++
			return 0, nil
		},
		DeleteSessionAutosaveEngrams: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ []uuid.UUID) ([]uuid.UUID, error) {
			calls["delete"]++
			return []uuid.UUID{}, nil
		},
		CreateEngram: func(_ context.Context, _ models.MemoryEngramCreate, _ int, _ uuid.UUID, _ string) (*models.EngramCreateResponse, error) {
			calls["create"]++
			createdAt := time.Date(2026, 2, 25, 18, 0, 0, 0, time.UTC)
			return &models.EngramCreateResponse{EngramID: uuid.MustParse("00000000-0000-0000-0000-000000001103"), CreatedAt: createdAt}, nil
		},
	}

	result, err := RunSessionLifecycleMaintenance(context.Background(), actorID, session, 256, dependencies)
	requireNoErrorSession(t, err)
	if result.SnapshotEngramID != nil {
		t.Fatalf("expected no snapshot id when autosave disabled")
	}
	requireEqualIntSession(t, 0, len(result.PrunedEngramIDs))
	requireEqualStringSession(t, "autosave_disabled", valueOrEmptySession(result.SkippedReason))
	requireEqualIntSession(t, 0, calls["linked"])
	requireEqualIntSession(t, 0, calls["messages"])
	requireEqualIntSession(t, 0, calls["count"])
	requireEqualIntSession(t, 0, calls["delete"])
	requireEqualIntSession(t, 0, calls["create"])
}

func TestRunSessionLifecycleSkipsMessageCountThreshold(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000001104")
	session := sessionRecord(actorID)
	session.AutosaveEnabled = true
	session.AutosaveStrategy = models.ChatAutosaveStrategyMessageCount
	session.AutosaveMinMessages = 5
	session.RetentionDays = 30
	session.RetentionMaxSnapshots = 10
	existingSnapshot := models.EngramSummary{
		EngramID:  uuid.MustParse("00000000-0000-0000-0000-000000001105"),
		ProjectID: session.ProjectID,
		ThreadID:  stringPtr("chat-session:" + session.SessionID.String() + ":autosave"),
		Title:     "Snapshot",
		Abstract:  "Useful summary",
		CreatedAt: time.Date(2026, 2, 25, 18, 0, 0, 0, time.UTC),
		Tags:      []string{"autosave_snapshot"},
		Keywords:  []string{},
	}
	deleteCalls := make([][]uuid.UUID, 0)
	createCalls := 0
	dependencies := SessionLifecycleDependencies{
		ListSessionLinkedEngrams: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ int) ([]models.EngramSummary, error) {
			return []models.EngramSummary{existingSnapshot}, nil
		},
		ListChatMessages: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ int) ([]models.ChatMessageRecord, error) {
			return []models.ChatMessageRecord{}, nil
		},
		CountSessionMessagesByRole: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (int, error) {
			return 2, nil
		},
		DeleteSessionAutosaveEngrams: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, engramIDs []uuid.UUID) ([]uuid.UUID, error) {
			deleteCalls = append(deleteCalls, engramIDs)
			return []uuid.UUID{}, nil
		},
		CreateEngram: func(_ context.Context, _ models.MemoryEngramCreate, _ int, _ uuid.UUID, _ string) (*models.EngramCreateResponse, error) {
			createCalls++
			createdAt := time.Date(2026, 2, 25, 18, 0, 0, 0, time.UTC)
			return &models.EngramCreateResponse{EngramID: uuid.MustParse("00000000-0000-0000-0000-000000001106"), CreatedAt: createdAt}, nil
		},
	}

	result, err := RunSessionLifecycleMaintenance(context.Background(), actorID, session, 256, dependencies)
	requireNoErrorSession(t, err)
	if result.SnapshotEngramID != nil {
		t.Fatalf("expected no snapshot id when threshold not met")
	}
	requireEqualStringSession(t, "message_count_threshold_not_met", valueOrEmptySession(result.SkippedReason))
	requireEqualIntSession(t, 1, len(deleteCalls))
	requireEqualIntSession(t, 0, len(deleteCalls[0]))
	requireEqualIntSession(t, 0, createCalls)
}

func TestRunSessionLifecycleCreatesSnapshotWhenThresholdIsMet(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000001107")
	session := sessionRecord(actorID)
	session.AutosaveEnabled = true
	session.AutosaveStrategy = models.ChatAutosaveStrategyMessageCount
	session.AutosaveMinMessages = 1
	session.RetentionDays = 30
	session.RetentionMaxSnapshots = 10
	createdID := uuid.MustParse("00000000-0000-0000-0000-000000001108")
	var createdPayload *models.MemoryEngramCreate
	dependencies := SessionLifecycleDependencies{
		ListSessionLinkedEngrams: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ int) ([]models.EngramSummary, error) {
			return []models.EngramSummary{}, nil
		},
		ListChatMessages: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ int) ([]models.ChatMessageRecord, error) {
			return []models.ChatMessageRecord{
				chatMessage(session.SessionID, "user", "Need help with a query"),
				chatMessage(session.SessionID, "assistant", "Detailed assistant summary with concrete decisions, actions, and next steps."),
			}, nil
		},
		CountSessionMessagesByRole: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (int, error) {
			return 1, nil
		},
		DeleteSessionAutosaveEngrams: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ []uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{}, nil
		},
		CreateEngram: func(_ context.Context, payload models.MemoryEngramCreate, _ int, _ uuid.UUID, _ string) (*models.EngramCreateResponse, error) {
			createdPayload = &payload
			createdAt := time.Date(2026, 2, 25, 18, 0, 0, 0, time.UTC)
			return &models.EngramCreateResponse{EngramID: createdID, CreatedAt: createdAt}, nil
		},
	}

	result, err := RunSessionLifecycleMaintenance(context.Background(), actorID, session, 256, dependencies)
	requireNoErrorSession(t, err)
	requireEqualStringSession(t, createdID.String(), result.SnapshotEngramID.String())
	requireEqualIntSession(t, 0, len(result.PrunedEngramIDs))
	if result.SkippedReason != nil {
		t.Fatalf("expected nil skipped reason, got %q", *result.SkippedReason)
	}
	if createdPayload == nil {
		t.Fatalf("expected created payload")
	}
	requireEqualStringSession(t, session.ProjectID, createdPayload.ProjectID)
	requireEqualStringSession(t, session.SessionID.String(), createdPayload.SourceSessionID.String())
	if !containsString(createdPayload.Tags, "autosave_snapshot") {
		t.Fatalf("expected autosave_snapshot tag in %v", createdPayload.Tags)
	}
}

func sessionRecord(ownerUserID uuid.UUID) models.ChatSessionRecord {
	now := time.Date(2026, 2, 25, 18, 0, 0, 0, time.UTC)
	return models.ChatSessionRecord{
		SessionID:               uuid.MustParse("00000000-0000-0000-0000-000000001100"),
		OwnerUserID:             ownerUserID,
		ProjectID:               "project-chat",
		Title:                   "Session",
		Provider:                models.ChatProviderOpenAI,
		ModelID:                 "gpt-4o-mini",
		SystemPrompt:            "be helpful",
		VisibilityScope:         models.VisibilityScopePrivate,
		AutosaveEnabled:         false,
		AutosaveStrategy:        models.ChatAutosaveStrategyOff,
		AutosaveIntervalMinutes: 30,
		AutosaveMinMessages:     6,
		RetentionDays:           30,
		RetentionMaxSnapshots:   60,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}

func chatMessage(sessionID uuid.UUID, role string, contentText string) models.ChatMessageRecord {
	return models.ChatMessageRecord{
		MessageID:      uuid.New(),
		SessionID:      sessionID,
		Role:           role,
		ContentText:    contentText,
		Provider:       nil,
		ModelID:        nil,
		TokenUsageJSON: map[string]any{},
		UsedEngramIDs:  []uuid.UUID{},
		CreatedAt:      time.Date(2026, 2, 25, 18, 0, 0, 0, time.UTC),
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func valueOrEmptySession(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func requireNoErrorSession(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func requireEqualStringSession[T ~string](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

func requireEqualIntSession(t *testing.T, expected int, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}
