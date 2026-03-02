package export

import (
	"context"
	"encoding/json"
	"testing"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

func setVisibleProject(projectService *fakeProjectLookup, actorID uuid.UUID) {
	projectService.getProjectFn = func(
		_ context.Context,
		_ projects.ActorContext,
		request projects.ProjectGetRequest,
	) (*models.ProjectRecord, error) {
		return &models.ProjectRecord{ProjectID: request.ProjectID, OwnerUserID: actorID}, nil
	}
}

func marshalBundle(t *testing.T, bundle ProjectExportBundle) []byte {
	t.Helper()
	fileBytes, err := json.Marshal(bundle)
	requireNoError(t, err)
	return fileBytes
}
