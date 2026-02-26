package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"engram/internal/models"
	"engram/internal/projects"
)

func (service *CompatibilityService) dispatchEngramCreateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCreate == nil {
		return nil, false, nil
	}
	payload, dispatchErr := parseEngramCreatePayload(params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	created, err := service.engramCreate.CreateEngram(
		ctx,
		EngramCreateRequest{
			ActorUserID: actor.UserID,
			ActorRole:   normalizedActorRole(actor),
			Payload:     payload,
		},
	)
	if err != nil {
		return nil, true, mapEngramCreateError(err)
	}
	if created == nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"engram": *created}, true, nil
}

func parseEngramCreatePayload(params map[string]any) (models.MemoryEngramCreate, *toolDispatchError) {
	payload, ok := decodeEngramCreatePayload(params)
	if !ok {
		return models.MemoryEngramCreate{}, invalidParamError("payload")
	}
	normalizeEngramCreatePayload(&payload)
	if strings.TrimSpace(payload.Title) == "" {
		return models.MemoryEngramCreate{}, invalidParamError("title")
	}
	if strings.TrimSpace(payload.DetailedSummaryMarkdown) == "" {
		return models.MemoryEngramCreate{}, invalidParamError("detailed_summary_markdown")
	}
	if _, err := models.ParseVisibilityScope(payload.VisibilityScope); err != nil {
		return models.MemoryEngramCreate{}, invalidParamError("visibility_scope")
	}
	return payload, nil
}

func decodeEngramCreatePayload(params map[string]any) (models.MemoryEngramCreate, bool) {
	raw, err := json.Marshal(params)
	if err != nil {
		return models.MemoryEngramCreate{}, false
	}
	payload := models.MemoryEngramCreate{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return models.MemoryEngramCreate{}, false
	}
	return payload, true
}

func normalizeEngramCreatePayload(payload *models.MemoryEngramCreate) {
	payload.ProjectID = strings.TrimSpace(payload.ProjectID)
	payload.Title = strings.TrimSpace(payload.Title)
	payload.Abstract = strings.TrimSpace(payload.Abstract)
	payload.DetailedSummaryMarkdown = strings.TrimSpace(payload.DetailedSummaryMarkdown)
	payload.VisibilityScope = normalizeEngramVisibilityScope(payload.VisibilityScope)
	if payload.Decisions == nil {
		payload.Decisions = []models.Decision{}
	}
	if payload.Assumptions == nil {
		payload.Assumptions = []string{}
	}
	if payload.OpenQuestions == nil {
		payload.OpenQuestions = []string{}
	}
	if payload.Claims == nil {
		payload.Claims = []models.Claim{}
	}
	if payload.Tags == nil {
		payload.Tags = []string{}
	}
	if payload.Keywords == nil {
		payload.Keywords = []string{}
	}
	if payload.Artifacts == nil {
		payload.Artifacts = []models.ArtifactIn{}
	}
}

func normalizeEngramVisibilityScope(scope string) string {
	trimmed := strings.TrimSpace(scope)
	if trimmed == "" {
		return string(models.VisibilityScopePrivate)
	}
	return trimmed
}

func mapEngramCreateError(err error) *toolDispatchError {
	switch {
	case errors.Is(err, projects.ErrProjectIDMustNotBeBlank):
		return invalidParamError("project_id")
	case errors.Is(err, projects.ErrProjectNotFound):
		return invalidParamsWithStatus(404, "Project not found")
	case errors.Is(err, projects.ErrProjectIDRequiredWhenNoDefaultProject):
		return invalidParamsWithStatus(422, projects.ErrProjectIDRequiredWhenNoDefaultProject.Error())
	case errors.Is(err, projects.ErrDefaultProjectNotAccessible):
		return invalidParamsWithStatus(422, projects.ErrDefaultProjectNotAccessible.Error())
	default:
		return internalToolDispatchError()
	}
}
