package mcp

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func newChatProjectDocumentsCompatibilityService(projectDocumentService ProjectDocumentListService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{ProjectDocumentService: projectDocumentService},
	)
}

func newFakeProjectDocumentListService(actorUserID uuid.UUID) *fakeProjectDocumentListService {
	return &fakeProjectDocumentListService{
		documents: []models.DocumentRecord{
			{
				DocumentID:      uuid.MustParse("37000000-0000-0000-0000-000000000371"),
				OwnerUserID:     actorUserID,
				ProjectID:       "proj-alpha",
				Title:           "Design Notes",
				SourceType:      models.DocumentSourceTypeText,
				VisibilityScope: models.VisibilityScopeProject,
				ContentHash:     "hash-a",
				ChunkCount:      3,
				CreatedAt:       time.Unix(1700002100, 0).UTC(),
				UpdatedAt:       time.Unix(1700002200, 0).UTC(),
			},
			{
				DocumentID:      uuid.MustParse("37000000-0000-0000-0000-000000000372"),
				OwnerUserID:     actorUserID,
				ProjectID:       "proj-alpha",
				Title:           "PRD",
				SourceType:      models.DocumentSourceTypeFile,
				VisibilityScope: models.VisibilityScopeProject,
				ContentHash:     "hash-b",
				ChunkCount:      5,
				CreatedAt:       time.Unix(1700002300, 0).UTC(),
				UpdatedAt:       time.Unix(1700002400, 0).UTC(),
			},
		},
	}
}

func projectDocumentsFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) []models.DocumentRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	rawDocuments, ok := payload["documents"]
	if !ok {
		t.Fatalf("expected documents payload")
	}
	return toDocumentRecords(t, rawDocuments)
}

func toDocumentRecords(t *testing.T, value any) []models.DocumentRecord {
	t.Helper()
	switch typed := value.(type) {
	case []models.DocumentRecord:
		return typed
	case []any:
		records := make([]models.DocumentRecord, 0, len(typed))
		for _, item := range typed {
			record, ok := item.(models.DocumentRecord)
			if !ok {
				t.Fatalf("expected DocumentRecord item, got %T", item)
			}
			records = append(records, record)
		}
		return records
	default:
		t.Fatalf("expected []DocumentRecord payload, got %T", value)
		return nil
	}
}

func assertProjectDocumentListCall(
	t *testing.T,
	call projectDocumentListCall,
	expected projectDocumentListExpectation,
) {
	t.Helper()
	if call.actorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	assertProjectFilter(t, call.projectID, expected.projectID)
	if call.limit != expected.limit || call.offset != expected.offset {
		t.Fatalf("expected limit/offset %d/%d, got %d/%d", expected.limit, expected.offset, call.limit, call.offset)
	}
}

type projectDocumentListExpectation struct {
	actorUserID uuid.UUID
	projectID   *string
	limit       int
	offset      int
}

type projectDocumentListCall struct {
	actorUserID uuid.UUID
	projectID   *string
	limit       int
	offset      int
}

type fakeProjectDocumentListService struct {
	documents []models.DocumentRecord
	err       error
	call      projectDocumentListCall
}

func (service *fakeProjectDocumentListService) ListProjectDocuments(
	_ context.Context,
	request ProjectDocumentListRequest,
) ([]models.DocumentRecord, error) {
	service.call = projectDocumentListCall{
		actorUserID: request.ActorUserID,
		projectID:   request.ProjectID,
		limit:       request.Limit,
		offset:      request.Offset,
	}
	if service.err != nil {
		return nil, service.err
	}
	return append([]models.DocumentRecord(nil), service.documents...), nil
}
