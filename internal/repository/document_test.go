package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/embeddings"
	"engram/internal/models"

	"github.com/google/uuid"
)

func TestUpsertDocumentWithChunksPersistsDocumentAndChunks(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000801")
	documentID := uuid.MustParse("00000000-0000-0000-0000-000000000802")
	chunkA := uuid.MustParse("00000000-0000-0000-0000-000000000803")
	chunkB := uuid.MustParse("00000000-0000-0000-0000-000000000804")
	createdAt := time.Date(2026, 2, 22, 13, 0, 0, 0, time.UTC)
	sourceName := "runbook.md"
	mimeType := "text/markdown"
	db := &fakeQueryer{
		queryRowResults: []*fakeRow{
			{values: documentRecordRowValues(documentRecordRowFixture{
				DocumentID:      documentID,
				OwnerUserID:     actorUserID,
				ProjectID:       "project-docs",
				Title:           "Runbook",
				SourceType:      "text",
				SourceName:      &sourceName,
				MimeType:        &mimeType,
				VisibilityScope: "project",
				ContentHash:     "hash-value",
				ChunkCount:      2,
				CreatedAt:       createdAt,
				UpdatedAt:       createdAt,
			})},
			{values: []any{chunkA}},
			{values: []any{chunkB}},
		},
	}
	setDocumentNow(t, createdAt)
	stubTwoChunkDocumentEmbeddings(t, 8)
	payload := buildDocumentUpsertPayloadForPersistenceTest(documentID, sourceName, mimeType, chunkA, chunkB)
	record, err := UpsertDocumentWithChunks(
		context.Background(),
		db,
		DocumentUpsertInput{
			ActorUserID:  actorUserID,
			EmbeddingDim: 8,
			Payload:      payload,
		},
	)
	assertDocumentUpsertPersistenceResult(t, err, record, db, documentID)
}

func TestUpsertDocumentWithChunksFailsOnEmbeddingCountMismatch(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000811")
	documentID := uuid.MustParse("00000000-0000-0000-0000-000000000812")
	chunkID := uuid.MustParse("00000000-0000-0000-0000-000000000813")
	db := &fakeQueryer{}

	originalEmbedMany := embedDocumentMany
	embedDocumentMany = func(_ []string, _ int) ([]embeddings.Result, error) {
		return []embeddings.Result{{ProviderID: "local", Vector: []float64{0.1}}}, nil
	}
	t.Cleanup(func() { embedDocumentMany = originalEmbedMany })

	record, err := UpsertDocumentWithChunks(
		context.Background(),
		db,
		DocumentUpsertInput{
			ActorUserID:  actorUserID,
			EmbeddingDim: 6,
			Payload: DocumentUpsertPayload{
				DocumentID:      documentID,
				ProjectID:       "project-docs",
				Title:           "Runbook",
				SourceType:      models.DocumentSourceTypeText,
				VisibilityScope: models.VisibilityScopePrivate,
				ContentText:     "hello",
				ContentHash:     "hash-value",
				Chunks: []DocumentChunkDraft{
					{ChunkID: chunkID, ChunkIndex: 0, ChunkText: "one"},
					{ChunkID: uuid.MustParse("00000000-0000-0000-0000-000000000814"), ChunkIndex: 1, ChunkText: "two"},
				},
			},
		},
	)
	if err == nil {
		t.Fatalf("expected embedding count mismatch error")
	}
	if !strings.Contains(err.Error(), "chunk embedding count mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil record on mismatch")
	}
	if len(db.queryRowSQL) != 0 || len(db.querySQL) != 0 {
		t.Fatalf("expected no db writes on mismatch")
	}
}

func TestListDocumentsAppliesProjectFilter(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000821")
	documentID := uuid.MustParse("00000000-0000-0000-0000-000000000822")
	createdAt := time.Date(2026, 2, 22, 13, 20, 0, 0, time.UTC)
	projectID := "project-docs"
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{
			documentRecordRowValues(documentRecordRowFixture{
				DocumentID:      documentID,
				OwnerUserID:     actorUserID,
				ProjectID:       projectID,
				Title:           "Runbook",
				SourceType:      "text",
				SourceName:      nil,
				MimeType:        nil,
				VisibilityScope: "private",
				ContentHash:     "hash-value",
				ChunkCount:      1,
				CreatedAt:       createdAt,
				UpdatedAt:       createdAt,
			}),
		}},
	}

	records, err := ListDocuments(
		context.Background(),
		db,
		DocumentListInput{
			ActorUserID: actorUserID,
			ProjectID:   &projectID,
			Limit:       20,
			Offset:      2,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, models.DocumentSourceTypeText, records[0].SourceType)

	query := db.querySQL[0]
	if !strings.Contains(query, "project_id = $2") {
		t.Fatalf("expected project filter clause, got %q", query)
	}
	expectedArgs := []any{actorUserID, projectID, 20, 2}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestQueryDocumentChunksBuildsQueryAndReranks(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000831")
	documentA := uuid.MustParse("00000000-0000-0000-0000-000000000832")
	documentB := uuid.MustParse("00000000-0000-0000-0000-000000000833")
	chunkA := uuid.MustParse("00000000-0000-0000-0000-000000000834")
	chunkB := uuid.MustParse("00000000-0000-0000-0000-000000000835")
	createdAt := time.Date(2026, 2, 22, 13, 30, 0, 0, time.UTC)
	projectID := "project-docs"
	visibility := "project"
	sourceName := "ops.md"
	documentIDs := []uuid.UUID{documentA, documentB}
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{
			{
				chunkA,
				documentA,
				projectID,
				"General notes",
				sourceName,
				0,
				"misc context",
				createdAt,
				visibility,
				"unrelated text",
				0.2,
			},
			{
				chunkB,
				documentB,
				projectID,
				"Queue depth runbook",
				sourceName,
				1,
				"queue depth exceeded threshold",
				createdAt,
				visibility,
				"queue depth exceeded threshold start dequeue worker burst",
				0.3,
			},
		}},
	}
	stubDocumentQueryEmbedding(t, "queue depth threshold", 16, []float64{0.4, 0.6})

	results, err := QueryDocumentChunks(
		context.Background(),
		db,
		DocumentChunkQueryInput{
			ActorUserID:  actorUserID,
			EmbeddingDim: 16,
			Request: models.DocumentChunkQueryRequest{
				Query:       "queue depth threshold",
				ProjectID:   &projectID,
				DocumentIDs: documentIDs,
				TopK:        1,
			},
		},
	)
	assertDocumentChunkQueryResult(t, err, results, chunkB)
	assertDocumentChunkQuerySQLAndArgs(t, db, actorUserID, projectID, documentIDs)
}

func TestBuildDocumentChunkWhereDefaultsToActorScopeOnly(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000841")
	whereSQL, params := buildDocumentChunkWhere(
		actorUserID,
		models.DocumentChunkQueryRequest{Query: "queue depth", TopK: 3},
	)
	requireEqual(t, "(d.owner_user_id = %s OR d.visibility_scope = 'project')", whereSQL)
	expectedParams := []any{actorUserID}
	if !reflect.DeepEqual(expectedParams, params) {
		t.Fatalf("expected params %#v, got %#v", expectedParams, params)
	}
}

func TestUpsertDocumentWithChunksRejectsInvalidVisibility(t *testing.T) {
	db := &fakeQueryer{}
	record, err := UpsertDocumentWithChunks(
		context.Background(),
		db,
		DocumentUpsertInput{
			ActorUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000851"),
			EmbeddingDim: 8,
			Payload: DocumentUpsertPayload{
				DocumentID:      uuid.MustParse("00000000-0000-0000-0000-000000000852"),
				ProjectID:       "project-docs",
				Title:           "Runbook",
				SourceType:      models.DocumentSourceTypeText,
				VisibilityScope: models.VisibilityScope("bad"),
				ContentText:     "text",
				ContentHash:     "hash",
			},
		},
	)
	if err == nil {
		t.Fatalf("expected invalid visibility error")
	}
	if !strings.Contains(err.Error(), "unsupported visibility scope") {
		t.Fatalf("expected visibility error, got %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil record on validation error")
	}
	if len(db.queryRowSQL) != 0 || len(db.querySQL) != 0 {
		t.Fatalf("expected no db calls on validation error")
	}
}

func setDocumentNow(t *testing.T, timestamp time.Time) {
	t.Helper()
	originalNow := nowDocumentUTC
	nowDocumentUTC = func() time.Time { return timestamp }
	t.Cleanup(func() { nowDocumentUTC = originalNow })
}

func stubTwoChunkDocumentEmbeddings(t *testing.T, expectedDim int) {
	t.Helper()
	originalEmbedMany := embedDocumentMany
	embedDocumentMany = func(texts []string, dim int) ([]embeddings.Result, error) {
		requireEqual(t, 2, len(texts))
		requireEqual(t, expectedDim, dim)
		return []embeddings.Result{
			{ProviderID: "local-deterministic-v1", Vector: []float64{0.1, 0.2}},
			{ProviderID: "local-deterministic-v1", Vector: []float64{0.3, 0.4}},
		}, nil
	}
	t.Cleanup(func() { embedDocumentMany = originalEmbedMany })
}

func buildDocumentUpsertPayloadForPersistenceTest(
	documentID uuid.UUID,
	sourceName string,
	mimeType string,
	chunkA uuid.UUID,
	chunkB uuid.UUID,
) DocumentUpsertPayload {
	return DocumentUpsertPayload{
		DocumentID:      documentID,
		ProjectID:       "project-docs",
		Title:           "Runbook",
		SourceType:      models.DocumentSourceTypeText,
		SourceName:      &sourceName,
		MimeType:        &mimeType,
		VisibilityScope: models.VisibilityScopeProject,
		ContentText:     "queue depth exceeded",
		ContentHash:     "hash-value",
		Metadata:        map[string]any{"owner": "ops"},
		Chunks: []DocumentChunkDraft{
			{
				ChunkID:       chunkA,
				ChunkIndex:    0,
				ChunkText:     "queue depth exceeded",
				Snippet:       "queue depth exceeded",
				CharStart:     0,
				CharEnd:       20,
				TokenEstimate: 3,
				Metadata:      map[string]any{"chunk": 0},
			},
			{
				ChunkID:       chunkB,
				ChunkIndex:    1,
				ChunkText:     "retry worker burst",
				Snippet:       "retry worker burst",
				CharStart:     21,
				CharEnd:       40,
				TokenEstimate: 3,
				Metadata:      map[string]any{"chunk": 1},
			},
		},
	}
}

func assertDocumentUpsertPersistenceResult(
	t *testing.T,
	err error,
	record *models.DocumentRecord,
	db *fakeQueryer,
	documentID uuid.UUID,
) {
	t.Helper()
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, documentID, record.DocumentID)
	requireEqual(t, models.DocumentSourceTypeText, record.SourceType)
	requireEqual(t, models.VisibilityScopeProject, record.VisibilityScope)
	requireEqual(t, 1, len(db.querySQL))
	if !strings.Contains(db.querySQL[0], "DELETE FROM document_chunks") {
		t.Fatalf("expected delete old chunks query, got %q", db.querySQL[0])
	}
	requireEqual(t, 3, len(db.queryRowArgs))
	if gotMetadata, ok := db.queryRowArgs[0][10].(string); !ok || !strings.Contains(gotMetadata, "\"owner\":\"ops\"") {
		t.Fatalf("expected document metadata json arg, got %#v", db.queryRowArgs[0][10])
	}
	if gotVector, ok := db.queryRowArgs[1][10].(string); !ok || gotVector != "[0.100000,0.200000]" {
		t.Fatalf("expected first chunk vector literal, got %#v", db.queryRowArgs[1][10])
	}
}

func stubDocumentQueryEmbedding(t *testing.T, expectedQuery string, expectedDim int, vector []float64) {
	t.Helper()
	originalEmbedText := embedDocumentText
	embedDocumentText = func(text string, dim int) (embeddings.Result, error) {
		requireEqual(t, expectedQuery, text)
		requireEqual(t, expectedDim, dim)
		return embeddings.Result{ProviderID: "local-deterministic-v1", Vector: vector}, nil
	}
	t.Cleanup(func() { embedDocumentText = originalEmbedText })
}

func assertDocumentChunkQueryResult(
	t *testing.T,
	err error,
	results []models.DocumentChunkQueryResult,
	expectedChunkID uuid.UUID,
) {
	t.Helper()
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, expectedChunkID, results[0].ChunkID)
	requireEqual(t, models.VisibilityScopeProject, results[0].VisibilityScope)
}

func assertDocumentChunkQuerySQLAndArgs(
	t *testing.T,
	db *fakeQueryer,
	actorUserID uuid.UUID,
	projectID string,
	documentIDs []uuid.UUID,
) {
	t.Helper()
	query := db.querySQL[0]
	if !strings.Contains(query, "d.project_id = $3") {
		t.Fatalf("expected project clause in query, got %q", query)
	}
	if !strings.Contains(query, "d.document_id = ANY($4::uuid[])") {
		t.Fatalf("expected document-id clause in query, got %q", query)
	}
	if !strings.Contains(query, "LIMIT $5") {
		t.Fatalf("expected candidate limit placeholder in query, got %q", query)
	}
	expectedArgs := []any{
		"[0.400000,0.600000]",
		actorUserID,
		projectID,
		documentIDs,
		4,
	}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

type documentRecordRowFixture struct {
	DocumentID      uuid.UUID
	OwnerUserID     uuid.UUID
	ProjectID       string
	Title           string
	SourceType      string
	SourceName      *string
	MimeType        *string
	VisibilityScope string
	ContentHash     string
	ChunkCount      int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func documentRecordRowValues(fixture documentRecordRowFixture) []any {
	var sourceNameValue any
	if fixture.SourceName != nil {
		sourceNameValue = *fixture.SourceName
	}
	var mimeTypeValue any
	if fixture.MimeType != nil {
		mimeTypeValue = *fixture.MimeType
	}
	return []any{
		fixture.DocumentID,
		fixture.OwnerUserID,
		fixture.ProjectID,
		fixture.Title,
		fixture.SourceType,
		sourceNameValue,
		mimeTypeValue,
		fixture.VisibilityScope,
		fixture.ContentHash,
		fixture.ChunkCount,
		fixture.CreatedAt,
		fixture.UpdatedAt,
	}
}
