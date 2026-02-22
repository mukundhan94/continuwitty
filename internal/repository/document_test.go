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
			{values: documentRecordRowValues(
				documentID,
				actorUserID,
				"project-docs",
				"Runbook",
				"text",
				&sourceName,
				&mimeType,
				"project",
				"hash-value",
				2,
				createdAt,
				createdAt,
			)},
			{values: []any{chunkA}},
			{values: []any{chunkB}},
		},
	}

	originalNow := nowDocumentUTC
	nowDocumentUTC = func() time.Time { return createdAt }
	t.Cleanup(func() { nowDocumentUTC = originalNow })

	originalEmbedMany := embedDocumentMany
	embedDocumentMany = func(texts []string, dim int) ([]embeddings.Result, error) {
		requireEqual(t, 2, len(texts))
		requireEqual(t, 8, dim)
		return []embeddings.Result{
			{ProviderID: "local-deterministic-v1", Vector: []float64{0.1, 0.2}},
			{ProviderID: "local-deterministic-v1", Vector: []float64{0.3, 0.4}},
		}, nil
	}
	t.Cleanup(func() { embedDocumentMany = originalEmbedMany })

	record, err := UpsertDocumentWithChunks(
		context.Background(),
		db,
		DocumentUpsertInput{
			ActorUserID:  actorUserID,
			EmbeddingDim: 8,
			Payload: DocumentUpsertPayload{
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
			},
		},
	)
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
			documentRecordRowValues(
				documentID,
				actorUserID,
				projectID,
				"Runbook",
				"text",
				nil,
				nil,
				"private",
				"hash-value",
				1,
				createdAt,
				createdAt,
			),
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

	originalEmbedText := embedDocumentText
	embedDocumentText = func(text string, dim int) (embeddings.Result, error) {
		requireEqual(t, "queue depth threshold", text)
		requireEqual(t, 16, dim)
		return embeddings.Result{ProviderID: "local-deterministic-v1", Vector: []float64{0.4, 0.6}}, nil
	}
	t.Cleanup(func() { embedDocumentText = originalEmbedText })

	results, err := QueryDocumentChunks(
		context.Background(),
		db,
		DocumentChunkQueryInput{
			ActorUserID:  actorUserID,
			EmbeddingDim: 16,
			Request: models.DocumentChunkQueryRequest{
				Query:       "queue depth threshold",
				ProjectID:   &projectID,
				DocumentIDs: []uuid.UUID{documentA, documentB},
				TopK:        1,
			},
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, chunkB, results[0].ChunkID)
	requireEqual(t, models.VisibilityScopeProject, results[0].VisibilityScope)

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
		[]uuid.UUID{documentA, documentB},
		4,
	}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
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

func documentRecordRowValues(
	documentID uuid.UUID,
	ownerUserID uuid.UUID,
	projectID string,
	title string,
	sourceType string,
	sourceName *string,
	mimeType *string,
	visibilityScope string,
	contentHash string,
	chunkCount int,
	createdAt time.Time,
	updatedAt time.Time,
) []any {
	var sourceNameValue any
	if sourceName != nil {
		sourceNameValue = *sourceName
	}
	var mimeTypeValue any
	if mimeType != nil {
		mimeTypeValue = *mimeType
	}
	return []any{
		documentID,
		ownerUserID,
		projectID,
		title,
		sourceType,
		sourceNameValue,
		mimeTypeValue,
		visibilityScope,
		contentHash,
		chunkCount,
		createdAt,
		updatedAt,
	}
}
