package export

import (
	"context"
	"reflect"
	"testing"
	"time"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

type fakeProjectLookup struct {
	getProjectFn func(
		ctx context.Context,
		actor projects.ActorContext,
		request projects.ProjectGetRequest,
	) (*models.ProjectRecord, error)
}

func (fake *fakeProjectLookup) GetProject(
	ctx context.Context,
	actor projects.ActorContext,
	request projects.ProjectGetRequest,
) (*models.ProjectRecord, error) {
	if fake.getProjectFn == nil {
		return nil, nil
	}
	return fake.getProjectFn(ctx, actor, request)
}

type fakeMemoryAdminLookup struct {
	listCollectionsFn  func(context.Context, admin.MemoryAdminListRequest) ([]models.EngramCollectionRecord, error)
	listEngramsFn      func(context.Context, admin.MemoryAdminEngramListRequest) ([]models.AdminEngramRecord, error)
	getEngramFn        func(context.Context, uuid.UUID, bool) (*models.AdminEngramRecord, error)
	createCollectionFn func(
		context.Context,
		uuid.UUID,
		string,
		admin.CollectionCreateRequest,
	) (*models.EngramCollectionRecord, error)
	addCollectionItemsFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		admin.CollectionItemsUpdateRequest,
	) (admin.CollectionItemsAddResponse, error)
}

func (fake *fakeMemoryAdminLookup) ListCollections(
	ctx context.Context,
	request admin.MemoryAdminListRequest,
) ([]models.EngramCollectionRecord, error) {
	if fake.listCollectionsFn == nil {
		return []models.EngramCollectionRecord{}, nil
	}
	return fake.listCollectionsFn(ctx, request)
}

func (fake *fakeMemoryAdminLookup) ListEngrams(
	ctx context.Context,
	request admin.MemoryAdminEngramListRequest,
) ([]models.AdminEngramRecord, error) {
	if fake.listEngramsFn == nil {
		return []models.AdminEngramRecord{}, nil
	}
	return fake.listEngramsFn(ctx, request)
}

func (fake *fakeMemoryAdminLookup) GetEngram(
	ctx context.Context,
	engramID uuid.UUID,
	includeDeleted bool,
) (*models.AdminEngramRecord, error) {
	if fake.getEngramFn == nil {
		return nil, admin.ErrEngramNotFound
	}
	return fake.getEngramFn(ctx, engramID, includeDeleted)
}

func (fake *fakeMemoryAdminLookup) CreateCollection(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole string,
	payload admin.CollectionCreateRequest,
) (*models.EngramCollectionRecord, error) {
	if fake.createCollectionFn == nil {
		record := newCollectionRecord(uuid.New(), payload.ProjectID, payload.Name)
		return &record, nil
	}
	return fake.createCollectionFn(ctx, actorUserID, actorRole, payload)
}

func (fake *fakeMemoryAdminLookup) AddCollectionItems(
	ctx context.Context,
	collectionID uuid.UUID,
	actorUserID uuid.UUID,
	payload admin.CollectionItemsUpdateRequest,
) (admin.CollectionItemsAddResponse, error) {
	if fake.addCollectionItemsFn == nil {
		return admin.CollectionItemsAddResponse{Added: len(payload.EngramIDs)}, nil
	}
	return fake.addCollectionItemsFn(ctx, collectionID, actorUserID, payload)
}

func newTestExportService() (*ProjectTransferService, *fakeProjectLookup, *fakeMemoryAdminLookup) {
	projectService := &fakeProjectLookup{}
	memoryService := &fakeMemoryAdminLookup{}
	service := NewService(nil, projectService, memoryService, 1536)
	return service, projectService, memoryService
}

func newCollectionRecord(collectionID uuid.UUID, projectID, name string) models.EngramCollectionRecord {
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000d01")
	return models.EngramCollectionRecord{
		CollectionID: collectionID,
		ProjectID:    projectID,
		OwnerUserID:  ownerID,
		Name:         name,
		Description:  "",
		CreatedAt:    time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
	}
}

func newAdminEngramRecord(engramID uuid.UUID, projectID, title string) models.AdminEngramRecord {
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000d02")
	return models.AdminEngramRecord{
		EngramID:                engramID,
		ProjectID:               projectID,
		Title:                   title,
		Abstract:                "",
		DetailedSummaryMarkdown: "summary",
		Tags:                    []string{},
		Keywords:                []string{},
		OwnerUserID:             &ownerID,
		VisibilityScope:         models.VisibilityScopePrivate,
		CreatedAt:               time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
		UpdatedAt:               time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
		Sources:                 []models.AdminEngramSourceRecord{},
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func requireEqual[T any](t *testing.T, expected, actual T) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %#v, got %#v", expected, actual)
	}
}
