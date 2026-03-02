package export

import (
	"context"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestImportProjectBundleSkipPolicyReusesExistingRecords(t *testing.T) {
	scenario := newSkipImportScenario(t)

	result, err := scenario.service.ImportProjectBundle(
		context.Background(),
		ImportProjectRequest{
			ActorUserID:     scenario.actorID,
			ActorRole:       string(models.UserRoleAnalyst),
			TargetProjectID: scenario.projectID,
			FileBytes:       scenario.fileBytes,
			ConflictPolicy:  ProjectImportConflictPolicySkip,
		},
	)
	requireNoError(t, err)
	requireEqual(t, scenario.projectID, result.TargetProjectID)
	requireEqual(t, 0, result.ImportedEngrams)
	requireEqual(t, 1, result.SkippedEngrams)
	requireEqual(t, 0, result.OverwrittenEngrams)
	requireEqual(t, 0, result.ImportedCollections)
	requireEqual(t, 1, result.ReusedCollections)
	requireEqual(t, 1, result.ImportedCollectionItems)
	requireEqual(t, []uuid.UUID{scenario.existingEngramID}, *scenario.capturedCollectionIDs)
}
