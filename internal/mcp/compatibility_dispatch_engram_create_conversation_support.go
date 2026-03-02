package mcp

import (
	"context"
	"strings"

	"engram/internal/models"

	"github.com/google/uuid"
)

type engramCreateFromConversationParams struct {
	ProjectID            string     `json:"project_id"`
	ThreadID             *string    `json:"thread_id,omitempty"`
	Title                string     `json:"title"`
	Abstract             string     `json:"abstract"`
	ConversationMarkdown string     `json:"conversation_markdown"`
	Tags                 []string   `json:"tags"`
	Keywords             []string   `json:"keywords"`
	VisibilityScope      string     `json:"visibility_scope"`
	RetrievalText        *string    `json:"retrieval_text,omitempty"`
	SourceSessionID      *uuid.UUID `json:"source_session_id,omitempty"`
}

func (service *CompatibilityService) dispatchEngramCreateFromConversationTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCreateConversation == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramCreateFromConversationRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	request.EnrichmentOrigin = "mcp.engram.create_from_conversation"
	created, err := service.engramCreateConversation.CreateEngramFromConversation(ctx, request)
	if err != nil {
		return nil, true, mapEngramCreateError(err)
	}
	if created == nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{
		"engram":            created.Engram,
		"enrichment_report": created.EnrichmentReport,
	}, true, nil
}

func parseEngramCreateFromConversationRequest(
	actor Actor,
	params map[string]any,
) (EngramCreateFromConversationRequest, *toolDispatchError) {
	raw, ok := decodeEngramCreateFromConversationParams(params)
	if !ok {
		return EngramCreateFromConversationRequest{}, invalidParamError("payload")
	}
	normalizeEngramCreateFromConversationParams(&raw)
	if strings.TrimSpace(raw.Title) == "" {
		return EngramCreateFromConversationRequest{}, invalidParamError("title")
	}
	if strings.TrimSpace(raw.ConversationMarkdown) == "" {
		return EngramCreateFromConversationRequest{}, invalidParamError("conversation_markdown")
	}
	if _, err := models.ParseVisibilityScope(raw.VisibilityScope); err != nil {
		return EngramCreateFromConversationRequest{}, invalidParamError("visibility_scope")
	}
	return EngramCreateFromConversationRequest{
		ActorUserID:          actor.UserID,
		ActorRole:            normalizedActorRole(actor),
		ProjectID:            raw.ProjectID,
		ThreadID:             raw.ThreadID,
		Title:                raw.Title,
		Abstract:             raw.Abstract,
		ConversationMarkdown: raw.ConversationMarkdown,
		Tags:                 raw.Tags,
		Keywords:             raw.Keywords,
		VisibilityScope:      raw.VisibilityScope,
		RetrievalText:        raw.RetrievalText,
		SourceSessionID:      raw.SourceSessionID,
	}, nil
}

func decodeEngramCreateFromConversationParams(
	params map[string]any,
) (engramCreateFromConversationParams, bool) {
	parsed := engramCreateFromConversationParams{}
	if !decodeMapParams(params, &parsed) {
		return engramCreateFromConversationParams{}, false
	}
	return parsed, true
}

func normalizeEngramCreateFromConversationParams(
	params *engramCreateFromConversationParams,
) {
	params.ProjectID = strings.TrimSpace(params.ProjectID)
	params.Title = strings.TrimSpace(params.Title)
	params.Abstract = strings.TrimSpace(params.Abstract)
	params.ConversationMarkdown = strings.TrimSpace(params.ConversationMarkdown)
	params.VisibilityScope = normalizeEngramVisibilityScope(params.VisibilityScope)
	if params.Tags == nil {
		params.Tags = []string{}
	}
	if params.Keywords == nil {
		params.Keywords = []string{}
	}
}
