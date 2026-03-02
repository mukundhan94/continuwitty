package export

import (
	"context"
	"errors"
	"time"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/projects"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const (
	exportCollectionListLimit = 5000
	exportEngramListLimit     = 50000
	importEnrichmentOrigin    = "export.import"
)

var (
	// ErrProjectNotFound indicates the actor cannot access the requested project.
	ErrProjectNotFound = errors.New("project not found")
	// ErrCollectionNotFoundForProject indicates one or more requested collection ids do not belong to the project.
	ErrCollectionNotFoundForProject = errors.New("collection not found for project")
	// ErrImportFileEmpty indicates import payload bytes were empty.
	ErrImportFileEmpty = errors.New("import file is empty")
	// ErrImportZipMissingExportJSON indicates ZIP payload omitted export.json.
	ErrImportZipMissingExportJSON = errors.New("zip missing export.json")
	// ErrUnsupportedImportFileFormat indicates payload bytes could not be decoded as UTF-8 JSON.
	ErrUnsupportedImportFileFormat = errors.New("unsupported import file format")
	// ErrInvalidExportBundle indicates payload JSON did not match the expected bundle shape.
	ErrInvalidExportBundle = errors.New("invalid export bundle")
	// ErrInvalidActorRole indicates the actor role in request metadata was not recognized.
	ErrInvalidActorRole = errors.New("invalid actor role")
)

type projectLookup interface {
	GetProject(
		ctx context.Context,
		actor projects.ActorContext,
		request projects.ProjectGetRequest,
	) (*models.ProjectRecord, error)
}

type memoryAdminLookup interface {
	ListCollections(
		ctx context.Context,
		request admin.MemoryAdminListRequest,
	) ([]models.EngramCollectionRecord, error)
	ListEngrams(
		ctx context.Context,
		request admin.MemoryAdminEngramListRequest,
	) ([]models.AdminEngramRecord, error)
	GetEngram(ctx context.Context, engramID uuid.UUID, includeDeleted bool) (*models.AdminEngramRecord, error)
	CreateCollection(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole string,
		payload admin.CollectionCreateRequest,
	) (*models.EngramCollectionRecord, error)
	AddCollectionItems(
		ctx context.Context,
		collectionID, actorUserID uuid.UUID,
		payload admin.CollectionItemsUpdateRequest,
	) (admin.CollectionItemsAddResponse, error)
}

type serviceDeps struct {
	nowUTC       func() time.Time
	createEngram func(
		ctx context.Context,
		db repository.Queryer,
		input repository.CreateEngramInput,
	) (*models.EngramCreateResponse, error)
	findExistingEngramID func(
		ctx context.Context,
		db repository.Queryer,
		input findExistingEngramIDInput,
	) (*uuid.UUID, error)
	softDeleteEngram func(
		ctx context.Context,
		db repository.Queryer,
		engramID, actorUserID uuid.UUID,
	) error
	findExistingCollectionID func(
		ctx context.Context,
		db repository.Queryer,
		projectID, name string,
	) (*uuid.UUID, error)
	buildUniqueName func(
		ctx context.Context,
		db repository.Queryer,
		input buildUniqueNameInput,
	) (string, error)
	listCollectionItemMap func(
		ctx context.Context,
		db repository.Queryer,
		collectionIDs []uuid.UUID,
	) (map[uuid.UUID][]uuid.UUID, error)
	listEngramSourcesMap func(
		ctx context.Context,
		db repository.Queryer,
		engramIDs []uuid.UUID,
	) (map[uuid.UUID][]models.AdminEngramSourceRecord, error)
	replaceEngramSources func(
		ctx context.Context,
		db repository.Queryer,
		engramID uuid.UUID,
		sources []models.AdminEngramSourceInput,
	) error
}

func defaultServiceDeps() serviceDeps {
	return serviceDeps{
		nowUTC:                   func() time.Time { return time.Now().UTC() },
		createEngram:             repository.CreateEngram,
		findExistingEngramID:     findExistingEngramID,
		softDeleteEngram:         softDeleteEngram,
		findExistingCollectionID: findExistingCollectionID,
		buildUniqueName:          buildUniqueName,
		listCollectionItemMap:    listCollectionItemMap,
		listEngramSourcesMap:     listEngramSourcesMap,
		replaceEngramSources:     replaceEngramSources,
	}
}

// ProjectTransferService implements project export/import orchestration.
type ProjectTransferService struct {
	db                 repository.Queryer
	projectService     projectLookup
	memoryAdminService memoryAdminLookup
	embeddingDim       int
	deps               serviceDeps
}

// NewService creates the export/import service.
func NewService(
	db repository.Queryer,
	projectService projectLookup,
	memoryAdminService memoryAdminLookup,
	embeddingDim int,
) *ProjectTransferService {
	return &ProjectTransferService{
		db:                 db,
		projectService:     projectService,
		memoryAdminService: memoryAdminService,
		embeddingDim:       embeddingDim,
		deps:               defaultServiceDeps(),
	}
}

type engramImportCounts struct {
	Imported    int
	Skipped     int
	Overwritten int
}

type collectionImportCounts struct {
	Imported int
	Reused   int
}
