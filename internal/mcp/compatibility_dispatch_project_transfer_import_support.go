package mcp

import (
	"context"
	"encoding/json"
	"strings"
)

func (service *CompatibilityService) dispatchProjectImportBundleTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectImport == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseProjectImportRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	return runProjectTransferDispatch(
		func() (*ProjectImportResponse, error) {
			return service.projectImport.ImportProjectBundle(ctx, request)
		},
		func(response ProjectImportResponse) map[string]any {
			return map[string]any{"summary": response.Summary}
		},
	)
}

func parseProjectImportRequest(
	actor Actor,
	params map[string]any,
) (ProjectImportRequest, *toolDispatchError) {
	projectID, ok := requiredStringParam(params, "project_id")
	if !ok {
		return ProjectImportRequest{}, invalidParamError("project_id")
	}
	conflictPolicy, dispatchErr := parseProjectImportConflictPolicy(params)
	if dispatchErr != nil {
		return ProjectImportRequest{}, dispatchErr
	}
	bundleBytes, dispatchErr := parseProjectImportBundleBytes(params)
	if dispatchErr != nil {
		return ProjectImportRequest{}, dispatchErr
	}
	return ProjectImportRequest{
		ActorUserID:     actor.UserID,
		ActorRole:       normalizedActorRole(actor),
		TargetProjectID: projectID,
		BundleBytes:     bundleBytes,
		ConflictPolicy:  conflictPolicy,
	}, nil
}

func parseProjectImportConflictPolicy(params map[string]any) (string, *toolDispatchError) {
	value := strings.TrimSpace(
		strings.ToLower(
			stringParamWithDefault(params, "conflict_policy", defaultProjectImportConflictPolicy),
		),
	)
	switch value {
	case "skip", "overwrite", "rename":
		return value, nil
	default:
		return "", invalidParamError("conflict_policy")
	}
}

func parseProjectImportBundleBytes(params map[string]any) ([]byte, *toolDispatchError) {
	if bundleBytes, handled, dispatchErr := parseProjectImportBundleMap(params); handled {
		return bundleBytes, dispatchErr
	}
	if bundleJSONBytes, handled, dispatchErr := parseProjectImportBundleJSON(params); handled {
		return bundleJSONBytes, dispatchErr
	}
	return nil, invalidParamError("bundle")
}

func parseProjectImportBundleMap(params map[string]any) ([]byte, bool, *toolDispatchError) {
	rawBundle, found := optionalParamValue(params, "bundle")
	if !found {
		return nil, false, nil
	}
	bundleMap, ok := rawBundle.(map[string]any)
	if !ok {
		return nil, true, invalidParamError("bundle")
	}
	bundleBytes, err := json.Marshal(bundleMap)
	if err != nil {
		return nil, true, invalidParamError("bundle")
	}
	return bundleBytes, true, nil
}

func parseProjectImportBundleJSON(params map[string]any) ([]byte, bool, *toolDispatchError) {
	rawBundleJSON, found := optionalParamValue(params, "bundle_json")
	if !found {
		return nil, false, nil
	}
	bundleJSON, ok := rawBundleJSON.(string)
	if !ok {
		return nil, true, invalidParamError("bundle_json")
	}
	return []byte(bundleJSON), true, nil
}
