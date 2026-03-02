package mcp

import (
	"context"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func newEngramLinkCompatibilityService(dependencies CompatibilityServiceDependencies) Service {
	return NewCompatibilityServiceWithDependencies("1.2.3", dependencies)
}

func sampleCompatibilityLinkRecord(
	linkID uuid.UUID,
	sourceEngramID uuid.UUID,
	targetEngramID uuid.UUID,
) models.EngramLinkRecord {
	createdAt := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)
	return models.EngramLinkRecord{
		LinkID:          linkID,
		ProjectID:       "proj-1",
		SourceEngramID:  sourceEngramID,
		TargetEngramID:  targetEngramID,
		RelationType:    models.EngramLinkRelationSupports,
		Weight:          0.8,
		TemporalWeight:  0.7,
		Confidence:      0.6,
		Origin:          models.EngramLinkOriginManual,
		Status:          models.EngramLinkStatusActive,
		EvidenceJSON:    map[string]any{},
		CreatedByUserID: uuid.MustParse("00000000-0000-0000-0000-000000007099"),
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
}

func engramLinkFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) models.EngramLinkRecord {
	return engramLinkFrameValue[models.EngramLinkRecord](t, frame, asToolsCallPath, "link")
}

func engramLinkFrameValue[T any](
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
	key string,
) T {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	value, ok := payload[key].(T)
	if !ok {
		t.Fatalf("expected %s payload", key)
	}
	return value
}

func assertEngramLinkCall[T any](t *testing.T, expected T, actual T) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected call %+v, got %+v", expected, actual)
	}
}

func TestFakeEngramLinkCallAssertionHelper(t *testing.T) {
	assertEngramLinkCall(t, 1, 1)
}

type fakeEngramLinkCreateService struct {
	created *models.EngramLinkRecord
	err     error
	call    EngramLinkCreateRequest
}

func (service *fakeEngramLinkCreateService) CreateEngramLink(
	_ context.Context,
	request EngramLinkCreateRequest,
) (*models.EngramLinkRecord, error) {
	service.call = request
	return copyEngramLinkOrError(service.created, service.err)
}

type fakeEngramLinkUpdateService struct {
	updated *models.EngramLinkRecord
	err     error
	call    EngramLinkUpdateRequest
}

func (service *fakeEngramLinkUpdateService) UpdateEngramLink(
	_ context.Context,
	request EngramLinkUpdateRequest,
) (*models.EngramLinkRecord, error) {
	service.call = request
	return copyEngramLinkOrError(service.updated, service.err)
}

type fakeEngramLinkArchiveService struct {
	archived *models.EngramLinkRecord
	err      error
	call     EngramLinkArchiveRequest
}

func (service *fakeEngramLinkArchiveService) ArchiveEngramLink(
	_ context.Context,
	request EngramLinkArchiveRequest,
) (*models.EngramLinkRecord, error) {
	service.call = request
	return copyEngramLinkOrError(service.archived, service.err)
}

type fakeEngramLinkGetService struct {
	link *models.EngramLinkRecord
	err  error
	call EngramLinkGetRequest
}

func (service *fakeEngramLinkGetService) GetEngramLink(
	_ context.Context,
	request EngramLinkGetRequest,
) (*models.EngramLinkRecord, error) {
	service.call = request
	return copyEngramLinkOrError(service.link, service.err)
}

func copyEngramLinkOrError(
	record *models.EngramLinkRecord,
	err error,
) (*models.EngramLinkRecord, error) {
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}
	copied := *record
	return &copied, nil
}
