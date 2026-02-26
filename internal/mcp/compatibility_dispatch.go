package mcp

import (
	"context"
)

const (
	defaultProjectListLimit           = 500
	defaultProjectListOffset          = 0
	defaultUserProjectsLimit          = 1000
	defaultUserProjectsOffset         = 0
	defaultChatSessionsLimit          = 50
	defaultChatSessionsOffset         = 0
	defaultChatMessagesLimit          = 200
	defaultChatMessagesOffset         = 0
	defaultChatTimelineLimit          = 100
	defaultChatTimelineOffset         = 0
	defaultChatProjectDocumentsLimit  = 200
	defaultChatProjectDocumentsOffset = 0
	defaultEngramListLimit            = 200
	defaultEngramListOffset           = 0
	defaultCollectionListLimit        = 200
	defaultCollectionListOffset       = 0
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

var implementedToolHandlers = buildImplementedToolHandlers()

func buildImplementedToolHandlers() map[string]implementedToolHandler {
	handlers := map[string]implementedToolHandler{
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
	registerChatToolHandlers(handlers)
	registerEngramToolHandlers(handlers)
	return handlers
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
