package mcp

import (
	"testing"

	"github.com/google/uuid"
)

func TestCompatibilityServiceDispatchesProjectList(t *testing.T) {
	actorUserID := uuid.MustParse("22000000-0000-0000-0000-000000000022")
	captured := projectListCallCapture{}
	frame := dispatchProjectListForTest(t, actorUserID, &captured)
	assertProjectListCall(t, captured, actorUserID)
	assertProjectListPayload(t, frame)
}
