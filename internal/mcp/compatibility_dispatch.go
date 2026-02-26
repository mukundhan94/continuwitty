package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

const (
	defaultProjectListLimit   = 500
	defaultProjectListOffset  = 0
	defaultUserProjectsLimit  = 1000
	defaultUserProjectsOffset = 0
	defaultChatSessionsLimit  = 50
	defaultChatSessionsOffset = 0
	defaultChatMessagesLimit  = 200
	defaultChatMessagesOffset = 0
	defaultChatTimelineLimit  = 100
	defaultChatTimelineOffset = 0
)

type toolDispatchError struct {
	code    int
	message string
	data    map[string]any
}

type implementedToolHandler func(
	service *CompatibilityService,
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError)

var implementedToolHandlers = map[string]implementedToolHandler{
	"chat.get_session": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchSessionPayloadTool(
			ctx,
			actor,
			params,
			func(session models.ChatSessionRecord) map[string]any {
				return map[string]any{"session": session}
			},
		)
	},
	"chat.list_sessions": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchChatListSessionsTool(ctx, actor, params)
	},
	"chat.get_lifecycle_policy": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchSessionPayloadTool(
			ctx,
			actor,
			params,
			func(session models.ChatSessionRecord) map[string]any {
				return map[string]any{"lifecycle_policy": lifecyclePolicyPayload(session)}
			},
		)
	},
	"chat.list_messages": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		if service.messageService == nil {
			return nil, false, nil
		}
		return service.dispatchSessionCollectionTool(
			ctx,
			actor,
			params,
			defaultChatMessagesLimit,
			defaultChatMessagesOffset,
			"messages",
			func(
				ctx context.Context,
				actorUserID uuid.UUID,
				sessionID uuid.UUID,
				paging pagingParams,
			) (any, error) {
				return service.messageService.ListMessages(
					ctx,
					MessageListRequest{
						ActorUserID: actorUserID,
						SessionID:   sessionID,
						Limit:       paging.limit,
						Offset:      paging.offset,
					},
				)
			},
		)
	},
	"chat.list_timeline": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		if service.timelineService == nil {
			return nil, false, nil
		}
		return service.dispatchSessionCollectionTool(
			ctx,
			actor,
			params,
			defaultChatTimelineLimit,
			defaultChatTimelineOffset,
			"events",
			func(
				ctx context.Context,
				actorUserID uuid.UUID,
				sessionID uuid.UUID,
				paging pagingParams,
			) (any, error) {
				return service.timelineService.ListTimeline(
					ctx,
					TimelineListRequest{
						ActorUserID: actorUserID,
						SessionID:   sessionID,
						Limit:       paging.limit,
						Offset:      paging.offset,
					},
				)
			},
		)
	},
	"chat.list_pinned_engrams": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		if service.pinnedEngramService == nil {
			return nil, false, nil
		}
		return service.dispatchSessionScopedCollectionTool(
			ctx,
			actor,
			params,
			"pinned_engrams",
			func(
				ctx context.Context,
				actorUserID uuid.UUID,
				sessionID uuid.UUID,
			) (any, error) {
				return service.pinnedEngramService.ListPinnedEngrams(
					ctx,
					SessionScopedRequest{
						ActorUserID: actorUserID,
						SessionID:   sessionID,
					},
				)
			},
		)
	},
	"chat.list_pinned_documents": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		if service.pinnedDocumentService == nil {
			return nil, false, nil
		}
		return service.dispatchSessionScopedCollectionTool(
			ctx,
			actor,
			params,
			"pinned_documents",
			func(
				ctx context.Context,
				actorUserID uuid.UUID,
				sessionID uuid.UUID,
			) (any, error) {
				return service.pinnedDocumentService.ListPinnedDocuments(
					ctx,
					SessionScopedRequest{
						ActorUserID: actorUserID,
						SessionID:   sessionID,
					},
				)
			},
		)
	},
	"user.get_profile": func(
		_ *CompatibilityService,
		_ context.Context,
		actor Actor,
		_ map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return map[string]any{"profile": actorPayload(actor)}, true, nil
	},
	"user.list_projects": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		_ map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchUserListProjectsTool(ctx, actor)
	},
	"project.list": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchProjectListTool(ctx, actor, params)
	},
	"project.create": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchProjectCreateTool(ctx, actor, params)
	},
	"project.get_default": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		_ map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchProjectGetDefaultTool(ctx, actor)
	},
	"project.set_default": func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchProjectSetDefaultTool(ctx, actor, params)
	},
}

func (service *CompatibilityService) dispatchImplementedTool(
	ctx context.Context,
	method string,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	handler, ok := implementedToolHandlers[method]
	if !ok {
		return nil, false, nil
	}
	return handler(service, ctx, actor, params)
}

func (service *CompatibilityService) lookupChatSession(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (*models.ChatSessionRecord, bool, *toolDispatchError) {
	if service.sessionGet == nil {
		return nil, false, nil
	}
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	session, err := service.sessionGet.GetSession(
		ctx,
		SessionGetRequest{
			ActorUserID: actor.UserID,
			SessionID:   sessionID,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if session == nil {
		return nil, true, invalidParamsWithStatus(404, "Chat session not found")
	}
	return session, true, nil
}

func (service *CompatibilityService) dispatchSessionPayloadTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
	buildPayload func(models.ChatSessionRecord) map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	session, handled, dispatchErr := service.lookupChatSession(ctx, actor, params)
	if dispatchErr != nil || !handled {
		return nil, handled, dispatchErr
	}
	return buildPayload(*session), true, nil
}

func (service *CompatibilityService) dispatchSessionCollectionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
	defaultLimit int,
	defaultOffset int,
	collectionKey string,
	collect func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		paging pagingParams,
	) (any, error),
) (map[string]any, bool, *toolDispatchError) {
	session, handled, dispatchErr := service.lookupChatSession(ctx, actor, params)
	if dispatchErr != nil || !handled {
		return nil, handled, dispatchErr
	}
	paging, pagingErr := parsePagingParams(params, defaultLimit, defaultOffset)
	if pagingErr != nil {
		return nil, true, pagingErr
	}
	items, err := collect(ctx, actor.UserID, session.SessionID, paging)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{collectionKey: items}, true, nil
}

func (service *CompatibilityService) dispatchSessionScopedCollectionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
	collectionKey string,
	collect func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
	) (any, error),
) (map[string]any, bool, *toolDispatchError) {
	session, handled, dispatchErr := service.lookupChatSession(ctx, actor, params)
	if dispatchErr != nil || !handled {
		return nil, handled, dispatchErr
	}
	items, err := collect(ctx, actor.UserID, session.SessionID)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{collectionKey: items}, true, nil
}

func lifecyclePolicyPayload(session models.ChatSessionRecord) map[string]any {
	return map[string]any{
		"autosave_enabled":          session.AutosaveEnabled,
		"autosave_strategy":         session.AutosaveStrategy,
		"autosave_interval_minutes": session.AutosaveIntervalMinutes,
		"autosave_min_messages":     session.AutosaveMinMessages,
		"retention_days":            session.RetentionDays,
		"retention_max_snapshots":   session.RetentionMaxSnapshots,
	}
}

func (service *CompatibilityService) dispatchUserListProjectsTool(
	ctx context.Context,
	actor Actor,
) (map[string]any, bool, *toolDispatchError) {
	if service.sessionService == nil {
		return nil, false, nil
	}
	sessions, err := service.sessionService.ListSessions(
		ctx,
		SessionListRequest{
			ActorUserID: actor.UserID,
			ProjectID:   nil,
			Limit:       defaultUserProjectsLimit,
			Offset:      defaultUserProjectsOffset,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{
		"project_ids": uniqueSortedProjectIDs(sessions),
	}, true, nil
}

func (service *CompatibilityService) dispatchChatListSessionsTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.sessionService == nil {
		return nil, false, nil
	}
	paging, pagingErr := parsePagingParams(
		params,
		defaultChatSessionsLimit,
		defaultChatSessionsOffset,
	)
	if pagingErr != nil {
		return nil, true, pagingErr
	}
	sessions, err := service.sessionService.ListSessions(
		ctx,
		SessionListRequest{
			ActorUserID: actor.UserID,
			ProjectID:   optionalProjectIDParam(params, "project_id"),
			Limit:       paging.limit,
			Offset:      paging.offset,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"sessions": sessions}, true, nil
}

func (service *CompatibilityService) dispatchProjectListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	includeArchived, ok := optionalBoolParam(params, "include_archived", false)
	if !ok {
		return nil, true, invalidParamError("include_archived")
	}
	limit, ok := optionalIntParam(params, "limit", defaultProjectListLimit)
	if !ok {
		return nil, true, invalidParamError("limit")
	}
	offset, ok := optionalIntParam(params, "offset", defaultProjectListOffset)
	if !ok {
		return nil, true, invalidParamError("offset")
	}

	projects, err := service.projectService.ListProjects(
		ctx,
		actor.UserID,
		normalizedActorRole(actor),
		includeArchived,
		limit,
		offset,
	)
	if err != nil {
		return nil, true, &toolDispatchError{code: -32603, message: "Internal error"}
	}
	return map[string]any{"projects": projects}, true, nil
}

func (service *CompatibilityService) dispatchProjectCreateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	ownerUserID, ok := optionalUUIDParam(params, "owner_user_id")
	if !ok {
		return nil, true, invalidParamError("owner_user_id")
	}
	created, err := service.projectService.CreateProject(
		ctx,
		actor.UserID,
		normalizedActorRole(actor),
		projects.CreateProjectRequest{
			ProjectID:   stringParamWithDefault(params, "project_id", ""),
			Name:        stringParamWithDefault(params, "name", ""),
			Description: stringParamWithDefault(params, "description", ""),
			OwnerUserID: ownerUserID,
		},
	)
	if err != nil {
		return nil, true, mapProjectCreateError(err)
	}
	return map[string]any{"project": created}, true, nil
}

func (service *CompatibilityService) dispatchProjectGetDefaultTool(
	ctx context.Context,
	actor Actor,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	defaultProjectID, err := service.projectService.GetDefaultProjectID(ctx, actor.UserID)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"default_project_id": optionalString(defaultProjectID)}, true, nil
}

func (service *CompatibilityService) dispatchProjectSetDefaultTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	projectID, ok := requiredStringParam(params, "project_id")
	if !ok {
		return nil, true, invalidParamError("project_id")
	}
	defaultProjectID, err := service.projectService.SetDefaultProjectID(
		ctx,
		actor.UserID,
		normalizedActorRole(actor),
		projectID,
	)
	if err != nil {
		return nil, true, mapProjectServiceError(err)
	}
	return map[string]any{"default_project_id": defaultProjectID}, true, nil
}

func invalidParamError(field string) *toolDispatchError {
	return &toolDispatchError{
		code:    -32602,
		message: "Invalid params",
		data:    map[string]any{"invalid": field},
	}
}

func internalToolDispatchError() *toolDispatchError {
	return &toolDispatchError{
		code:    -32603,
		message: "Internal error",
	}
}

func mapProjectServiceError(err error) *toolDispatchError {
	switch {
	case errors.Is(err, projects.ErrProjectIDMustNotBeBlank):
		return invalidParamError("project_id")
	case errors.Is(err, projects.ErrProjectNotFound):
		return invalidParamsWithStatus(404, "Project not found")
	case errors.Is(err, projects.ErrUserNotFound):
		return invalidParamsWithStatus(404, "User not found")
	default:
		return internalToolDispatchError()
	}
}

func mapProjectCreateError(err error) *toolDispatchError {
	switch {
	case errors.Is(err, projects.ErrProjectIDMustNotBeBlank):
		return invalidParamError("project_id")
	default:
		return internalToolDispatchError()
	}
}

func invalidParamsWithStatus(statusCode int, detail string) *toolDispatchError {
	return &toolDispatchError{
		code:    -32602,
		message: "Invalid params",
		data: map[string]any{
			"status_code": statusCode,
			"detail":      detail,
		},
	}
}

func normalizedActorRole(actor Actor) models.UserRole {
	return models.UserRole(strings.ToLower(strings.TrimSpace(actor.Role)))
}

func optionalString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func uniqueSortedProjectIDs(sessions []models.ChatSessionRecord) []string {
	seen := map[string]struct{}{}
	for _, session := range sessions {
		projectID := strings.TrimSpace(session.ProjectID)
		if projectID == "" {
			continue
		}
		seen[projectID] = struct{}{}
	}
	projectIDs := make([]string, 0, len(seen))
	for projectID := range seen {
		projectIDs = append(projectIDs, projectID)
	}
	sort.Strings(projectIDs)
	return projectIDs
}

func requiredStringParam(params map[string]any, key string) (string, bool) {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return "", false
	}
	value, ok := rawValue.(string)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != ""
}

func requiredUUIDParam(params map[string]any, key string) (uuid.UUID, bool) {
	value, ok := optionalUUIDParam(params, key)
	if !ok || value == nil {
		return uuid.Nil, false
	}
	return *value, true
}

func optionalProjectIDParam(params map[string]any, key string) *string {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return nil
	}
	text := strings.TrimSpace(fmt.Sprint(rawValue))
	if text == "" {
		return nil
	}
	return &text
}

func stringParamWithDefault(params map[string]any, key string, defaultValue string) string {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return defaultValue
	}
	return fmt.Sprint(rawValue)
}

func optionalUUIDParam(params map[string]any, key string) (*uuid.UUID, bool) {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return nil, true
	}
	candidate := strings.TrimSpace(fmt.Sprint(rawValue))
	if candidate == "" {
		return nil, false
	}
	parsed, err := uuid.Parse(candidate)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func optionalBoolParam(params map[string]any, key string, defaultValue bool) (bool, bool) {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return defaultValue, true
	}
	switch value := rawValue.(type) {
	case bool:
		return value, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

type pagingParams struct {
	limit  int
	offset int
}

func parsePagingParams(
	params map[string]any,
	defaultLimit int,
	defaultOffset int,
) (pagingParams, *toolDispatchError) {
	limit, ok := optionalIntParam(params, "limit", defaultLimit)
	if !ok {
		return pagingParams{}, invalidParamError("limit")
	}
	offset, ok := optionalIntParam(params, "offset", defaultOffset)
	if !ok {
		return pagingParams{}, invalidParamError("offset")
	}
	return pagingParams{limit: limit, offset: offset}, nil
}

func optionalIntParam(params map[string]any, key string, defaultValue int) (int, bool) {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return defaultValue, true
	}
	switch value := rawValue.(type) {
	case float64:
		intValue := int(value)
		if float64(intValue) != value {
			return 0, false
		}
		return intValue, true
	case json.Number:
		intValue, err := strconv.Atoi(value.String())
		if err != nil {
			return 0, false
		}
		return intValue, true
	default:
		return parseIntValue(fmt.Sprint(rawValue))
	}
}

func parseIntValue(raw string) (int, bool) {
	intValue, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, false
	}
	return intValue, true
}

func optionalParamValue(params map[string]any, key string) (any, bool) {
	if params == nil {
		return nil, false
	}
	rawValue, exists := params[key]
	if !exists || rawValue == nil {
		return nil, false
	}
	return rawValue, true
}

func actorPayload(actor Actor) map[string]any {
	payload := map[string]any{
		"user_id": actor.UserID,
		"role":    strings.ToLower(strings.TrimSpace(actor.Role)),
	}
	if username := strings.TrimSpace(actor.Username); username != "" {
		payload["username"] = username
	}
	return payload
}

func buildToolCallSuccessResult(toolName string, payload map[string]any) map[string]any {
	return map[string]any{
		"tool_name":         toolName,
		"structuredContent": payload,
		"content": []map[string]any{
			{"type": "text", "text": marshalPayloadText(payload)},
		},
		"isError": false,
	}
}

func marshalPayloadText(payload map[string]any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
