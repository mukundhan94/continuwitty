package mcp

import (
	"context"

	"engram/internal/models"
)

func registerChatSessionToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["chat.create_session"] = chatCreateSessionHandler()
	handlers["chat.get_session"] = chatGetSessionHandler()
	handlers["chat.list_sessions"] = chatListSessionsHandler()
	handlers["chat.get_lifecycle_policy"] = chatGetLifecyclePolicyHandler()
}

func chatCreateSessionHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchChatCreateSessionTool(ctx, actor, params)
	}
}

func chatGetSessionHandler() implementedToolHandler {
	return func(
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
	}
}

func chatListSessionsHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchChatListSessionsTool(ctx, actor, params)
	}
}

func chatGetLifecyclePolicyHandler() implementedToolHandler {
	return func(
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
	}
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
