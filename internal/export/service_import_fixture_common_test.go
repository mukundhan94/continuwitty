package export

import (
	"context"
	"encoding/json"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func setVisibleProject(projectService *fakeProjectLookup, actorID uuid.UUID) {
	projectService.getProjectFn = func(
		_ context.Context,
		_ uuid.UUID,
		_ models.UserRole,
		projectID string,
		_ bool,
	) (*models.ProjectRecord, error) {
		return &models.ProjectRecord{ProjectID: projectID, OwnerUserID: actorID}, nil
	}
}

func marshalBundle(t *testing.T, bundle ProjectExportBundle) []byte {
	t.Helper()
	fileBytes, err := json.Marshal(bundle)
	requireNoError(t, err)
	return fileBytes
}
