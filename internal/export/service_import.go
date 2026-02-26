package export

import (
	"context"
	"errors"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/google/uuid"
)

// ImportProjectBundle imports a project bundle into a target project with conflict policy handling.
func (service *ProjectTransferService) ImportProjectBundle(
	ctx context.Context,
	request ImportProjectRequest,
) (ProjectImportResponse, error) {
	if _, err := service.resolveProjectOrError(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		request.TargetProjectID,
	); err != nil {
		return ProjectImportResponse{}, err
	}
	bundle, err := parseProjectExportBundle(request.FileBytes)
	if err != nil {
		return ProjectImportResponse{}, err
	}
	engramIDMap, engramCounts, err := service.importEngrams(ctx, bundle, request)
	if err != nil {
		return ProjectImportResponse{}, err
	}
	collectionIDMap, collectionCounts, err := service.importCollections(ctx, bundle, request)
	if err != nil {
		return ProjectImportResponse{}, err
	}
	importedCollectionItems, err := service.importCollectionItems(
		ctx,
		bundle.CollectionItems,
		request,
		engramIDMap,
		collectionIDMap,
	)
	if err != nil {
		return ProjectImportResponse{}, err
	}
	return ProjectImportResponse{
		TargetProjectID:         request.TargetProjectID,
		ImportedEngrams:         engramCounts.Imported,
		SkippedEngrams:          engramCounts.Skipped,
		OverwrittenEngrams:      engramCounts.Overwritten,
		ImportedCollections:     collectionCounts.Imported,
		ReusedCollections:       collectionCounts.Reused,
		ImportedCollectionItems: importedCollectionItems,
		ConflictPolicy:          request.ConflictPolicy,
	}, nil
}

func (service *ProjectTransferService) importCollections(
	ctx context.Context,
	bundle ProjectExportBundle,
	request ImportProjectRequest,
) (map[uuid.UUID]uuid.UUID, collectionImportCounts, error) {
	idMap := make(map[uuid.UUID]uuid.UUID, len(bundle.Collections))
	counts := collectionImportCounts{}
	for _, exported := range bundle.Collections {
		targetID, reused, err := service.importSingleCollection(ctx, request, exported)
		if err != nil {
			return nil, collectionImportCounts{}, err
		}
		if reused {
			counts.Reused++
		} else {
			counts.Imported++
		}
		idMap[exported.CollectionID] = targetID
	}
	return idMap, counts, nil
}

func (service *ProjectTransferService) importSingleCollection(
	ctx context.Context,
	request ImportProjectRequest,
	exported models.EngramCollectionRecord,
) (uuid.UUID, bool, error) {
	existingID, err := service.deps.findExistingCollectionID(
		ctx,
		service.db,
		request.TargetProjectID,
		exported.Name,
	)
	if err != nil {
		return uuid.Nil, false, err
	}
	if existingID != nil && request.ConflictPolicy != ProjectImportConflictPolicyRename {
		return *existingID, true, nil
	}
	name, err := service.resolveImportedCollectionName(ctx, request, exported.Name, existingID != nil)
	if err != nil {
		return uuid.Nil, false, err
	}
	created, err := service.memoryAdminService.CreateCollection(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		admin.CollectionCreateRequest{
			ProjectID:   request.TargetProjectID,
			Name:        name,
			Description: exported.Description,
		},
	)
	if err != nil {
		return uuid.Nil, false, err
	}
	if created == nil {
		return uuid.Nil, false, errors.New("collection creation returned nil response")
	}
	return created.CollectionID, false, nil
}

func (service *ProjectTransferService) resolveImportedCollectionName(
	ctx context.Context,
	request ImportProjectRequest,
	baseName string,
	hasExisting bool,
) (string, error) {
	if !hasExisting || request.ConflictPolicy != ProjectImportConflictPolicyRename {
		return baseName, nil
	}
	return service.deps.buildUniqueName(
		ctx,
		service.db,
		"engram_collections",
		"name",
		request.TargetProjectID,
		baseName,
	)
}

func (service *ProjectTransferService) importCollectionItems(
	ctx context.Context,
	itemRecords []ProjectExportCollectionItemsRecord,
	request ImportProjectRequest,
	engramIDMap map[uuid.UUID]uuid.UUID,
	collectionIDMap map[uuid.UUID]uuid.UUID,
) (int, error) {
	totalAdded := 0
	for _, itemRecord := range itemRecords {
		targetCollectionID, ok := collectionIDMap[itemRecord.CollectionID]
		if !ok {
			continue
		}
		mappedEngramIDs := mapCollectionItemEngramIDs(itemRecord.EngramIDs, engramIDMap)
		if len(mappedEngramIDs) == 0 {
			continue
		}
		result, err := service.memoryAdminService.AddCollectionItems(
			ctx,
			targetCollectionID,
			request.ActorUserID,
			admin.CollectionItemsUpdateRequest{EngramIDs: mappedEngramIDs},
		)
		if err != nil {
			return 0, err
		}
		totalAdded += result.Added
	}
	return totalAdded, nil
}

func mapCollectionItemEngramIDs(
	sourceIDs []uuid.UUID,
	engramIDMap map[uuid.UUID]uuid.UUID,
) []uuid.UUID {
	mapped := make([]uuid.UUID, 0, len(sourceIDs))
	for _, sourceID := range sourceIDs {
		targetID, ok := engramIDMap[sourceID]
		if !ok {
			continue
		}
		mapped = append(mapped, targetID)
	}
	return mapped
}
