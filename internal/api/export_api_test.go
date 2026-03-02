package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	internalexport "engram/internal/export"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestMountExportRoutesJSONResponse(t *testing.T) {
	collectionID := uuid.MustParse("00000000-0000-0000-0000-000000000b01")
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000b02")
	service := &fakeExportRouteService{
		buildBundleFn: func(
			_ context.Context,
			request internalexport.ExportProjectRequest,
		) (internalexport.ProjectExportBundle, error) {
			assertExportRequestMatches(t, request, actorID, collectionID)
			return testExportBundle(), nil
		},
	}
	router := chi.NewRouter()
	MountExportRoutes(router, service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/project-one/export?format=json&include_embeddings=true&collection_ids="+collectionID.String(),
		nil,
	)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "analyst"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected json content type, got %q", response.Header().Get("Content-Type"))
	}
	if !strings.Contains(response.Header().Get("Content-Disposition"), "engram-export-project-one.json") {
		t.Fatalf("expected json export filename, got %q", response.Header().Get("Content-Disposition"))
	}
}

func assertExportRequestMatches(
	t *testing.T,
	request internalexport.ExportProjectRequest,
	expectedActorID uuid.UUID,
	expectedCollectionID uuid.UUID,
) {
	t.Helper()
	if request.ActorUserID != expectedActorID {
		t.Fatalf("unexpected actor user id in request: %#v", request)
	}
	if !request.IncludeEmbeddings {
		t.Fatalf("expected include_embeddings=true in request: %#v", request)
	}
	if len(request.CollectionIDs) != 1 {
		t.Fatalf("expected one collection id in request: %#v", request)
	}
	if request.CollectionIDs[0] != expectedCollectionID {
		t.Fatalf("unexpected collection id in request: %#v", request)
	}
}

func TestMountExportRoutesZIPResponse(t *testing.T) {
	service := &fakeExportRouteService{
		buildBundleFn: func(_ context.Context, _ internalexport.ExportProjectRequest) (internalexport.ProjectExportBundle, error) {
			return testExportBundle(), nil
		},
	}
	router := chi.NewRouter()
	MountExportRoutes(router, service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/project-one/export?format=zip",
		nil,
	)
	request = WithAdminActor(request, AdminActor{UserID: uuid.New(), Role: "analyst"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "application/zip") {
		t.Fatalf("expected zip content type, got %q", response.Header().Get("Content-Type"))
	}

	readerAt := bytes.NewReader(response.Body.Bytes())
	archive, err := zip.NewReader(readerAt, int64(readerAt.Len()))
	if err != nil {
		t.Fatalf("parse zip payload: %v", err)
	}
	if len(archive.File) != 1 || archive.File[0].Name != "export.json" {
		t.Fatalf("expected zip payload with export.json, got %#v", archive.File)
	}
}

func TestMountExportRoutesImportReadsFile(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000b03")
	service := &fakeExportRouteService{
		importBundleFn: func(
			_ context.Context,
			request internalexport.ImportProjectRequest,
		) (internalexport.ProjectImportResponse, error) {
			if request.ActorUserID != actorID || request.TargetProjectID != "project-one" {
				t.Fatalf("unexpected import request: %#v", request)
			}
			if string(request.FileBytes) != "{\"schema_version\":\"1.0\"}" {
				t.Fatalf("expected file bytes to be forwarded")
			}
			return internalexport.ProjectImportResponse{
				TargetProjectID: "project-one",
				ConflictPolicy:  request.ConflictPolicy,
			}, nil
		},
	}
	router := chi.NewRouter()
	MountExportRoutes(router, service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", "bundle.json")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	_, _ = fileWriter.Write([]byte(`{"schema_version":"1.0"}`))
	_ = writer.Close()

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/project-one/import?conflict_policy=overwrite",
		body,
	)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "analyst"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload internalexport.ProjectImportResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	if payload.ConflictPolicy != internalexport.ProjectImportConflictPolicyOverwrite {
		t.Fatalf("expected conflict policy overwrite, got %q", payload.ConflictPolicy)
	}
}

func TestMountExportRoutesValidationAndAuthErrors(t *testing.T) {
	router := chi.NewRouter()
	MountExportRoutes(router, &fakeExportRouteService{})

	unauthenticated := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-one/export", nil)
	unauthenticatedResponse := httptest.NewRecorder()
	router.ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status 401, got %d", unauthenticatedResponse.Code)
	}

	invalidFormat := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-one/export?format=tgz", nil)
	invalidFormat = WithAdminActor(invalidFormat, AdminActor{UserID: uuid.New(), Role: "analyst"})
	invalidFormatResponse := httptest.NewRecorder()
	router.ServeHTTP(invalidFormatResponse, invalidFormat)
	if invalidFormatResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid format status 400, got %d", invalidFormatResponse.Code)
	}

	invalidCollectionID := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-one/export?collection_ids=bad-uuid", nil)
	invalidCollectionID = WithAdminActor(invalidCollectionID, AdminActor{UserID: uuid.New(), Role: "analyst"})
	invalidCollectionResponse := httptest.NewRecorder()
	router.ServeHTTP(invalidCollectionResponse, invalidCollectionID)
	if invalidCollectionResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid collection_id status 400, got %d", invalidCollectionResponse.Code)
	}

	invalidPolicy := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-one/import?conflict_policy=merge", nil)
	invalidPolicy = WithAdminActor(invalidPolicy, AdminActor{UserID: uuid.New(), Role: "analyst"})
	invalidPolicyResponse := httptest.NewRecorder()
	router.ServeHTTP(invalidPolicyResponse, invalidPolicy)
	if invalidPolicyResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid policy status 400, got %d", invalidPolicyResponse.Code)
	}
}

type fakeExportRouteService struct {
	buildBundleFn  func(context.Context, internalexport.ExportProjectRequest) (internalexport.ProjectExportBundle, error)
	importBundleFn func(context.Context, internalexport.ImportProjectRequest) (internalexport.ProjectImportResponse, error)
}

func (service *fakeExportRouteService) BuildProjectExportBundle(
	ctx context.Context,
	request internalexport.ExportProjectRequest,
) (internalexport.ProjectExportBundle, error) {
	if service.buildBundleFn == nil {
		return testExportBundle(), nil
	}
	return service.buildBundleFn(ctx, request)
}

func (service *fakeExportRouteService) ImportProjectBundle(
	ctx context.Context,
	request internalexport.ImportProjectRequest,
) (internalexport.ProjectImportResponse, error) {
	if service.importBundleFn == nil {
		return internalexport.ProjectImportResponse{
			TargetProjectID: request.TargetProjectID,
			ConflictPolicy:  request.ConflictPolicy,
		}, nil
	}
	return service.importBundleFn(ctx, request)
}

func testExportBundle() internalexport.ProjectExportBundle {
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000b04")
	return internalexport.ProjectExportBundle{
		SchemaVersion: "1.0",
		ExportedAt:    time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
		Project: models.ProjectRecord{
			ProjectID:   "project-one",
			Name:        "Project One",
			Description: "",
			OwnerUserID: ownerID,
			IsArchived:  false,
			CreatedAt:   time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
		},
		Collections:           []models.EngramCollectionRecord{},
		CollectionItems:       []internalexport.ProjectExportCollectionItemsRecord{},
		Engrams:               []models.AdminEngramRecord{},
		SelectedCollectionIDs: []uuid.UUID{},
		IncludeEmbeddings:     false,
	}
}
