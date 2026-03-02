package export

import (
	"context"
	"errors"
	"testing"
	"time"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/projects"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestBuildProjectExportBundleCollectionFilter(t *testing.T) {
	service, projectService, memoryService := newTestExportService()
	now := time.Date(2026, 2, 26, 13, 0, 0, 0, time.UTC)
	projectID := "project-alpha"
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000c01")
	collectionAID := uuid.MustParse("00000000-0000-0000-0000-000000000c02")
	collectionBID := uuid.MustParse("00000000-0000-0000-0000-000000000c03")
	engramAID := uuid.MustParse("00000000-0000-0000-0000-000000000c04")
	engramBID := uuid.MustParse("00000000-0000-0000-0000-000000000c05")

	projectService.getProjectFn = func(
		_ context.Context,
		_ projects.ActorContext,
		request projects.ProjectGetRequest,
	) (*models.ProjectRecord, error) {
		return &models.ProjectRecord{
			ProjectID:   request.ProjectID,
			Name:        "Alpha",
			Description: "",
			OwnerUserID: actorID,
			IsArchived:  false,
			CreatedAt:   now,
			UpdatedAt:   now,
		}, nil
	}
	memoryService.listCollectionsFn = func(
		_ context.Context,
		_ admin.MemoryAdminListRequest,
	) ([]models.EngramCollectionRecord, error) {
		return []models.EngramCollectionRecord{
			newCollectionRecord(collectionAID, projectID, "Ops"),
			newCollectionRecord(collectionBID, projectID, "Research"),
		}, nil
	}
	memoryService.getEngramFn = func(
		_ context.Context,
		engramID uuid.UUID,
		_ bool,
	) (*models.AdminEngramRecord, error) {
		switch engramID {
		case engramAID:
			record := newAdminEngramRecord(engramAID, projectID, "A")
			return &record, nil
		case engramBID:
			record := newAdminEngramRecord(engramBID, projectID, "B")
			return &record, nil
		default:
			return nil, admin.ErrEngramNotFound
		}
	}
	service.deps.nowUTC = func() time.Time { return now }
	service.deps.listCollectionItemMap = func(
		_ context.Context,
		_ repository.Queryer,
		_ []uuid.UUID,
	) (map[uuid.UUID][]uuid.UUID, error) {
		return map[uuid.UUID][]uuid.UUID{
			collectionAID: {engramAID},
			collectionBID: {engramBID},
		}, nil
	}

	bundle, err := service.BuildProjectExportBundle(
		context.Background(),
		ExportProjectRequest{
			ActorUserID:       actorID,
			ActorRole:         string(models.UserRoleAnalyst),
			ProjectID:         projectID,
			CollectionIDs:     []uuid.UUID{collectionAID},
			IncludeEmbeddings: true,
		},
	)
	requireNoError(t, err)
	requireEqual(t, "1.0", bundle.SchemaVersion)
	requireEqual(t, now, bundle.ExportedAt)
	requireEqual(t, true, bundle.IncludeEmbeddings)
	requireEqual(t, []uuid.UUID{collectionAID}, bundle.SelectedCollectionIDs)
	requireEqual(t, 1, len(bundle.Collections))
	requireEqual(t, collectionAID, bundle.Collections[0].CollectionID)
	requireEqual(t, 1, len(bundle.CollectionItems))
	requireEqual(t, []uuid.UUID{engramAID}, bundle.CollectionItems[0].EngramIDs)
	requireEqual(t, 1, len(bundle.Engrams))
	requireEqual(t, engramAID, bundle.Engrams[0].EngramID)
}

func TestBuildProjectExportBundleWithoutCollectionFilterAttachesSources(t *testing.T) {
	service, projectService, memoryService := newTestExportService()
	now := time.Date(2026, 2, 26, 13, 10, 0, 0, time.UTC)
	projectID := "project-alpha"
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000c11")
	collectionID := uuid.MustParse("00000000-0000-0000-0000-000000000c12")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000c13")
	sourceID := uuid.MustParse("00000000-0000-0000-0000-000000000c14")

	projectService.getProjectFn = func(
		_ context.Context,
		_ projects.ActorContext,
		request projects.ProjectGetRequest,
	) (*models.ProjectRecord, error) {
		return &models.ProjectRecord{
			ProjectID:   request.ProjectID,
			Name:        "Alpha",
			Description: "",
			OwnerUserID: actorID,
			IsArchived:  false,
			CreatedAt:   now,
			UpdatedAt:   now,
		}, nil
	}
	memoryService.listCollectionsFn = func(
		_ context.Context,
		_ admin.MemoryAdminListRequest,
	) ([]models.EngramCollectionRecord, error) {
		return []models.EngramCollectionRecord{
			newCollectionRecord(collectionID, projectID, "Ops"),
		}, nil
	}
	memoryService.listEngramsFn = func(
		_ context.Context,
		_ admin.MemoryAdminEngramListRequest,
	) ([]models.AdminEngramRecord, error) {
		return []models.AdminEngramRecord{
			newAdminEngramRecord(engramID, projectID, "Sourceful"),
		}, nil
	}
	service.deps.nowUTC = func() time.Time { return now }
	service.deps.listCollectionItemMap = func(
		_ context.Context,
		_ repository.Queryer,
		_ []uuid.UUID,
	) (map[uuid.UUID][]uuid.UUID, error) {
		return map[uuid.UUID][]uuid.UUID{
			collectionID: {engramID},
		}, nil
	}
	service.deps.listEngramSourcesMap = func(
		_ context.Context,
		_ repository.Queryer,
		_ []uuid.UUID,
	) (map[uuid.UUID][]models.AdminEngramSourceRecord, error) {
		return map[uuid.UUID][]models.AdminEngramSourceRecord{
			engramID: {
				{
					SourceID:   sourceID,
					CapturedAt: now,
					URL:        "https://example.com/source",
				},
			},
		}, nil
	}

	bundle, err := service.BuildProjectExportBundle(
		context.Background(),
		ExportProjectRequest{
			ActorUserID: actorID,
			ActorRole:   string(models.UserRoleAnalyst),
			ProjectID:   projectID,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(bundle.Engrams))
	requireEqual(t, 1, len(bundle.Engrams[0].Sources))
	requireEqual(t, sourceID, bundle.Engrams[0].Sources[0].SourceID)
}

func TestBuildProjectExportBundleValidationErrors(t *testing.T) {
	t.Run("project not found", func(t *testing.T) {
		service, projectService, _ := newTestExportService()
			projectService.getProjectFn = func(
				_ context.Context,
				_ projects.ActorContext,
				_ projects.ProjectGetRequest,
			) (*models.ProjectRecord, error) {
				return nil, nil
			}

		_, err := service.BuildProjectExportBundle(
			context.Background(),
			ExportProjectRequest{
				ActorUserID: uuid.New(),
				ActorRole:   string(models.UserRoleAnalyst),
				ProjectID:   "missing",
			},
		)
		if !errors.Is(err, ErrProjectNotFound) {
			t.Fatalf("expected ErrProjectNotFound, got %v", err)
		}
	})

	t.Run("collection not found for project", func(t *testing.T) {
		service, projectService, memoryService := newTestExportService()
			projectService.getProjectFn = func(
				_ context.Context,
				_ projects.ActorContext,
				request projects.ProjectGetRequest,
			) (*models.ProjectRecord, error) {
				return &models.ProjectRecord{
					ProjectID:   request.ProjectID,
					OwnerUserID: uuid.MustParse("00000000-0000-0000-0000-000000000c21"),
				}, nil
			}
		memoryService.listCollectionsFn = func(
			_ context.Context,
			_ admin.MemoryAdminListRequest,
		) ([]models.EngramCollectionRecord, error) {
			return []models.EngramCollectionRecord{
				newCollectionRecord(
					uuid.MustParse("00000000-0000-0000-0000-000000000c22"),
					"project-alpha",
					"Ops",
				),
			}, nil
		}

		_, err := service.BuildProjectExportBundle(
			context.Background(),
			ExportProjectRequest{
				ActorUserID:   uuid.New(),
				ActorRole:     string(models.UserRoleAnalyst),
				ProjectID:     "project-alpha",
				CollectionIDs: []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000c23")},
			},
		)
		if !errors.Is(err, ErrCollectionNotFoundForProject) {
			t.Fatalf("expected ErrCollectionNotFoundForProject, got %v", err)
		}
	})
}
