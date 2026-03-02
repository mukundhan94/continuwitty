package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	internalexport "engram/internal/export"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestMountExportRoutesMapsProjectAndCollectionErrors(t *testing.T) {
	testCases := []struct {
		name       string
		serviceErr error
		statusCode int
	}{
		{
			name:       "project not found",
			serviceErr: internalexport.ErrProjectNotFound,
			statusCode: http.StatusNotFound,
		},
		{
			name:       "collection not found",
			serviceErr: internalexport.ErrCollectionNotFoundForProject,
			statusCode: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := chi.NewRouter()
			MountExportRoutes(
				router,
				&fakeExportRouteService{
					buildBundleFn: func(_ context.Context, _ internalexport.ExportProjectRequest) (internalexport.ProjectExportBundle, error) {
						return internalexport.ProjectExportBundle{}, testCase.serviceErr
					},
				},
			)

			request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-one/export", nil)
			request = WithAdminActor(request, AdminActor{UserID: uuid.New(), Role: "analyst"})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != testCase.statusCode {
				t.Fatalf("expected status %d, got %d", testCase.statusCode, response.Code)
			}
		})
	}
}

func TestMountExportRoutesMapsImportBundleErrors(t *testing.T) {
	testCases := []struct {
		name       string
		serviceErr error
	}{
		{name: "import file empty", serviceErr: internalexport.ErrImportFileEmpty},
		{name: "zip missing export json", serviceErr: internalexport.ErrImportZipMissingExportJSON},
		{name: "unsupported format", serviceErr: internalexport.ErrUnsupportedImportFileFormat},
		{name: "invalid bundle", serviceErr: internalexport.ErrInvalidExportBundle},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := chi.NewRouter()
			MountExportRoutes(
				router,
				&fakeExportRouteService{
					importBundleFn: func(_ context.Context, _ internalexport.ImportProjectRequest) (internalexport.ProjectImportResponse, error) {
						return internalexport.ProjectImportResponse{}, testCase.serviceErr
					},
				},
			)

			request := newImportRequestWithActor(t, "/api/v1/projects/project-one/import")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected status 422, got %d", response.Code)
			}
		})
	}
}

func newImportRequestWithActor(t *testing.T, path string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", "bundle.json")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = fileWriter.Write([]byte(`{"schema_version":"1.0"}`))
	_ = writer.Close()

	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return WithAdminActor(request, AdminActor{UserID: uuid.New(), Role: "analyst"})
}
