package export

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func (service *ProjectTransferService) importEngrams(
	ctx context.Context,
	bundle ProjectExportBundle,
	request ImportProjectRequest,
) (map[uuid.UUID]uuid.UUID, engramImportCounts, error) {
	idMap := make(map[uuid.UUID]uuid.UUID, len(bundle.Engrams))
	counts := engramImportCounts{}
	for _, exported := range bundle.Engrams {
		targetID, action, err := service.importSingleEngram(ctx, request, exported)
		if err != nil {
			return nil, engramImportCounts{}, err
		}
		counts = addEngramImportCount(counts, action)
		idMap[exported.EngramID] = targetID
	}
	return idMap, counts, nil
}

func addEngramImportCount(counts engramImportCounts, action string) engramImportCounts {
	switch action {
	case "imported":
		counts.Imported++
	case "skipped":
		counts.Skipped++
	case "overwritten":
		counts.Imported++
		counts.Overwritten++
	}
	return counts
}

func (service *ProjectTransferService) importSingleEngram(
	ctx context.Context,
	request ImportProjectRequest,
	exported models.AdminEngramRecord,
) (uuid.UUID, string, error) {
	existingID, err := service.deps.findExistingEngramID(
		ctx,
		service.db,
		findExistingEngramIDInput{
			ProjectID: request.TargetProjectID,
			Title:     exported.Title,
			Markdown:  exported.DetailedSummaryMarkdown,
		},
	)
	if err != nil {
		return uuid.Nil, "", err
	}
	title, action, err := service.resolveImportedEngramTitle(ctx, request, exported.Title, existingID)
	if err != nil {
		return uuid.Nil, "", err
	}
	if action == "skipped" {
		if existingID == nil {
			return uuid.Nil, "", nil
		}
		return *existingID, action, nil
	}
	createdID, err := service.createImportedEngram(ctx, request, exported, title)
	if err != nil {
		return uuid.Nil, "", err
	}
	return createdID, action, nil
}

func (service *ProjectTransferService) resolveImportedEngramTitle(
	ctx context.Context,
	request ImportProjectRequest,
	baseTitle string,
	existingID *uuid.UUID,
) (string, string, error) {
	if existingID == nil {
		return baseTitle, "imported", nil
	}
	switch request.ConflictPolicy {
	case ProjectImportConflictPolicySkip:
		return "", "skipped", nil
	case ProjectImportConflictPolicyOverwrite:
		if err := service.deps.softDeleteEngram(ctx, service.db, *existingID, request.ActorUserID); err != nil {
			return "", "", err
		}
		return baseTitle, "overwritten", nil
	case ProjectImportConflictPolicyRename:
		title, err := service.deps.buildUniqueName(
			ctx,
			service.db,
			buildUniqueNameInput{
				Table:     "engrams",
				Column:    "title",
				ProjectID: request.TargetProjectID,
				BaseName:  baseTitle,
			},
		)
		if err != nil {
			return "", "", err
		}
		return title, "imported", nil
	default:
		return "", "", fmt.Errorf("unsupported conflict policy %q", request.ConflictPolicy)
	}
}

func (service *ProjectTransferService) createImportedEngram(
	ctx context.Context,
	request ImportProjectRequest,
	exported models.AdminEngramRecord,
	title string,
) (uuid.UUID, error) {
	ownerUserID := request.ActorUserID
	created, err := service.deps.createEngram(
		ctx,
		service.db,
		repository.CreateEngramInput{
			Payload:          buildImportedEngramCreatePayload(request.TargetProjectID, title, exported),
			EmbeddingDim:     service.embeddingDim,
			OwnerUserID:      &ownerUserID,
			EnrichmentOrigin: importEnrichmentOrigin,
		},
	)
	if err != nil {
		return uuid.Nil, err
	}
	if created == nil {
		return uuid.Nil, errors.New("engram creation returned nil response")
	}
	if err := service.deps.replaceEngramSources(
		ctx,
		service.db,
		created.EngramID,
		toSourceInputs(exported.Sources),
	); err != nil {
		return uuid.Nil, err
	}
	return created.EngramID, nil
}

func buildImportedEngramCreatePayload(
	targetProjectID string,
	title string,
	exported models.AdminEngramRecord,
) models.MemoryEngramCreate {
	return models.MemoryEngramCreate{
		ProjectID:               targetProjectID,
		ThreadID:                exported.ThreadID,
		Title:                   title,
		Abstract:                exported.Abstract,
		DetailedSummaryMarkdown: exported.DetailedSummaryMarkdown,
		Tags:                    copyStringSlice(exported.Tags),
		Keywords:                copyStringSlice(exported.Keywords),
		VisibilityScope:         normalizeVisibilityScope(exported.VisibilityScope),
		SourceSessionID:         exported.SourceSessionID,
	}
}

func normalizeVisibilityScope(value models.VisibilityScope) string {
	normalized, err := models.ParseVisibilityScope(strings.TrimSpace(string(value)))
	if err != nil {
		return string(models.VisibilityScopePrivate)
	}
	return string(normalized)
}
