package mcp

import (
	"context"
	"testing"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramShareAndUnshareParity(t *testing.T) {
	actorUserID := uuid.MustParse("50040000-0000-0000-0000-000000000501")
	engramID := uuid.MustParse("50040000-0000-0000-0000-000000000502")
	service := &fakeProjectShareService{
		shareResult: &models.EngramVisibilityRecord{
			EngramID:        engramID,
			ProjectID:       "engram-vault",
			VisibilityScope: models.VisibilityScopeProject,
		},
		unshareResult: &models.EngramVisibilityRecord{
			EngramID:        engramID,
			ProjectID:       "engram-vault",
			VisibilityScope: models.VisibilityScopePrivate,
		},
	}

	shareFrame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"engram.share",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	shared := engramVisibilityPayloadFromFrame(t, shareFrame, false)
	if shared.VisibilityScope != models.VisibilityScopeProject {
		t.Fatalf("expected project visibility from share response")
	}
	if service.shareCall.actorRole != models.UserRoleViewer {
		t.Fatalf("expected direct-call actor role normalization")
	}

	unshareFrame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(service),
		toolsCallRequest(
			actorUserID.String(),
			"engram_unshare",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	unshared := engramVisibilityPayloadFromFrame(t, unshareFrame, true)
	if unshared.VisibilityScope != models.VisibilityScopePrivate {
		t.Fatalf("expected private visibility from unshare response")
	}
	if service.unshareCall.actorRole != models.UserRoleAnalyst {
		t.Fatalf("expected tools-call actor role normalization")
	}
}

func TestCompatibilityServiceEngramShareValidationAndErrorMapping(t *testing.T) {
	invalidFrame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(&fakeProjectShareService{}),
		toolsCallRequest(
			"50050000-0000-0000-0000-000000000501",
			"engram_share",
			map[string]any{"engram_id": "bad"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, invalidFrame), -32602)

	forbiddenFrame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(
			&fakeProjectShareService{shareErr: projects.ErrEngramShareForbidden},
		),
		toolsCallRequest(
			"50050000-0000-0000-0000-000000000502",
			"engram_share",
			map[string]any{"engram_id": uuid.NewString()},
		),
	)
	forbiddenPayload := errorPayloadFromFrame(t, forbiddenFrame)
	requireErrorCode(t, forbiddenPayload, -32602)
	assertErrorStatusCode(t, forbiddenPayload, 403)
	assertErrorDetail(t, forbiddenPayload, projects.ErrEngramShareForbidden.Error())

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(
			&fakeProjectShareService{unshareErr: projects.ErrEngramNotFound},
		),
		toolsCallRequest(
			"50050000-0000-0000-0000-000000000503",
			"engram_unshare",
			map[string]any{"engram_id": uuid.NewString()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertErrorStatusCode(t, notFoundPayload, 404)
	assertErrorDetail(t, notFoundPayload, projects.ErrEngramNotFound.Error())
}

func engramVisibilityPayloadFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.EngramVisibilityRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	engramValue, exists := payload["engram"]
	if !exists {
		t.Fatalf("expected engram visibility payload")
	}
	switch typed := engramValue.(type) {
	case models.EngramVisibilityRecord:
		return typed
	case *models.EngramVisibilityRecord:
		if typed == nil {
			t.Fatalf("expected non-nil engram pointer")
		}
		return *typed
	default:
		t.Fatalf("unexpected engram payload type %T", engramValue)
		return models.EngramVisibilityRecord{}
	}
}

type projectShareCall struct {
	actorUserID uuid.UUID
	actorRole   models.UserRole
	engramID    uuid.UUID
}

type fakeProjectShareService struct {
	fakeProjectListService
	shareResult   *models.EngramVisibilityRecord
	unshareResult *models.EngramVisibilityRecord
	shareErr      error
	unshareErr    error
	shareCall     projectShareCall
	unshareCall   projectShareCall
}

func (service *fakeProjectShareService) ShareEngram(
	_ context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	service.shareCall = projectShareCall{
		actorUserID: actorUserID,
		actorRole:   actorRole,
		engramID:    engramID,
	}
	if service.shareErr != nil {
		return nil, service.shareErr
	}
	return service.shareResult, nil
}

func (service *fakeProjectShareService) UnshareEngram(
	_ context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	service.unshareCall = projectShareCall{
		actorUserID: actorUserID,
		actorRole:   actorRole,
		engramID:    engramID,
	}
	if service.unshareErr != nil {
		return nil, service.unshareErr
	}
	return service.unshareResult, nil
}
