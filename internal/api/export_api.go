package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	internalexport "engram/internal/export"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	exportRouteDefaultPolicy = string(internalexport.ProjectImportConflictPolicySkip)
)

// MountExportRoutes registers project export/import endpoints.
func MountExportRoutes(router chi.Router, service internalexport.Service) {
	router.Get("/api/v1/projects/{project_id}/export", func(writer http.ResponseWriter, request *http.Request) {
		handleProjectExport(writer, request, service)
	})
	router.Post("/api/v1/projects/{project_id}/import", func(writer http.ResponseWriter, request *http.Request) {
		handleProjectImport(writer, request, service)
	})
}

func handleProjectExport(
	writer http.ResponseWriter,
	request *http.Request,
	service internalexport.Service,
) {
	if service == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "export service is not configured"})
		return
	}
	actor, ok := requireExportActor(writer, request)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(chi.URLParam(request, "project_id"))
	if projectID == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	exportFormat, ok := parseExportFormat(writer, request.URL.Query().Get("format"))
	if !ok {
		return
	}
	collectionIDs, ok := parseCollectionIDs(writer, request.URL.Query()["collection_ids"])
	if !ok {
		return
	}
	includeEmbeddings := parseOptionalExportBoolQuery(request.URL.Query().Get("include_embeddings"), false)

	bundle, err := service.BuildProjectExportBundle(
		request.Context(),
		internalexport.ExportProjectRequest{
			ActorUserID:       actor.UserID,
			ActorRole:         actor.Role,
			ProjectID:         projectID,
			CollectionIDs:     collectionIDs,
			IncludeEmbeddings: includeEmbeddings,
		},
	)
	if err != nil {
		writeExportServiceError(writer, err)
		return
	}

	jsonPayload, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	filename := buildExportFilename(projectID, exportFormat)
	writer.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	if exportFormat == internalexport.ProjectExportFormatZIP {
		zipPayload, err := buildZipExportPayload(jsonPayload)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
			return
		}
		writer.Header().Set("Content-Type", "application/zip")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(zipPayload)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(jsonPayload)
}

func handleProjectImport(
	writer http.ResponseWriter,
	request *http.Request,
	service internalexport.Service,
) {
	if service == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "export service is not configured"})
		return
	}
	actor, ok := requireExportActor(writer, request)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(chi.URLParam(request, "project_id"))
	if projectID == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	conflictPolicy, ok := parseConflictPolicy(writer, request.URL.Query().Get("conflict_policy"))
	if !ok {
		return
	}
	fileBytes, ok := parseImportFileBytes(writer, request)
	if !ok {
		return
	}

	result, err := service.ImportProjectBundle(
		request.Context(),
		internalexport.ImportProjectRequest{
			ActorUserID:     actor.UserID,
			ActorRole:       actor.Role,
			TargetProjectID: projectID,
			FileBytes:       fileBytes,
			ConflictPolicy:  conflictPolicy,
		},
	)
	if err != nil {
		writeExportServiceError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func writeExportServiceError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, internalexport.ErrProjectNotFound):
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Project not found"})
	case errors.Is(err, internalexport.ErrCollectionNotFoundForProject):
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Collection not found for project"})
	case errors.Is(err, internalexport.ErrImportFileEmpty):
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"detail": "Import file is empty"})
	case errors.Is(err, internalexport.ErrImportZipMissingExportJSON):
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"detail": "ZIP missing export.json"})
	case errors.Is(err, internalexport.ErrUnsupportedImportFileFormat):
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"detail": "Unsupported import file format"})
	case errors.Is(err, internalexport.ErrInvalidExportBundle):
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"detail": "Invalid export bundle"})
	default:
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
	}
}

func requireExportActor(writer http.ResponseWriter, request *http.Request) (AdminActor, bool) {
	actor, ok := AdminActorFromContext(request.Context())
	if !ok {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "authentication required"})
		return AdminActor{}, false
	}
	return actor, true
}

func parseExportFormat(writer http.ResponseWriter, raw string) (internalexport.ProjectExportFormat, bool) {
	value, err := internalexport.ParseProjectExportFormat(raw)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid format"})
		return "", false
	}
	return value, true
}

func parseCollectionIDs(writer http.ResponseWriter, raw []string) ([]uuid.UUID, bool) {
	if len(raw) == 0 {
		return []uuid.UUID{}, true
	}
	parsed := make([]uuid.UUID, 0, len(raw))
	for _, item := range raw {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		collectionID, err := uuid.Parse(value)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid collection_ids"})
			return nil, false
		}
		parsed = append(parsed, collectionID)
	}
	return parsed, true
}

func parseOptionalExportBoolQuery(raw string, defaultValue bool) bool {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	switch trimmed {
	case "":
		return defaultValue
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseConflictPolicy(
	writer http.ResponseWriter,
	raw string,
) (internalexport.ProjectImportConflictPolicy, bool) {
	value := raw
	if strings.TrimSpace(value) == "" {
		value = exportRouteDefaultPolicy
	}
	parsed, err := internalexport.ParseProjectImportConflictPolicy(value)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid conflict_policy"})
		return "", false
	}
	return parsed, true
}

func parseImportFileBytes(writer http.ResponseWriter, request *http.Request) ([]byte, bool) {
	file, _, err := request.FormFile("file")
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "file is required"})
		return nil, false
	}
	defer file.Close()
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "file is required"})
		return nil, false
	}
	return fileBytes, true
}

func buildExportFilename(
	projectID string,
	exportFormat internalexport.ProjectExportFormat,
) string {
	suffix := "json"
	if exportFormat == internalexport.ProjectExportFormatZIP {
		suffix = "zip"
	}
	safeProject := strings.NewReplacer("/", "-", " ", "-").Replace(projectID)
	return "engram-export-" + safeProject + "." + suffix
}

func buildZipExportPayload(jsonPayload []byte) ([]byte, error) {
	buffer := &bytes.Buffer{}
	archive := zip.NewWriter(buffer)
	file, err := archive.Create("export.json")
	if err != nil {
		return nil, err
	}
	if _, err := file.Write(jsonPayload); err != nil {
		_ = archive.Close()
		return nil, err
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
