package mcp

import (
	"context"
	"strings"

	"engram/internal/models"

	"github.com/google/uuid"
)

var sessionScopedProjectTools = map[string]struct{}{
	"chat.get_session":             {},
	"chat.get_lifecycle_policy":    {},
	"chat.update_lifecycle_policy": {},
	"chat.list_messages":           {},
	"chat.list_timeline":           {},
	"chat.send_message":            {},
	"chat.list_pinned_engrams":     {},
	"chat.pin_engram":              {},
	"chat.unpin_engram":            {},
	"chat.list_pinned_documents":   {},
	"chat.pin_document":            {},
	"chat.unpin_document":          {},
	"chat.continue_session":        {},
	"chat.delete_session":          {},
	"chat.restore_session":         {},
}

var engramScopedProjectTools = map[string]struct{}{
	"engram.get":          {},
	"engram.update":       {},
	"engram.share":        {},
	"engram.unshare":      {},
	"engram.move_project": {},
	"engram.delete":       {},
	"engram.restore":      {},
	"engram.link_create":  {},
	"engram.link_list":    {},
	"engram.link_suggest": {},
	"engram.trace_path":   {},
	"engram.feedback":     {},
}

var linkScopedProjectTools = map[string]struct{}{
	"engram.link_update":  {},
	"engram.link_archive": {},
}

var collectionScopedProjectTools = map[string]struct{}{
	"engram.collection_update":       {},
	"engram.collection_delete":       {},
	"engram.collection_add_items":    {},
	"engram.collection_remove_items": {},
}

var projectInputTools = map[string]struct{}{
	"chat.create_session":             {},
	"engram.create":                   {},
	"engram.create_from_conversation": {},
	"project.create":                  {},
	"project.member_list":             {},
	"project.member_add":              {},
	"project.member_update":           {},
	"project.member_remove":           {},
	"project.export_bundle":           {},
	"project.import_bundle":           {},
	"project.set_default":             {},
	"engram.collection_create":        {},
}

type tokenProjectPolicyRequest struct {
	ctx       context.Context
	actor     Actor
	toolName  string
	params    map[string]any
	tokenAuth *models.MCPTokenAuthContext
}

func (service *CompatibilityService) enforceTokenProjectPolicy(input tokenProjectPolicyRequest) *toolPolicyError {
	if !hasTokenProjectAllowlist(input.tokenAuth) {
		return nil
	}
	canonicalTool := canonicalToolName(toolIdentifier(input.toolName))
	if policyError := service.enforceResolvedTokenProjectPolicy(
		input,
		canonicalTool.String(),
		service.resolveProjectIDForTokenPolicy,
	); policyError != nil {
		return policyError
	}
	return service.enforceResolvedTokenProjectPolicy(
		input,
		canonicalTool.String(),
		service.resolveSecondaryProjectIDForTokenPolicy,
	)
}

func hasTokenProjectAllowlist(tokenAuth *models.MCPTokenAuthContext) bool {
	return tokenAuth != nil && len(tokenAuth.AllowedProjectIDs) > 0
}

type tokenProjectResolver func(
	ctx context.Context,
	actor Actor,
	canonicalTool string,
	params map[string]any,
) (string, *toolPolicyError)

type scopedProjectResource string

const (
	scopedProjectResourceEngram     scopedProjectResource = "engram"
	scopedProjectResourceCollection scopedProjectResource = "collection"
	scopedProjectResourceLink       scopedProjectResource = "link"
)

type scopedProjectResolveRequest struct {
	ctx      context.Context
	actor    Actor
	params   map[string]any
	idField  string
	resource scopedProjectResource
}

type scopedProjectLookupKind string

const (
	scopedProjectLookupSession   scopedProjectLookupKind = "session"
	scopedProjectLookupRehydrate scopedProjectLookupKind = "rehydrate"
)

func (service *CompatibilityService) enforceResolvedTokenProjectPolicy(
	input tokenProjectPolicyRequest,
	canonicalTool string,
	resolver tokenProjectResolver,
) *toolPolicyError {
	projectID, policyError := resolver(
		input.ctx,
		input.actor,
		canonicalTool,
		input.params,
	)
	if policyError != nil {
		return policyError
	}
	if projectID == "" {
		return nil
	}
	return validateTokenProjectPolicy(
		input.toolName,
		canonicalTool,
		input.tokenAuth,
		projectID,
	)
}

func validateTokenProjectPolicy(
	toolName string,
	canonicalTool string,
	tokenAuth *models.MCPTokenAuthContext,
	resolvedValue string,
) *toolPolicyError {
	if projectAllowedByToken(resolvedValue, tokenAuth.AllowedProjectIDs) {
		return nil
	}
	return disallowedProjectPolicyError(toolName, canonicalTool, tokenAuth.Scope, resolvedValue)
}

func (service *CompatibilityService) resolveProjectIDForTokenPolicy(
	ctx context.Context,
	actor Actor,
	canonicalTool string,
	params map[string]any,
) (string, *toolPolicyError) {
	if projectID, resolved := resolveProjectByToolInput(canonicalTool, params); resolved {
		return projectID, nil
	}
	if lookupKind, scoped := lookupKindForTool(canonicalTool); scoped {
		return service.resolveProjectByLookup(ctx, actor.UserID, params, lookupKind)
	}
	if canonicalTool == "chat.save_as_engram" {
		return service.resolveSaveAsEngramProjectID(ctx, actor.UserID, params)
	}
	if _, engramScoped := engramScopedProjectTools[canonicalTool]; engramScoped {
		return service.resolveEngramProjectID(ctx, actor, canonicalTool, params)
	}
	if resourceRequest, scoped := buildScopedResourceRequestForTool(
		ctx,
		actor,
		canonicalTool,
		params,
	); scoped {
		return service.resolveScopedProjectByResource(resourceRequest)
	}
	return "", nil
}

func resolveProjectByToolInput(canonicalTool string, params map[string]any) (string, bool) {
	if _, projectInputTool := projectInputTools[canonicalTool]; projectInputTool {
		return normalizeProjectIDParam(params), true
	}
	if _, optionalProjectTool := optionalProjectTools[canonicalTool]; optionalProjectTool {
		return normalizeProjectIDParam(params), true
	}
	return "", false
}

func lookupKindForTool(canonicalTool string) (scopedProjectLookupKind, bool) {
	if _, sessionScoped := sessionScopedProjectTools[canonicalTool]; sessionScoped {
		return scopedProjectLookupSession, true
	}
	if canonicalTool == "engram.rehydrate" {
		return scopedProjectLookupRehydrate, true
	}
	return "", false
}

func buildScopedResourceRequestForTool(
	ctx context.Context,
	actor Actor,
	canonicalTool string,
	params map[string]any,
) (scopedProjectResolveRequest, bool) {
	if _, collectionScoped := collectionScopedProjectTools[canonicalTool]; collectionScoped {
		return scopedProjectResolveRequest{
			ctx:      ctx,
			actor:    actor,
			params:   params,
			idField:  "collection_id",
			resource: scopedProjectResourceCollection,
		}, true
	}
	if _, linkScoped := linkScopedProjectTools[canonicalTool]; linkScoped {
		return scopedProjectResolveRequest{
			ctx:      ctx,
			actor:    actor,
			params:   params,
			idField:  "link_id",
			resource: scopedProjectResourceLink,
		}, true
	}
	return scopedProjectResolveRequest{}, false
}

func (service *CompatibilityService) resolveSecondaryProjectIDForTokenPolicy(
	ctx context.Context,
	actor Actor,
	canonicalTool string,
	params map[string]any,
) (string, *toolPolicyError) {
	if canonicalTool != "engram.move_project" {
		return "", nil
	}
	if normalizeTargetProjectIDParam(params) == "" {
		return "", nil
	}
	return service.resolveScopedProjectByResource(
		scopedProjectResolveRequest{
			ctx:      ctx,
			actor:    actor,
			params:   params,
			idField:  "engram_id",
			resource: scopedProjectResourceEngram,
		},
	)
}

func (service *CompatibilityService) resolveEngramProjectID(
	ctx context.Context,
	actor Actor,
	canonicalTool string,
	params map[string]any,
) (string, *toolPolicyError) {
	if _, ok := requiredUUIDParam(params, "engram_id"); !ok {
		return "", invalidParamPolicyError("engram_id")
	}
	if canonicalTool == "engram.move_project" {
		if targetProjectID := normalizeTargetProjectIDParam(params); targetProjectID != "" {
			return targetProjectID, nil
		}
	}
	return service.resolveScopedProjectByResource(
		scopedProjectResolveRequest{
			ctx:      ctx,
			actor:    actor,
			params:   params,
			idField:  "engram_id",
			resource: scopedProjectResourceEngram,
		},
	)
}

func (service *CompatibilityService) resolveScopedProjectByResource(
	input scopedProjectResolveRequest,
) (string, *toolPolicyError) {
	return resolveScopedProjectID(
		input.params,
		input.idField,
		func(resourceID uuid.UUID) (string, error) {
			return service.lookupScopedProjectID(input.ctx, input.actor, input.resource, resourceID)
		},
	)
}

func (service *CompatibilityService) resolveSaveAsEngramProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
	params map[string]any,
) (string, *toolPolicyError) {
	if sessionIDValue, hasSessionID := optionalParamValue(params, "session_id"); hasSessionID {
		if strings.TrimSpace(stringParam(sessionIDValue)) != "" {
			return service.resolveProjectByLookup(ctx, actorUserID, params, scopedProjectLookupSession)
		}
	}
	return normalizeProjectIDParam(params), nil
}

func (service *CompatibilityService) resolveProjectByLookup(
	ctx context.Context,
	actorUserID uuid.UUID,
	params map[string]any,
	lookupKind scopedProjectLookupKind,
) (string, *toolPolicyError) {
	idField, lookup := service.resolveProjectLookupResolver(ctx, actorUserID, lookupKind)
	if idField == "" || lookup == nil {
		return "", nil
	}
	return resolveScopedProjectID(params, idField, lookup)
}

func (service *CompatibilityService) resolveProjectLookupResolver(
	ctx context.Context,
	actorUserID uuid.UUID,
	lookupKind scopedProjectLookupKind,
) (string, func(uuid.UUID) (string, error)) {
	switch lookupKind {
	case scopedProjectLookupSession:
		return "session_id", service.buildActorProjectLookup(ctx, actorUserID, scopedProjectLookupSession)
	case scopedProjectLookupRehydrate:
		return "engram_id", service.buildActorProjectLookup(ctx, actorUserID, scopedProjectLookupRehydrate)
	default:
		return "", nil
	}
}

func (service *CompatibilityService) buildActorProjectLookup(
	ctx context.Context,
	actorUserID uuid.UUID,
	lookupKind scopedProjectLookupKind,
) func(uuid.UUID) (string, error) {
	return func(resourceID uuid.UUID) (string, error) {
		switch lookupKind {
		case scopedProjectLookupSession:
			return runScopedProjectLookup(service.sessionGet != nil, func() (string, bool, error) {
				session, err := service.sessionGet.GetSession(
					ctx,
					SessionGetRequest{
						ActorUserID: actorUserID,
						SessionID:   resourceID,
					},
				)
				if err != nil || session == nil {
					return "", false, err
				}
				return session.ProjectID, true, nil
			})
		case scopedProjectLookupRehydrate:
			return runScopedProjectLookup(service.engramRehydrate != nil, func() (string, bool, error) {
				bundle, err := service.engramRehydrate.RehydrateEngram(
					ctx,
					EngramRehydrateRequest{
						ActorUserID: actorUserID,
						EngramID:    resourceID,
					},
				)
				if err != nil || bundle == nil {
					return "", false, err
				}
				return bundle.ProjectID, true, nil
			})
		default:
			return "", nil
		}
	}
}

func (service *CompatibilityService) lookupScopedProjectID(
	ctx context.Context,
	actor Actor,
	resource scopedProjectResource,
	resourceID uuid.UUID,
) (string, error) {
	switch resource {
	case scopedProjectResourceEngram:
		return lookupScopedProjectFromService(
			service.engramGet != nil,
			func() (*models.AdminEngramRecord, error) {
				return service.engramGet.GetEngram(
					ctx,
					EngramGetRequest{
						ActorUserID:    actor.UserID,
						ActorRole:      normalizedActorRole(actor),
						EngramID:       resourceID,
						IncludeDeleted: true,
					},
				)
			},
			func(record *models.AdminEngramRecord) string { return record.ProjectID },
		)
	case scopedProjectResourceCollection:
		return lookupScopedProjectFromService(
			service.engramCollectionGet != nil,
			func() (*models.EngramCollectionRecord, error) {
				return service.engramCollectionGet.GetCollection(
					ctx,
					EngramCollectionGetRequest{
						ActorUserID:    actor.UserID,
						ActorRole:      normalizedActorRole(actor),
						CollectionID:   resourceID,
						IncludeDeleted: true,
					},
				)
			},
			func(record *models.EngramCollectionRecord) string { return record.ProjectID },
		)
	case scopedProjectResourceLink:
		return lookupScopedProjectFromService(
			service.engramLinkGet != nil,
			func() (*models.EngramLinkRecord, error) {
				return service.engramLinkGet.GetEngramLink(
					ctx,
					EngramLinkGetRequest{
						ActorUserID:     actor.UserID,
						LinkID:          resourceID,
						IncludeArchived: true,
					},
				)
			},
			func(record *models.EngramLinkRecord) string { return record.ProjectID },
		)
	default:
		return "", nil
	}
}

func lookupScopedProjectFromService[T any](
	enabled bool,
	fetch func() (*T, error),
	projectIDForRecord func(*T) string,
) (string, error) {
	return runScopedProjectLookup(enabled, func() (string, bool, error) {
		record, err := fetch()
		if err != nil || record == nil {
			return "", false, err
		}
		return strings.TrimSpace(projectIDForRecord(record)), true, nil
	})
}

func runScopedProjectLookup(
	enabled bool,
	lookup func() (string, bool, error),
) (string, error) {
	if !enabled {
		return "", nil
	}
	projectID, found, err := lookup()
	if err != nil || !found {
		return "", err
	}
	return projectID, nil
}

func resolveScopedProjectID(
	params map[string]any,
	idField string,
	lookupProjectID func(uuid.UUID) (string, error),
) (string, *toolPolicyError) {
	resourceID, ok := requiredUUIDParam(params, idField)
	if !ok {
		return "", invalidParamPolicyError(idField)
	}
	projectID, err := lookupProjectID(resourceID)
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(projectID), nil
}

func invalidParamPolicyError(field string) *toolPolicyError {
	return &toolPolicyError{
		code:    -32602,
		message: "Invalid params",
		data: map[string]any{
			"invalid": field,
		},
	}
}

func normalizeTargetProjectIDParam(params map[string]any) string {
	value, found := optionalParamValue(params, "target_project_id")
	if !found {
		return ""
	}
	return strings.TrimSpace(stringParam(value))
}

func stringParam(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}
