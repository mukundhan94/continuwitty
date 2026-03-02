package export

import (
	"context"
	"fmt"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

// ProjectExportFormat controls export payload container type.
type ProjectExportFormat string

const (
	// ProjectExportFormatJSON returns plain JSON payload bytes.
	ProjectExportFormatJSON ProjectExportFormat = "json"
	// ProjectExportFormatZIP returns ZIP payload bytes containing export.json.
	ProjectExportFormatZIP ProjectExportFormat = "zip"
)

// ParseProjectExportFormat validates and parses export format query values.
func ParseProjectExportFormat(value string) (ProjectExportFormat, error) {
	switch ProjectExportFormat(strings.TrimSpace(strings.ToLower(value))) {
	case "", ProjectExportFormatJSON:
		return ProjectExportFormatJSON, nil
	case ProjectExportFormatZIP:
		return ProjectExportFormatZIP, nil
	default:
		return "", fmt.Errorf("unsupported export format %q", value)
	}
}

// ProjectImportConflictPolicy controls behavior when import conflicts are found.
type ProjectImportConflictPolicy string

const (
	// ProjectImportConflictPolicySkip keeps existing rows and skips conflicts.
	ProjectImportConflictPolicySkip ProjectImportConflictPolicy = "skip"
	// ProjectImportConflictPolicyOverwrite soft-deletes existing rows and imports replacements.
	ProjectImportConflictPolicyOverwrite ProjectImportConflictPolicy = "overwrite"
	// ProjectImportConflictPolicyRename preserves existing rows and imports with renamed entities.
	ProjectImportConflictPolicyRename ProjectImportConflictPolicy = "rename"
)

// ParseProjectImportConflictPolicy validates and parses import conflict policy values.
func ParseProjectImportConflictPolicy(value string) (ProjectImportConflictPolicy, error) {
	switch ProjectImportConflictPolicy(strings.TrimSpace(strings.ToLower(value))) {
	case "", ProjectImportConflictPolicySkip:
		return ProjectImportConflictPolicySkip, nil
	case ProjectImportConflictPolicyOverwrite:
		return ProjectImportConflictPolicyOverwrite, nil
	case ProjectImportConflictPolicyRename:
		return ProjectImportConflictPolicyRename, nil
	default:
		return "", fmt.Errorf("unsupported conflict policy %q", value)
	}
}

// ProjectExportCollectionItemsRecord captures collection membership rows in a bundle.
type ProjectExportCollectionItemsRecord struct {
	CollectionID uuid.UUID   `json:"collection_id"`
	EngramIDs    []uuid.UUID `json:"engram_ids"`
}

// ProjectExportBundle captures project-level export payload contents.
type ProjectExportBundle struct {
	SchemaVersion         string                               `json:"schema_version"`
	ExportedAt            time.Time                            `json:"exported_at"`
	Project               models.ProjectRecord                 `json:"project"`
	Collections           []models.EngramCollectionRecord      `json:"collections"`
	CollectionItems       []ProjectExportCollectionItemsRecord `json:"collection_items"`
	Engrams               []models.AdminEngramRecord           `json:"engrams"`
	SelectedCollectionIDs []uuid.UUID                          `json:"selected_collection_ids"`
	IncludeEmbeddings     bool                                 `json:"include_embeddings"`
}

// ProjectImportResponse captures import counters returned to API callers.
type ProjectImportResponse struct {
	TargetProjectID         string                      `json:"target_project_id"`
	ImportedEngrams         int                         `json:"imported_engrams"`
	SkippedEngrams          int                         `json:"skipped_engrams"`
	OverwrittenEngrams      int                         `json:"overwritten_engrams"`
	ImportedCollections     int                         `json:"imported_collections"`
	ReusedCollections       int                         `json:"reused_collections"`
	ImportedCollectionItems int                         `json:"imported_collection_items"`
	ConflictPolicy          ProjectImportConflictPolicy `json:"conflict_policy"`
}

// ExportProjectRequest captures actor-scoped export operation inputs.
type ExportProjectRequest struct {
	ActorUserID       uuid.UUID
	ActorRole         string
	ProjectID         string
	CollectionIDs     []uuid.UUID
	IncludeEmbeddings bool
}

// ImportProjectRequest captures actor-scoped import operation inputs.
type ImportProjectRequest struct {
	ActorUserID     uuid.UUID
	ActorRole       string
	TargetProjectID string
	FileBytes       []byte
	ConflictPolicy  ProjectImportConflictPolicy
}

// Service captures export/import operations required by route handlers.
type Service interface {
	BuildProjectExportBundle(ctx context.Context, request ExportProjectRequest) (ProjectExportBundle, error)
	ImportProjectBundle(ctx context.Context, request ImportProjectRequest) (ProjectImportResponse, error)
}
