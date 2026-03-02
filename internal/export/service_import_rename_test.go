package export

import (
	"context"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestImportProjectBundleRenamePolicyCreatesRows(t *testing.T) {
	scenario := newRenameImportScenario(t)

	result, err := scenario.service.ImportProjectBundle(
		context.Background(),
		ImportProjectRequest{
			ActorUserID:     scenario.actorID,
			ActorRole:       string(models.UserRoleAnalyst),
			TargetProjectID: scenario.projectID,
			FileBytes:       scenario.fileBytes,
			ConflictPolicy:  ProjectImportConflictPolicyRename,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, result.ImportedEngrams)
	requireEqual(t, 0, result.SkippedEngrams)
	requireEqual(t, 0, result.OverwrittenEngrams)
	requireEqual(t, 1, result.ImportedCollections)
	requireEqual(t, 0, result.ReusedCollections)
	requireEqual(t, 1, result.ImportedCollectionItems)
	requireEqual(t, "Collision (imported)", scenario.createdPayload.Payload.Title)
	requireEqual(t, "Collision Collection (imported)", *scenario.capturedCollectionName)
	requireEqual(t, []uuid.UUID{scenario.newEngramID}, *scenario.capturedCollectionItems)
	requireEqual(t, 1, len(*scenario.capturedSourceInputs))
	requireEqual(t, "https://example.com/export-source", (*scenario.capturedSourceInputs)[0].URL)
}
