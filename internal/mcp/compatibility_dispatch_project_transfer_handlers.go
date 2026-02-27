package mcp

import "context"

func registerProjectTransferToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["project.export_bundle"] = func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchProjectExportBundleTool(ctx, actor, params)
	}
	handlers["project.import_bundle"] = func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchProjectImportBundleTool(ctx, actor, params)
	}
}
