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
	rowValues := chatSessionRowValues(chatSessionRowFixture{
		SessionID:             sessionID,
		OwnerUserID:           ownerUserID,
		ProjectID:             "engram-vault",
		Title:                 "Daily triage",
		Provider:              "openai",
		ModelID:               "gpt-4o-mini",
		SystemPrompt:          "be concise",
		VisibilityScope:       "private",
		AutosaveEnabled:       false,
		AutosaveStrategy:      "off",
		AutosaveIntervalMins:  30,
		AutosaveMinMessages:   6,
		RetentionDays:         30,
		RetentionMaxSnapshots: 60,
		CreatedAt:             createdAt,
		UpdatedAt:             createdAt,
	})
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
				chatSessionRowValues(chatSessionRowFixture{
					SessionID:             sessionID,
					OwnerUserID:           actorUserID,
					ProjectID:             "engram-vault",
					Title:                 "Daily triage",
					Provider:              "openai",
					ModelID:               "gpt-4o-mini",
					SystemPrompt:          "",
					VisibilityScope:       "project",
					AutosaveEnabled:       true,
					AutosaveStrategy:      "interval",
					AutosaveIntervalMins:  15,
					AutosaveMinMessages:   4,
					RetentionDays:         14,
					RetentionMaxSnapshots: 20,
					CreatedAt:             createdAt,
					UpdatedAt:             createdAt,
				}),
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
	if !strings.Contains(query, "project_members pm") || !strings.Contains(query, "actor_user.role = 'admin'") {
		t.Fatalf("expected membership/admin visibility clause in query, got %q", query)
	}
	if !strings.Contains(query, "project_id = $2") {
		t.Fatalf("expected project filter clause in query, got %q", query)
	}
	expectedArgs := []any{actorUserID, projectID, 25, 5}
	if !reflect.DeepEqual(db.queryArgs[0], expectedArgs) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestGetChatSessionAdminRecordReturnsRecord(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000525")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000526")
	deletedAt := time.Date(2026, 2, 21, 9, 45, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{sessionID, ownerUserID, "project-admin", deletedAt},
		},
	}

	record, err := GetChatSessionAdminRecord(context.Background(), db, sessionID)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, sessionID, record.SessionID)
	requireEqual(t, ownerUserID, record.OwnerUserID)
	requireEqual(t, "project-admin", record.ProjectID)
	if record.DeletedAt == nil {
		t.Fatalf("expected deleted_at to be populated")
	}
	requireEqual(t, deletedAt, *record.DeletedAt)
}

func TestChatSessionGetAndUpdateReturnNilWhenRowMissing(t *testing.T) {
	db := &fakeQueryer{
		queryRowResult: &fakeRow{err: pgx.ErrNoRows},
	}

	testCases := []struct {
		name    string
		execute func() (*models.ChatSessionRecord, error)
	}{
		{
			name: "get",
			execute: func() (*models.ChatSessionRecord, error) {
				return GetChatSession(
					context.Background(),
					db,
					ChatSessionGetInput{
						SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000000521"),
						ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000522"),
					},
				)
			},
		},
		{
			name: "update",
			execute: func() (*models.ChatSessionRecord, error) {
				title := "Updated title"
				return UpdateChatSession(
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
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			record, err := testCase.execute()
			requireNoError(t, err)
			if record != nil {
				t.Fatalf("expected nil session when row missing")
			}
		})
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

type chatSessionRowFixture struct {
	SessionID             uuid.UUID
	OwnerUserID           uuid.UUID
	ProjectID             string
	Title                 string
	Provider              string
	ModelID               string
	SystemPrompt          string
	VisibilityScope       string
	AutosaveEnabled       bool
	AutosaveStrategy      string
	AutosaveIntervalMins  int
	AutosaveMinMessages   int
	RetentionDays         int
	RetentionMaxSnapshots int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func chatSessionRowValues(fixture chatSessionRowFixture) []any {
	return []any{
		fixture.SessionID,
		fixture.OwnerUserID,
		fixture.ProjectID,
		fixture.Title,
		fixture.Provider,
		fixture.ModelID,
		fixture.SystemPrompt,
		fixture.VisibilityScope,
		fixture.AutosaveEnabled,
		fixture.AutosaveStrategy,
		fixture.AutosaveIntervalMins,
		fixture.AutosaveMinMessages,
		fixture.RetentionDays,
		fixture.RetentionMaxSnapshots,
		fixture.CreatedAt,
		fixture.UpdatedAt,
	}
}
