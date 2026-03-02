package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestCreateChatMessageReturnsInsertedRecord(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000601")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000602")
	messageID := uuid.MustParse("00000000-0000-0000-0000-000000000603")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000604")
	createdAt := time.Date(2026, 2, 22, 9, 30, 0, 0, time.UTC)
	provider := "openai"
	modelID := "gpt-4o-mini"
	tokenUsage := map[string]any{"input": 3, "output": 1}
	usedEngramIDs := []uuid.UUID{engramID}
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: []any{
			messageID,
			sessionID,
			"user",
			"hello world",
			provider,
			modelID,
			tokenUsage,
			usedEngramIDs,
			createdAt,
		}},
	}

	originalMessageUUID := newChatMessageUUID
	newChatMessageUUID = func() uuid.UUID { return messageID }
	t.Cleanup(func() { newChatMessageUUID = originalMessageUUID })

	originalNow := nowChatUTC
	nowChatUTC = func() time.Time { return createdAt }
	t.Cleanup(func() { nowChatUTC = originalNow })

	record, err := CreateChatMessage(
		context.Background(),
		db,
		ChatMessageCreateInput{
			SessionID:   sessionID,
			ActorUserID: actorUserID,
			Role:        "user",
			ContentText: "hello world",
			Metadata: &ChatMessageMetadata{
				Provider:       &provider,
				ModelID:        &modelID,
				TokenUsageJSON: tokenUsage,
				UsedEngramIDs:  usedEngramIDs,
			},
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, messageID, record.MessageID)
	requireEqual(t, sessionID, record.SessionID)
	requireEqual(t, "user", record.Role)
	requireEqual(t, provider, *record.Provider)
	requireEqual(t, modelID, *record.ModelID)
	if !reflect.DeepEqual(tokenUsage, record.TokenUsageJSON) {
		t.Fatalf("expected token usage %#v, got %#v", tokenUsage, record.TokenUsageJSON)
	}
	if !reflect.DeepEqual(usedEngramIDs, record.UsedEngramIDs) {
		t.Fatalf("expected used engrams %#v, got %#v", usedEngramIDs, record.UsedEngramIDs)
	}

	requireEqual(t, 1, len(db.queryRowSQL))
	if !strings.Contains(db.queryRowSQL[0], "INSERT INTO chat_messages") {
		t.Fatalf("expected insert query, got %q", db.queryRowSQL[0])
	}
	requireEqual(t, 1, len(db.queryRowArgs))
	expectedArgs := []any{
		messageID,
		"user",
		"hello world",
		&provider,
		&modelID,
		tokenUsage,
		usedEngramIDs,
		createdAt,
		sessionID,
		actorUserID,
	}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestCreateChatMessageReturnsNilWhenSessionNotVisible(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}

	record, err := CreateChatMessage(
		context.Background(),
		db,
		ChatMessageCreateInput{
			SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000000611"),
			ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000612"),
			Role:        "assistant",
			ContentText: "blocked",
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil message when session not visible")
	}
}

func TestListChatMessagesParsesJSONAndDefaults(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000621")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000622")
	messageID := uuid.MustParse("00000000-0000-0000-0000-000000000623")
	createdAt := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{
			{
				messageID,
				sessionID,
				"assistant",
				"response",
				nil,
				nil,
				[]byte(`{"input":4,"output":9}`),
				[]uuid.UUID{},
				createdAt,
			},
		}},
	}

	records, err := ListChatMessages(
		context.Background(),
		db,
		ChatMessageListInput{
			SessionID:   sessionID,
			ActorUserID: actorUserID,
			Limit:       50,
			Offset:      0,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, "assistant", records[0].Role)
	if records[0].Provider != nil {
		t.Fatalf("expected nil provider")
	}
	if records[0].ModelID != nil {
		t.Fatalf("expected nil model id")
	}
	if len(records[0].UsedEngramIDs) != 0 {
		t.Fatalf("expected default empty used engram ids")
	}
	if got, ok := records[0].TokenUsageJSON["input"].(float64); !ok || got != 4 {
		t.Fatalf("expected token input value 4, got %#v", records[0].TokenUsageJSON)
	}

	requireEqual(t, 1, len(db.queryArgs))
	expectedArgs := []any{sessionID, actorUserID, 50, 0}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestListSessionLinkedEngramsAppliesVisibilityFilters(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000631")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000632")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000633")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000634")
	createdAt := time.Date(2026, 2, 22, 10, 15, 0, 0, time.UTC)
	threadID := "chat-session-thread"
	visibility := "project"
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{
			{
				engramID,
				"project-lifecycle",
				threadID,
				"Autosave",
				"snapshot",
				createdAt,
				[]string{"autosave_snapshot"},
				[]string{"lifecycle"},
				ownerUserID,
				visibility,
			},
		}},
	}

	records, err := ListSessionLinkedEngrams(
		context.Background(),
		db,
		SessionLinkedEngramsListInput{
			SessionID:   sessionID,
			ActorUserID: actorUserID,
			Limit:       25,
			Offset:      5,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, engramID, records[0].EngramID)
	requireEqual(t, "project", records[0].VisibilityScope)

	requireEqual(t, 1, len(db.querySQL))
	query := db.querySQL[0]
	if !strings.Contains(query, "e.source_session_id") {
		t.Fatalf("expected source_session_id clause, got %q", query)
	}
	if !strings.Contains(query, "e.deleted_at IS NULL") {
		t.Fatalf("expected engram visibility clause, got %q", query)
	}
	expectedArgs := []any{sessionID, actorUserID, actorUserID, 25, 5}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestCountSessionMessagesByRoleReturnsValue(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000641")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000642")
	db := &fakeQueryer{queryRowResult: &fakeRow{values: []any{3}}}

	count, err := CountSessionMessagesByRole(
		context.Background(),
		db,
		SessionMessageRoleCountInput{
			SessionID:   sessionID,
			ActorUserID: actorUserID,
			Role:        "assistant",
		},
	)
	requireNoError(t, err)
	requireEqual(t, 3, count)
	expectedArgs := []any{sessionID, "assistant", actorUserID}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestDeleteSessionAutosaveEngramsReturnsDeletedIDs(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000651")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000652")
	engramA := uuid.MustParse("00000000-0000-0000-0000-000000000653")
	engramB := uuid.MustParse("00000000-0000-0000-0000-000000000654")
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{{engramA}, {engramB}}},
	}

	removed, err := DeleteSessionAutosaveEngrams(
		context.Background(),
		db,
		SessionAutosaveDeleteInput{
			SessionID:   sessionID,
			ActorUserID: actorUserID,
			EngramIDs:   []uuid.UUID{engramA, engramB},
		},
	)
	requireNoError(t, err)
	if !reflect.DeepEqual([]uuid.UUID{engramA, engramB}, removed) {
		t.Fatalf("expected removed ids [%s %s], got %#v", engramA, engramB, removed)
	}

	query := db.querySQL[0]
	if !strings.Contains(query, "e.tags @> ARRAY['autosave_snapshot']::TEXT[]") {
		t.Fatalf("expected autosave tag filter clause, got %q", query)
	}
	expectedArgs := []any{sessionID, []uuid.UUID{engramA, engramB}, actorUserID, actorUserID}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestDeleteSessionAutosaveEngramsSkipsQueryWhenNoIDs(t *testing.T) {
	db := &fakeQueryer{}

	removed, err := DeleteSessionAutosaveEngrams(
		context.Background(),
		db,
		SessionAutosaveDeleteInput{
			SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000000661"),
			ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000662"),
			EngramIDs:   nil,
		},
	)
	requireNoError(t, err)
	if len(removed) != 0 {
		t.Fatalf("expected empty removed ids, got %#v", removed)
	}
	if len(db.querySQL) != 0 {
		t.Fatalf("expected no query execution when engram ids empty")
	}
}
