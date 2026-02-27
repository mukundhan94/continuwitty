package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/projects"
	"engram/internal/repository"

	"github.com/google/uuid"
)

type mcpEngramCreateAdapter struct {
	db             repository.Queryer
	projectService *projects.Service
	embeddingDim   int
}

func newMCPEngramCreateAdapter(
	db repository.Queryer,
	projectService *projects.Service,
	embeddingDim int,
) mcp.EngramCreateService {
	return newMCPEngramCreateAdapterBase(db, projectService, embeddingDim)
}

func newMCPEngramCreateFromConversationAdapter(
	db repository.Queryer,
	projectService *projects.Service,
	embeddingDim int,
) mcp.EngramCreateFromConversationService {
	return newMCPEngramCreateAdapterBase(db, projectService, embeddingDim)
}

func newMCPEngramCreateAdapterBase(
	db repository.Queryer,
	projectService *projects.Service,
	embeddingDim int,
) *mcpEngramCreateAdapter {
	if db == nil || projectService == nil {
		return nil
	}
	return &mcpEngramCreateAdapter{
		db:             db,
		projectService: projectService,
		embeddingDim:   embeddingDim,
	}
}

func (adapter mcpEngramCreateAdapter) CreateEngram(
	ctx context.Context,
	request mcp.EngramCreateRequest,
) (*models.EngramCreateResponse, error) {
	created, _, err := adapter.createEngramWithProjectResolution(
		ctx,
		createEngramOperation{
			ActorUserID:      request.ActorUserID,
			ActorRole:        request.ActorRole,
			Payload:          request.Payload,
			EnrichmentOrigin: "mcp.engram.create",
		},
	)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (adapter mcpEngramCreateAdapter) CreateEngramFromConversation(
	ctx context.Context,
	request mcp.EngramCreateFromConversationRequest,
) (*mcp.EngramCreateFromConversationResponse, error) {
	enrichmentOrigin := request.EnrichmentOrigin
	if enrichmentOrigin == "" {
		enrichmentOrigin = "mcp.engram.create_from_conversation"
	}
	payload := models.MemoryEngramCreate{
		ProjectID:               request.ProjectID,
		ThreadID:                request.ThreadID,
		Title:                   request.Title,
		Abstract:                request.Abstract,
		DetailedSummaryMarkdown: request.ConversationMarkdown,
		Decisions:               []models.Decision{},
		Assumptions:             []string{},
		OpenQuestions:           []string{},
		Claims:                  []models.Claim{},
		Tags:                    append([]string(nil), request.Tags...),
		Keywords:                append([]string(nil), request.Keywords...),
		Artifacts:               []models.ArtifactIn{},
		RetrievalText:           request.RetrievalText,
		VisibilityScope:         request.VisibilityScope,
		SourceSessionID:         request.SourceSessionID,
	}
	created, enrichmentReport, err := adapter.createEngramWithProjectResolution(
		ctx,
		createEngramOperation{
			ActorUserID:      request.ActorUserID,
			ActorRole:        request.ActorRole,
			Payload:          payload,
			EnrichmentOrigin: enrichmentOrigin,
		},
	)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, nil
	}
	reportPayload := buildConversationEnrichmentReport(
		enrichmentReport,
		created.ResolvedProjectID,
		created.UsedDefaultProject,
	)
	return &mcp.EngramCreateFromConversationResponse{
		Engram:           *created,
		EnrichmentReport: reportPayload,
	}, nil
}

func (adapter mcpEngramCreateAdapter) createEngramWithProjectResolution(
	ctx context.Context,
	operation createEngramOperation,
) (*models.EngramCreateResponse, map[string]any, error) {
	resolution, err := adapter.projectService.ResolveProjectIDForWrite(
		ctx,
		operation.ActorUserID,
		operation.ActorRole,
		operation.Payload.ProjectID,
	)
	if err != nil {
		return nil, nil, err
	}
	payload := operation.Payload
	payload.ProjectID = resolution.ProjectID
	created, enrichmentReport, err := repository.CreateEngramWithReport(
		ctx,
		adapter.db,
		repository.CreateEngramInput{
			Payload:          payload,
			EmbeddingDim:     adapter.embeddingDim,
			OwnerUserID:      &operation.ActorUserID,
			EnrichmentOrigin: operation.EnrichmentOrigin,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if created == nil {
		return nil, nil, nil
	}
	resolvedProjectID := resolution.ProjectID
	created.ResolvedProjectID = &resolvedProjectID
	created.UsedDefaultProject = resolution.UsedDefaultProject
	return created, enrichmentReport, nil
}

type createEngramOperation struct {
	ActorUserID      uuid.UUID
	ActorRole        models.UserRole
	Payload          models.MemoryEngramCreate
	EnrichmentOrigin string
}

func buildConversationEnrichmentReport(
	report map[string]any,
	resolvedProjectID *string,
	usedDefaultProject bool,
) map[string]any {
	return map[string]any{
		"enrichment_applied": boolFromMapValue(report, "enrichment_applied"),
		"auto_tags":          stringSliceFromMapValue(report, "auto_tags"),
		"auto_keywords":      stringSliceFromMapValue(report, "auto_keywords"),
		"abstract_derived":   boolFromMapValue(report, "abstract_derived"),
		"resolved_project_id": func() string {
			if resolvedProjectID == nil {
				return ""
			}
			return *resolvedProjectID
		}(),
		"used_default_project": usedDefaultProject,
	}
}

func boolFromMapValue(source map[string]any, key string) bool {
	if source == nil {
		return false
	}
	value, exists := source[key]
	if !exists {
		return false
	}
	parsed, ok := value.(bool)
	if !ok {
		return false
	}
	return parsed
}

func stringSliceFromMapValue(source map[string]any, key string) []string {
	if source == nil {
		return []string{}
	}
	value, exists := source[key]
	if !exists {
		return []string{}
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		return stringSliceFromAny(typed)
	default:
		return []string{}
	}
}

func stringSliceFromAny(items []any) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			continue
		}
		result = append(result, text)
	}
	return result
}
