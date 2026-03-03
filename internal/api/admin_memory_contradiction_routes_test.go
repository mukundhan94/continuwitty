package api

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestMountMemoryAdminRoutesListEngramContradictionAlertsUsesRequestObject(t *testing.T) {
	captured := admin.EngramContradictionAlertListRequest{}
	status := models.ContradictionAlertStatusOpen
	service := &fakeMemoryAdminService{
		listContradictionFn: func(
			_ context.Context,
			request admin.EngramContradictionAlertListRequest,
		) ([]models.EngramContradictionAlert, error) {
			captured = request
			return []models.EngramContradictionAlert{}, nil
		},
	}
	router := contradictionAdminRouter(service, uuid.MustParse("00000000-0000-0000-0000-000000000175"))

	response := executeRequest(
		router,
		http.MethodGet,
		"/api/v1/admin/memory/engrams/contradictions/alerts?project_id=engram-vault&status=open&limit=25&offset=7",
		nil,
	)
	requireEqual(t, http.StatusOK, response.Code)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, status, *captured.Status)
	requireEqual(t, 25, captured.Limit)
	requireEqual(t, 7, captured.Offset)
}

func TestMountMemoryAdminRoutesResolveContradictionAlertUsesActorAndPayload(t *testing.T) {
	alertID := uuid.MustParse("00000000-0000-0000-0000-000000000181")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000182")
	capturedAlertID := uuid.Nil
	capturedActorUserID := uuid.Nil
	capturedRequest := admin.EngramContradictionAlertResolveRequest{}
	service := &fakeMemoryAdminService{
		resolveContradictionFn: func(
			_ context.Context,
			receivedAlertID uuid.UUID,
			receivedActorUserID uuid.UUID,
			request admin.EngramContradictionAlertResolveRequest,
		) (*models.EngramContradictionAlert, error) {
			capturedAlertID = receivedAlertID
			capturedActorUserID = receivedActorUserID
			capturedRequest = request
			return &models.EngramContradictionAlert{
				AlertID: alertID,
				Status:  models.ContradictionAlertStatusResolved,
			}, nil
		},
	}
	router := contradictionAdminRouter(service, actorUserID)

	response := executeRequest(
		router,
		http.MethodPost,
		"/api/v1/admin/memory/engrams/contradictions/alerts/"+alertID.String()+"/resolve",
		[]byte(`{"project_id":"engram-vault","status":"resolved"}`),
	)
	requireEqual(t, http.StatusOK, response.Code)
	requireEqual(t, alertID, capturedAlertID)
	requireEqual(t, actorUserID, capturedActorUserID)
	requireEqual(t, "engram-vault", derefString(capturedRequest.ProjectID))
	requireEqual(t, models.ContradictionAlertStatusResolved, capturedRequest.Status)
}

func TestMountMemoryAdminRoutesContradictionRoutesRejectInvalidStatus(t *testing.T) {
	service := &fakeMemoryAdminService{
		listContradictionFn: func(
			_ context.Context,
			_ admin.EngramContradictionAlertListRequest,
		) ([]models.EngramContradictionAlert, error) {
			return nil, errors.New("unexpected")
		},
		resolveContradictionFn: func(
			_ context.Context,
			_ uuid.UUID,
			_ uuid.UUID,
			_ admin.EngramContradictionAlertResolveRequest,
		) (*models.EngramContradictionAlert, error) {
			return nil, errors.New("unexpected")
		},
	}
	router := contradictionAdminRouter(service, uuid.MustParse("00000000-0000-0000-0000-000000000183"))
	testCases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{
			name:   "list invalid status",
			method: http.MethodGet,
			path:   "/api/v1/admin/memory/engrams/contradictions/alerts?status=invalid",
		},
		{
			name:   "resolve invalid status",
			method: http.MethodPost,
			path:   "/api/v1/admin/memory/engrams/contradictions/alerts/00000000-0000-0000-0000-000000000184/resolve",
			body:   []byte(`{"status":"open"}`),
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			response := executeRequest(router, testCase.method, testCase.path, testCase.body)
			requireEqual(t, http.StatusBadRequest, response.Code)
		})
	}
}

func contradictionAdminRouter(service MemoryAdminService, actorUserID uuid.UUID) http.Handler {
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: actorUserID, Role: "admin"}, nil
	})
	return router
}
