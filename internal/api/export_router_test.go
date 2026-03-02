package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"engram/internal/config"
	internalexport "engram/internal/export"
	"engram/internal/models"

	"github.com/google/uuid"
)

func TestExportRoutesNotMountedWithoutDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	router := NewRouter(settings)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-one/export", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}

func TestExportRoutesMountedWithDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	serviceCalled := false
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			ExportService: &fakeExportRouteService{
				buildBundleFn: func(
					_ context.Context,
					_ internalexport.ExportProjectRequest,
				) (internalexport.ProjectExportBundle, error) {
					serviceCalled = true
					return internalexport.ProjectExportBundle{
						SchemaVersion: "1.0",
						ExportedAt:    time.Date(2026, 2, 26, 16, 0, 0, 0, time.UTC),
						Project: models.ProjectRecord{
							ProjectID:   "project-one",
							Name:        "Project One",
							Description: "",
							OwnerUserID: uuid.MustParse("00000000-0000-0000-0000-000000000e01"),
							IsArchived:  false,
							CreatedAt:   time.Date(2026, 2, 26, 16, 0, 0, 0, time.UTC),
							UpdatedAt:   time.Date(2026, 2, 26, 16, 0, 0, 0, time.UTC),
						},
					}, nil
				},
			},
		},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-one/export", nil)
	request = WithAdminActor(
		request,
		AdminActor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000e02"), Role: "analyst"},
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !serviceCalled {
		t.Fatalf("expected export service to be called")
	}
}
