package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestCreateChatSessionReturnsInsertedRecord(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000501")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000502")
	createdAt := time.Date(2026, 2, 20, 16, 0, 0, 0, time.UTC)
	rowValues := chatSessionRowValues(
		sessionID,
		ownerUserID,
		"engram-vault",
		"Daily triage",
		"openai",
		"gpt-4o-mini",
		"be concise",
		"private",
		false,
		"off",
		30,
		6,
		30,
		60,
		createdAt,
		createdAt,
	)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: rowValues},
	}

	originalUUID := newChatUUID
	newChatUUID = func() uuid.UUID { return sessionID }
	t.Cleanup(func() { newChatUUID = originalUUID })

	originalNow := nowChatUTC
	nowChatUTC = func() time.Time { return createdAt }
	t.Cleanup(func() { nowChatUTC = originalNow })

	record, err := CreateChatSession(
		context.Background(),
		db,
		ChatSessionCreateInput{
			OwnerUserID: ownerUserID,
			Payload: models.ChatSessionCreateRequest{
				ProjectID:    "engram-vault",
				Title:        "Daily triage",
				Provider:     models.ChatProviderOpenAI,
				ModelID:      "gpt-4o-mini",
				SystemPrompt: "be concise",
			},
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, sessionID, record.SessionID)
	requireEqual(t, models.ChatProviderOpenAI, record.Provider)
	requireEqual(t, models.VisibilityScopePrivate, record.VisibilityScope)
	requireEqual(t, models.ChatAutosaveStrategyOff, record.AutosaveStrategy)
	requireEqual(t, 1, len(db.queryRowArgs))
	requireEqual(t, sessionID, db.queryRowArgs[0][0].(uuid.UUID))
}

func TestListChatSessionsAppliesVisibilityAndProjectFilter(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000511")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000512")
	createdAt := time.Date(2026, 2, 20, 16, 5, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				chatSessionRowValues(
					sessionID,
					actorUserID,
					"engram-vault",
					"Daily triage",
					"openai",
					"gpt-4o-mini",
					"",
					"project",
					true,
					"interval",
					15,
					4,
					14,
					20,
					createdAt,
					createdAt,
				),
			},
		},
	}
	projectID := "engram-vault"

	records, err := ListChatSessions(
		context.Background(),
		db,
		ChatSessionListInput{
			ActorUserID: actorUserID,
			ProjectID:   &projectID,
			Limit:       25,
			Offset:      5,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, models.VisibilityScopeProject, records[0].VisibilityScope)
	requireEqual(t, models.ChatAutosaveStrategyInterval, records[0].AutosaveStrategy)

	requireEqual(t, 1, len(db.querySQL))
	query := db.querySQL[0]
	if !strings.Contains(query, "(owner_user_id = $1 OR visibility_scope = 'project')") {
		t.Fatalf("expected visibility clause in query, got %q", query)
	}
	if !strings.Contains(query, "project_id = $2") {
		t.Fatalf("expected project filter clause in query, got %q", query)
	}
	expectedArgs := []any{actorUserID, projectID, 25, 5}
	if !reflect.DeepEqual(db.queryArgs[0], expectedArgs) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestGetChatSessionReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{
		queryRowResult: &fakeRow{err: pgx.ErrNoRows},
	}

	record, err := GetChatSession(
		context.Background(),
		db,
		ChatSessionGetInput{
			SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000000521"),
			ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000522"),
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil session when row missing")
	}
}

func TestUpdateChatSessionReturnsNilWhenNoRows(t *testing.T) {
	db := &fakeQueryer{
		queryRowResult: &fakeRow{err: pgx.ErrNoRows},
	}
	title := "Updated title"

	record, err := UpdateChatSession(
		context.Background(),
		db,
		ChatSessionUpdateInput{
			SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000000531"),
			ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000532"),
			Payload: models.ChatSessionUpdateRequest{
				Title: &title,
			},
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil session when update affected no rows")
	}
}

func TestUpdateChatSessionValidatesProviderValue(t *testing.T) {
	db := &fakeQueryer{}
	invalidProvider := models.ChatProvider("unknown")

	record, err := UpdateChatSession(
		context.Background(),
		db,
		ChatSessionUpdateInput{
			SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000000541"),
			ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000542"),
			Payload: models.ChatSessionUpdateRequest{
				Provider: &invalidProvider,
			},
		},
	)
	if err == nil {
		t.Fatalf("expected validation error for invalid provider")
	}
	if !strings.Contains(err.Error(), "unsupported chat provider") {
		t.Fatalf("expected invalid provider error, got %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil record on validation error")
	}
	if len(db.queryRowSQL) != 0 {
		t.Fatalf("expected no DB call on validation error")
	}
}

func TestNormalizeCreatePayloadRejectsInvalidVisibility(t *testing.T) {
	_, err := normalizeCreatePayload(
		models.ChatSessionCreateRequest{
			ProjectID:       "engram-vault",
			Title:           "Daily triage",
			Provider:        models.ChatProviderOpenAI,
			ModelID:         "gpt-4o-mini",
			VisibilityScope: models.VisibilityScope("bad"),
		},
	)
	if err == nil {
		t.Fatalf("expected invalid visibility error")
	}
	if !strings.Contains(err.Error(), "unsupported visibility scope") {
		t.Fatalf("expected visibility scope error, got %v", err)
	}
}

func chatSessionRowValues(
	sessionID uuid.UUID,
	ownerUserID uuid.UUID,
	projectID string,
	title string,
	provider string,
	modelID string,
	systemPrompt string,
	visibilityScope string,
	autosaveEnabled bool,
	autosaveStrategy string,
	autosaveIntervalMinutes int,
	autosaveMinMessages int,
	retentionDays int,
	retentionMaxSnapshots int,
	createdAt time.Time,
	updatedAt time.Time,
) []any {
	return []any{
		sessionID,
		ownerUserID,
		projectID,
		title,
		provider,
		modelID,
		systemPrompt,
		visibilityScope,
		autosaveEnabled,
		autosaveStrategy,
		autosaveIntervalMinutes,
		autosaveMinMessages,
		retentionDays,
		retentionMaxSnapshots,
		createdAt,
		updatedAt,
	}
}
