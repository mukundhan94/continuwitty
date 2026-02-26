package mcp

import (
	"context"

	"engram/internal/models"
)

func registerChatSessionToolHandlers(handlers map[string]implementedToolHandler) {
	registerForwardedSessionHandler(
		handlers,
		"chat.create_session",
		(*CompatibilityService).dispatchChatCreateSessionTool,
	)
	registerSessionPayloadHandler(
		handlers,
		"chat.get_session",
		func(session models.ChatSessionRecord) map[string]any {
			return map[string]any{"session": session}
		},
	)
	registerForwardedSessionHandler(
		handlers,
		"chat.list_sessions",
		(*CompatibilityService).dispatchChatListSessionsTool,
	)
	registerSessionPayloadHandler(
		handlers,
		"chat.get_lifecycle_policy",
		func(session models.ChatSessionRecord) map[string]any {
			return map[string]any{"lifecycle_policy": lifecyclePolicyPayload(session)}
		},
	)
	registerForwardedSessionHandler(
		handlers,
		"chat.update_lifecycle_policy",
		(*CompatibilityService).dispatchChatUpdateLifecyclePolicyTool,
	)
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

func registerForwardedSessionHandler(
	handlers map[string]implementedToolHandler,
	methodName string,
	dispatch func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError),
) {
	handlers[methodName] = func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return dispatch(service, ctx, actor, params)
	}
}

func registerSessionPayloadHandler(
	handlers map[string]implementedToolHandler,
	methodName string,
	buildPayload func(session models.ChatSessionRecord) map[string]any,
) {
	handlers[methodName] = func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchSessionPayloadTool(ctx, actor, params, buildPayload)
	}
}
