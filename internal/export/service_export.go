package export

import (
	"context"
	"strings"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

// BuildProjectExportBundle builds a project-scoped export bundle for JSON/ZIP route responses.
func (service *ProjectTransferService) BuildProjectExportBundle(
	ctx context.Context,
	request ExportProjectRequest,
) (ProjectExportBundle, error) {
	project, err := service.resolveProjectOrError(ctx, request.ActorUserID, request.ActorRole, request.ProjectID)
	if err != nil {
		return ProjectExportBundle{}, err
	}
	collections, err := service.listProjectCollections(ctx, project.ProjectID)
	if err != nil {
		return ProjectExportBundle{}, err
	}
	selectedCollections, selectedCollectionIDs, err := selectCollectionsForExport(collections, request.CollectionIDs)
	if err != nil {
		return ProjectExportBundle{}, err
	}
	collectionIDs := collectionIDsOf(selectedCollections)
	collectionItemMap, err := service.deps.listCollectionItemMap(ctx, service.db, collectionIDs)
	if err != nil {
		return ProjectExportBundle{}, err
	}
	engrams, err := service.listExportEngrams(
		ctx,
		project.ProjectID,
		selectedCollections,
		collectionItemMap,
		len(request.CollectionIDs) > 0,
	)
	if err != nil {
		return ProjectExportBundle{}, err
	}
	return ProjectExportBundle{
		SchemaVersion:         "1.0",
		ExportedAt:            service.deps.nowUTC(),
		Project:               *project,
		Collections:           cloneCollections(selectedCollections),
		CollectionItems:       buildCollectionItemRecords(selectedCollections, collectionItemMap),
		Engrams:               engrams,
		SelectedCollectionIDs: copyUUIDSlice(selectedCollectionIDs),
		IncludeEmbeddings:     request.IncludeEmbeddings,
	}, nil
}

func (service *ProjectTransferService) listExportEngrams(
	ctx context.Context,
	projectID string,
	selectedCollections []models.EngramCollectionRecord,
	collectionItemMap map[uuid.UUID][]uuid.UUID,
	filteredByCollection bool,
) ([]models.AdminEngramRecord, error) {
	if filteredByCollection {
		return service.listCollectionScopedEngrams(ctx, selectedCollections, collectionItemMap)
	}
	projectIDCopy := projectID
	records, err := service.memoryAdminService.ListEngrams(
		ctx,
		admin.MemoryAdminEngramListRequest{
			MemoryAdminListRequest: admin.MemoryAdminListRequest{
				ProjectID:      &projectIDCopy,
				OwnerUserID:    nil,
				IncludeDeleted: false,
				Limit:          exportEngramListLimit,
				Offset:         0,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return service.attachSourcesForEngrams(ctx, records)
}

func (service *ProjectTransferService) listCollectionScopedEngrams(
	ctx context.Context,
	selectedCollections []models.EngramCollectionRecord,
	collectionItemMap map[uuid.UUID][]uuid.UUID,
) ([]models.AdminEngramRecord, error) {
	engramIDs := flattenUniqueCollectionItemEngramIDs(selectedCollections, collectionItemMap)
	records := make([]models.AdminEngramRecord, 0, len(engramIDs))
	for _, engramID := range engramIDs {
		record, err := service.memoryAdminService.GetEngram(ctx, engramID, false)
		if err != nil {
			return nil, err
		}
		if record == nil {
			return nil, admin.ErrEngramNotFound
		}
		records = append(records, *record)
	}
	return records, nil
}

func (service *ProjectTransferService) attachSourcesForEngrams(
	ctx context.Context,
	engrams []models.AdminEngramRecord,
) ([]models.AdminEngramRecord, error) {
	engramIDs := make([]uuid.UUID, 0, len(engrams))
	for _, record := range engrams {
		engramIDs = append(engramIDs, record.EngramID)
	}
	sourceMap, err := service.deps.listEngramSourcesMap(ctx, service.db, engramIDs)
	if err != nil {
		return nil, err
	}
	result := make([]models.AdminEngramRecord, 0, len(engrams))
	for _, record := range engrams {
		updated := record
		updated.Sources = cloneSourceRecords(sourceMap[record.EngramID])
		result = append(result, updated)
	}
	return result, nil
}

func (service *ProjectTransferService) listProjectCollections(
	ctx context.Context,
	projectID string,
) ([]models.EngramCollectionRecord, error) {
	projectIDCopy := projectID
	return service.memoryAdminService.ListCollections(
		ctx,
		admin.MemoryAdminListRequest{
			ProjectID:      &projectIDCopy,
			OwnerUserID:    nil,
			IncludeDeleted: false,
			Limit:          exportCollectionListLimit,
			Offset:         0,
		},
	)
}

func (service *ProjectTransferService) resolveProjectOrError(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole string,
	projectID string,
) (*models.ProjectRecord, error) {
	role, err := models.ParseUserRole(strings.TrimSpace(actorRole))
	if err != nil {
		return nil, ErrInvalidActorRole
	}
	project, err := service.projectService.GetProject(
		ctx,
		projects.ActorContext{UserID: actorUserID, Role: role},
		projects.ProjectGetRequest{ProjectID: projectID, IncludeArchived: true},
	)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	return project, nil
}

func selectCollectionsForExport(
	collections []models.EngramCollectionRecord,
	requestedCollectionIDs []uuid.UUID,
) ([]models.EngramCollectionRecord, []uuid.UUID, error) {
	if len(requestedCollectionIDs) == 0 {
		selectedCollectionIDs := collectionIDsOf(collections)
		return cloneCollections(collections), selectedCollectionIDs, nil
	}
	byID := make(map[uuid.UUID]models.EngramCollectionRecord, len(collections))
	for _, collection := range collections {
		byID[collection.CollectionID] = collection
	}
	selected := make([]models.EngramCollectionRecord, 0, len(requestedCollectionIDs))
	for _, collectionID := range requestedCollectionIDs {
		collection, ok := byID[collectionID]
		if !ok {
			return nil, nil, ErrCollectionNotFoundForProject
		}
		selected = append(selected, collection)
	}
	return selected, copyUUIDSlice(requestedCollectionIDs), nil
}

func buildCollectionItemRecords(
	collections []models.EngramCollectionRecord,
	collectionItemMap map[uuid.UUID][]uuid.UUID,
) []ProjectExportCollectionItemsRecord {
	records := make([]ProjectExportCollectionItemsRecord, 0, len(collections))
	for _, collection := range collections {
		records = append(
			records,
			ProjectExportCollectionItemsRecord{
				CollectionID: collection.CollectionID,
				EngramIDs:    copyUUIDSlice(collectionItemMap[collection.CollectionID]),
			},
		)
	}
	return records
}

func flattenUniqueCollectionItemEngramIDs(
	selectedCollections []models.EngramCollectionRecord,
	collectionItemMap map[uuid.UUID][]uuid.UUID,
) []uuid.UUID {
	result := make([]uuid.UUID, 0)
	seen := make(map[uuid.UUID]struct{})
	for _, collection := range selectedCollections {
		for _, engramID := range collectionItemMap[collection.CollectionID] {
			if _, ok := seen[engramID]; ok {
				continue
			}
			seen[engramID] = struct{}{}
			result = append(result, engramID)
		}
	}
	return result
}
