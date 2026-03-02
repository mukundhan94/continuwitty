package ingestion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestIngestTextRejectsOverlapNotSmallerThanChunkSize(t *testing.T) {
	service := NewService(nil, 256, 1024, 20000)
	_, err := service.IngestText(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-000000000901"),
		DocumentIngestTextRequest{
			ProjectID:         "engram-vault",
			Title:             "Bad chunk shape",
			Text:              "some text",
			ChunkSizeChars:    400,
			ChunkOverlapChars: 400,
		},
	)
	requireServiceError(t, err, 422, "chunk_overlap_chars must be smaller than chunk_size_chars")
}

func TestIngestFileValidationErrors(t *testing.T) {
	testCases := []struct {
		name           string
		fileRequest    FileIngestRequest
		expectedStatus int
		expectedDetail string
	}{
		{
			name: "unsupported type",
			fileRequest: FileIngestRequest{
				Filename:     "binary.bin",
				MimeType:     stringRef("application/octet-stream"),
				ContentBytes: []byte{0x80, 0x81, 0x82},
			},
			expectedStatus: 415,
			expectedDetail: "Unsupported file type. Upload a UTF-8 text-based document.",
		},
		{
			name: "empty file",
			fileRequest: FileIngestRequest{
				Filename:     "notes.txt",
				MimeType:     stringRef("text/plain"),
				ContentBytes: []byte{},
			},
			expectedStatus: 400,
			expectedDetail: "Uploaded file is empty",
		},
		{
			name: "oversized file",
			fileRequest: FileIngestRequest{
				Filename:     "big.txt",
				MimeType:     stringRef("text/plain"),
				ContentBytes: []byte(strings.Repeat("x", 1025)),
			},
			expectedStatus: 413,
			expectedDetail: "Uploaded file exceeds max allowed size of 1024 bytes",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			service := NewService(nil, 256, 1024, 20000)
			_, err := service.IngestFile(
				context.Background(),
				uuid.MustParse("00000000-0000-0000-0000-000000000902"),
				DocumentIngestFileRequest{ProjectID: "engram-vault", ChunkSizeChars: 1000, ChunkOverlapChars: 180},
				testCase.fileRequest,
			)
			requireServiceError(t, err, testCase.expectedStatus, testCase.expectedDetail)
		})
	}
}

func TestIngestTextPersistsChunkedDocument(t *testing.T) {
	service := NewService(nil, 256, 1024, 20000)
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000903")
	captured := repository.DocumentUpsertPayload{}
	service.deps.ensureProjectExists = func(_ context.Context, _ repository.Queryer, _ repository.ProjectEnsureInput) (*models.ProjectRecord, error) {
		return &models.ProjectRecord{ProjectID: "engram-vault", OwnerUserID: actorUserID}, nil
	}
	service.deps.upsertDocumentWithChunks = func(_ context.Context, _ repository.Queryer, input repository.DocumentUpsertInput) (*models.DocumentRecord, error) {
		captured = input.Payload
		now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
		return &models.DocumentRecord{
			DocumentID:      input.Payload.DocumentID,
			OwnerUserID:     actorUserID,
			ProjectID:       input.Payload.ProjectID,
			Title:           input.Payload.Title,
			SourceType:      input.Payload.SourceType,
			SourceName:      input.Payload.SourceName,
			MimeType:        input.Payload.MimeType,
			VisibilityScope: input.Payload.VisibilityScope,
			ContentHash:     input.Payload.ContentHash,
			ChunkCount:      len(input.Payload.Chunks),
			CreatedAt:       now,
			UpdatedAt:       now,
		}, nil
	}

	response, err := service.IngestText(
		context.Background(),
		actorUserID,
		DocumentIngestTextRequest{
			ProjectID:         "engram-vault",
			Title:             "Incident Notes",
			Text:              strings.Join(makeRepeatedWords("incident", 720), " "),
			ChunkSizeChars:    300,
			ChunkOverlapChars: 80,
		},
	)
	requireNoError(t, err)
	requireEqualString(t, "Incident Notes", response.Document.Title)
	if response.Document.ChunkCount < 2 {
		t.Fatalf("expected at least two chunks, got %d", response.Document.ChunkCount)
	}
	requireEqualString(t, "engram-vault", captured.ProjectID)
}

func TestIngestFilePersistsFileMetadataAndFallbackTitle(t *testing.T) {
	service := NewService(nil, 256, 4096, 20000)
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000904")
	captured := repository.DocumentUpsertPayload{}
	service.deps.ensureProjectExists = func(_ context.Context, _ repository.Queryer, _ repository.ProjectEnsureInput) (*models.ProjectRecord, error) {
		return &models.ProjectRecord{ProjectID: "engram-vault", OwnerUserID: actorUserID}, nil
	}
	service.deps.upsertDocumentWithChunks = func(_ context.Context, _ repository.Queryer, input repository.DocumentUpsertInput) (*models.DocumentRecord, error) {
		captured = input.Payload
		now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
		return &models.DocumentRecord{
			DocumentID:      input.Payload.DocumentID,
			OwnerUserID:     actorUserID,
			ProjectID:       input.Payload.ProjectID,
			Title:           input.Payload.Title,
			SourceType:      input.Payload.SourceType,
			SourceName:      input.Payload.SourceName,
			MimeType:        input.Payload.MimeType,
			VisibilityScope: input.Payload.VisibilityScope,
			ContentHash:     input.Payload.ContentHash,
			ChunkCount:      len(input.Payload.Chunks),
			CreatedAt:       now,
			UpdatedAt:       now,
		}, nil
	}

	content := "# On-call Notes\n\n" + strings.Join(makeRepeatedWords("queue-depth", 40), " ")
	mimeType := "text/markdown"
	fileBytes := []byte(content)
	response, err := service.IngestFile(
		context.Background(),
		actorUserID,
		DocumentIngestFileRequest{
			ProjectID:         "engram-vault",
			Metadata:          map[string]any{"domain": "incident"},
			ChunkSizeChars:    320,
			ChunkOverlapChars: 80,
		},
		FileIngestRequest{
			Filename:     "runbook.md",
			MimeType:     &mimeType,
			ContentBytes: fileBytes,
		},
	)
	requireNoError(t, err)
	requireEqualString(t, "runbook", response.Document.Title)
	if response.Document.ChunkCount < 1 {
		t.Fatalf("expected at least one chunk, got %d", response.Document.ChunkCount)
	}
	requireEqualString(t, "runbook.md", valueOrEmpty(captured.SourceName))
	requireEqualString(t, "text/markdown", valueOrEmpty(captured.MimeType))
	requireEqualString(t, "incident", captured.Metadata["domain"].(string))
	requireEqualString(t, "runbook.md", captured.Metadata["filename"].(string))
	requireEqualString(t, "text/markdown", captured.Metadata["mime_type"].(string))
	requireEqualInt(t, len(fileBytes), captured.Metadata["byte_size"].(int))
}

func makeRepeatedWords(word string, count int) []string {
	items := make([]string, 0, count)
	for index := 0; index < count; index++ {
		items = append(items, word)
	}
	return items
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func requireServiceError(t *testing.T, err error, expectedStatus int, expectedDetail string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected service error")
	}
	serviceErr := ServiceError{}
	if !errors.As(err, &serviceErr) {
		t.Fatalf("expected ServiceError, got %T", err)
	}
	requireEqualInt(t, expectedStatus, serviceErr.StatusCode)
	requireEqualString(t, expectedDetail, serviceErr.Detail)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func requireEqualString[T ~string](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

func requireEqualInt(t *testing.T, expected int, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}
