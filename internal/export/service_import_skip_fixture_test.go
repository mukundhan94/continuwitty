package export

import (
	"context"
	"testing"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

type skipImportScenario struct {
	service               *ProjectTransferService
	actorID               uuid.UUID
	projectID             string
	existingEngramID      uuid.UUID
	capturedCollectionIDs *[]uuid.UUID
	fileBytes             []byte
}

func newSkipImportScenario(t *testing.T) skipImportScenario {
	service, projectService, memoryService := newTestExportService()
	projectID := "project-target"
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000c31")
	exportedEngramID := uuid.MustParse("00000000-0000-0000-0000-000000000c32")
	existingEngramID := uuid.MustParse("00000000-0000-0000-0000-000000000c33")
	exportedCollectionID := uuid.MustParse("00000000-0000-0000-0000-000000000c34")
	existingCollectionID := uuid.MustParse("00000000-0000-0000-0000-000000000c35")

	setVisibleProject(projectService, actorID)
	service.deps.findExistingEngramID = func(
		_ context.Context,
		_ repository.Queryer,
		_ findExistingEngramIDInput,
	) (*uuid.UUID, error) {
		return &existingEngramID, nil
	}
	service.deps.findExistingCollectionID = func(
		_ context.Context,
		_ repository.Queryer,
		_, _ string,
	) (*uuid.UUID, error) {
		return &existingCollectionID, nil
	}
	service.deps.createEngram = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.CreateEngramInput,
	) (*models.EngramCreateResponse, error) {
		t.Fatalf("createEngram should not be called for skip policy")
		return nil, nil
	}
	capturedCollectionIDs := []uuid.UUID{}
	memoryService.addCollectionItemsFn = func(
		_ context.Context,
		collectionID uuid.UUID,
		_ uuid.UUID,
		payload admin.CollectionItemsUpdateRequest,
	) (admin.CollectionItemsAddResponse, error) {
		requireEqual(t, existingCollectionID, collectionID)
		capturedCollectionIDs = payload.EngramIDs
		return admin.CollectionItemsAddResponse{Added: len(payload.EngramIDs)}, nil
	}

	bundle := ProjectExportBundle{
		SchemaVersion: "1.0",
		Project: models.ProjectRecord{
			ProjectID:   "project-source",
			OwnerUserID: actorID,
		},
		Engrams: []models.AdminEngramRecord{
			newAdminEngramRecord(exportedEngramID, "project-source", "Skip me"),
		},
		Collections: []models.EngramCollectionRecord{
			newCollectionRecord(exportedCollectionID, "project-source", "Collection"),
		},
		CollectionItems: []ProjectExportCollectionItemsRecord{{
			CollectionID: exportedCollectionID,
			EngramIDs:    []uuid.UUID{exportedEngramID},
		}},
	}
	fileBytes := marshalBundle(t, bundle)
	return skipImportScenario{
		service:               service,
		actorID:               actorID,
		projectID:             projectID,
		existingEngramID:      existingEngramID,
		capturedCollectionIDs: &capturedCollectionIDs,
		fileBytes:             fileBytes,
	}
}
