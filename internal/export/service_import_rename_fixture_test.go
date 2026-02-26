package export

import (
	"context"
	"testing"
	"time"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

type renameImportScenario struct {
	service                 *ProjectTransferService
	actorID                 uuid.UUID
	projectID               string
	newEngramID             uuid.UUID
	createdPayload          *repository.CreateEngramInput
	capturedCollectionName  *string
	capturedCollectionItems *[]uuid.UUID
	capturedSourceInputs    *[]models.AdminEngramSourceInput
	fileBytes               []byte
}

type renameFixtureConfig struct {
	now                  time.Time
	projectID            string
	actorID              uuid.UUID
	sourceEngramID       uuid.UUID
	sourceCollectionID   uuid.UUID
	existingEngramID     uuid.UUID
	existingCollectionID uuid.UUID
	newEngramID          uuid.UUID
	newCollectionID      uuid.UUID
}

func newRenameImportScenario(t *testing.T) renameImportScenario {
	service, projectService, memoryService := newTestExportService()
	config := defaultRenameFixtureConfig()
	setVisibleProject(projectService, config.actorID)
	configureRenameLookupDeps(t, service, config)
	createdPayload := configureRenameCreateEngramDep(service, config)
	capturedSources := configureRenameSourceDep(t, service, config)
	capturedCollectionName := configureRenameCollectionCreateDep(memoryService, config)
	capturedCollectionItems := configureRenameCollectionItemsDep(t, memoryService, config)
	bundle := buildRenameFixtureBundle(config)
	fileBytes := marshalBundle(t, bundle)
	return renameImportScenario{
		service:                 service,
		actorID:                 config.actorID,
		projectID:               config.projectID,
		newEngramID:             config.newEngramID,
		createdPayload:          createdPayload,
		capturedCollectionName:  capturedCollectionName,
		capturedCollectionItems: capturedCollectionItems,
		capturedSourceInputs:    capturedSources,
		fileBytes:               fileBytes,
	}
}

func defaultRenameFixtureConfig() renameFixtureConfig {
	return renameFixtureConfig{
		now:                  time.Date(2026, 2, 26, 13, 20, 0, 0, time.UTC),
		projectID:            "project-target",
		actorID:              uuid.MustParse("00000000-0000-0000-0000-000000000c41"),
		sourceEngramID:       uuid.MustParse("00000000-0000-0000-0000-000000000c42"),
		sourceCollectionID:   uuid.MustParse("00000000-0000-0000-0000-000000000c43"),
		existingEngramID:     uuid.MustParse("00000000-0000-0000-0000-000000000c44"),
		existingCollectionID: uuid.MustParse("00000000-0000-0000-0000-000000000c45"),
		newEngramID:          uuid.MustParse("00000000-0000-0000-0000-000000000c46"),
		newCollectionID:      uuid.MustParse("00000000-0000-0000-0000-000000000c47"),
	}
}

func configureRenameLookupDeps(t *testing.T, service *ProjectTransferService, config renameFixtureConfig) {
	service.deps.findExistingEngramID = func(
		_ context.Context,
		_ repository.Queryer,
		_, _, _ string,
	) (*uuid.UUID, error) {
		return &config.existingEngramID, nil
	}
	service.deps.findExistingCollectionID = func(
		_ context.Context,
		_ repository.Queryer,
		_, _ string,
	) (*uuid.UUID, error) {
		return &config.existingCollectionID, nil
	}
	service.deps.buildUniqueName = func(
		_ context.Context,
		_ repository.Queryer,
		table, column, _, baseName string,
	) (string, error) {
		requireEqual(t, true, table == "engrams" || table == "engram_collections")
		requireEqual(t, true, column == "title" || column == "name")
		return baseName + " (imported)", nil
	}
}

func configureRenameCreateEngramDep(
	service *ProjectTransferService,
	config renameFixtureConfig,
) *repository.CreateEngramInput {
	captured := &repository.CreateEngramInput{}
	service.deps.createEngram = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.CreateEngramInput,
	) (*models.EngramCreateResponse, error) {
		*captured = input
		return &models.EngramCreateResponse{EngramID: config.newEngramID, CreatedAt: config.now}, nil
	}
	return captured
}

func configureRenameSourceDep(
	t *testing.T,
	service *ProjectTransferService,
	config renameFixtureConfig,
) *[]models.AdminEngramSourceInput {
	captured := &[]models.AdminEngramSourceInput{}
	service.deps.replaceEngramSources = func(
		_ context.Context,
		_ repository.Queryer,
		engramID uuid.UUID,
		sources []models.AdminEngramSourceInput,
	) error {
		requireEqual(t, config.newEngramID, engramID)
		*captured = sources
		return nil
	}
	return captured
}

func configureRenameCollectionCreateDep(
	memoryService *fakeMemoryAdminLookup,
	config renameFixtureConfig,
) *string {
	capturedName := new(string)
	memoryService.createCollectionFn = func(
		_ context.Context,
		_ uuid.UUID,
		_ string,
		payload admin.CollectionCreateRequest,
	) (*models.EngramCollectionRecord, error) {
		*capturedName = payload.Name
		created := newCollectionRecord(config.newCollectionID, config.projectID, payload.Name)
		return &created, nil
	}
	return capturedName
}

func configureRenameCollectionItemsDep(
	t *testing.T,
	memoryService *fakeMemoryAdminLookup,
	config renameFixtureConfig,
) *[]uuid.UUID {
	capturedItems := &[]uuid.UUID{}
	memoryService.addCollectionItemsFn = func(
		_ context.Context,
		collectionID uuid.UUID,
		_ uuid.UUID,
		payload admin.CollectionItemsUpdateRequest,
	) (admin.CollectionItemsAddResponse, error) {
		requireEqual(t, config.newCollectionID, collectionID)
		*capturedItems = payload.EngramIDs
		return admin.CollectionItemsAddResponse{Added: 1}, nil
	}
	return capturedItems
}

func buildRenameFixtureBundle(config renameFixtureConfig) ProjectExportBundle {
	return ProjectExportBundle{
		SchemaVersion: "1.0",
		Project: models.ProjectRecord{
			ProjectID:   "project-source",
			OwnerUserID: config.actorID,
		},
		Engrams: []models.AdminEngramRecord{{
			EngramID:                config.sourceEngramID,
			ProjectID:               "project-source",
			Title:                   "Collision",
			Abstract:                "",
			DetailedSummaryMarkdown: "Summary",
			Tags:                    []string{"alpha"},
			Keywords:                []string{"migration"},
			VisibilityScope:         models.VisibilityScopeProject,
			Sources: []models.AdminEngramSourceRecord{{
				SourceID:   uuid.MustParse("00000000-0000-0000-0000-000000000c48"),
				CapturedAt: config.now,
				URL:        "https://example.com/export-source",
			}},
		}},
		Collections: []models.EngramCollectionRecord{
			newCollectionRecord(config.sourceCollectionID, "project-source", "Collision Collection"),
		},
		CollectionItems: []ProjectExportCollectionItemsRecord{{
			CollectionID: config.sourceCollectionID,
			EngramIDs:    []uuid.UUID{config.sourceEngramID},
		}},
	}
}
