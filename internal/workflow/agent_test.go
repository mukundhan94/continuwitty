package workflow

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestRunCreatesEngramAndStoresState(t *testing.T) {
	creator := &stubEngramCreator{
		issuedIDs: []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000a01"),
		},
	}
	service := NewService(creator.create)
	service.deps.nowUTC = func() time.Time {
		return time.Date(2026, 2, 26, 9, 0, 0, 0, time.UTC)
	}
	capturedAt := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)

	state, err := service.Run(
		context.Background(),
		AgentRunRequest{
			ProjectID:           "project-agent",
			ThreadID:            "thread-1",
			Objective:           "Evaluate durable memory checkpointing",
			Notes:               []string{"Use LangGraph for resumable threads."},
			Assumptions:         []string{"Local single-user mode"},
			Tags:                []string{"langgraph"},
			Keywords:            []string{"checkpoint"},
			Sources:             []AgentSourceInput{{URL: "https://example.com/langgraph", Snippet: "Checkpointed runs can resume by thread id.", CapturedAt: &capturedAt}},
			AutoPersistEngram:   true,
			SnapshotEnabled:     false,
			SnapshotEveryNNotes: 3,
		},
	)
	requireNoWorkflowError(t, err)
	requireEqualWorkflow(t, defaultRunStatusCompleted, state.Status)
	if state.EngramID == nil {
		t.Fatalf("expected engram_id to be set")
	}
	requireEqualWorkflow(t, 0, len(state.SnapshotEngramIDs))
	requireEqualWorkflow(t, 1, len(creator.calls))
	requireEqualWorkflow(t, "agent.persist", creator.calls[0].origin)

	persisted, found := service.GetState("thread-1")
	if !found {
		t.Fatalf("expected stored state for thread")
	}
	requireEqualWorkflow(t, "Evaluate durable memory checkpointing", persisted.Objective)
	if persisted.EngramID == nil {
		t.Fatalf("expected persisted state to contain engram_id")
	}
}

func TestResumeAppendsNotesWithoutPersistence(t *testing.T) {
	creator := &stubEngramCreator{}
	service := NewService(creator.create)

	_, err := service.Run(
		context.Background(),
		AgentRunRequest{
			ProjectID:           "project-resume",
			ThreadID:            "thread-2",
			Objective:           "Resume objective",
			Notes:               []string{"Initial note"},
			AutoPersistEngram:   false,
			SnapshotEnabled:     false,
			SnapshotEveryNNotes: 3,
		},
	)
	requireNoWorkflowError(t, err)

	state, found, err := service.Resume(
		context.Background(),
		"thread-2",
		AgentResumeRequest{
			Notes:             []string{"Follow-up note from resumed run"},
			Assumptions:       []string{"Assume continuous context"},
			AutoPersistEngram: boolPointer(false),
		},
	)
	requireNoWorkflowError(t, err)
	if !found {
		t.Fatalf("expected thread to be resumable")
	}
	requireEqualWorkflow(t, defaultRunStatusCompleted, state.Status)
	if state.EngramID != nil {
		t.Fatalf("expected engram_id to be nil when auto_persist_engram is false")
	}
	if !containsString(state.Notes, "Initial note") || !containsString(state.Notes, "Follow-up note from resumed run") {
		t.Fatalf("expected merged notes, got %#v", state.Notes)
	}
	requireEqualWorkflow(t, 0, len(creator.calls))
}

func TestSnapshotCreationOnNoteThreshold(t *testing.T) {
	creator := &stubEngramCreator{
		issuedIDs: []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000a11"),
		},
	}
	service := NewService(creator.create)

	state, err := service.Run(
		context.Background(),
		AgentRunRequest{
			ProjectID:           "project-snapshot",
			ThreadID:            "thread-3",
			Objective:           "Capture periodic snapshots",
			Notes:               []string{"note-1", "note-2", "note-3"},
			AutoPersistEngram:   false,
			SnapshotEnabled:     true,
			SnapshotEveryNNotes: 2,
		},
	)
	requireNoWorkflowError(t, err)
	if state.EngramID != nil {
		t.Fatalf("expected engram_id to be nil when auto_persist_engram is false")
	}
	requireEqualWorkflow(t, 1, len(state.SnapshotEngramIDs))
	requireEqualWorkflow(t, 1, len(creator.calls))
	requireEqualWorkflow(t, "agent.snapshot", creator.calls[0].origin)
}

func TestSnapshotCreationContinuesAcrossResume(t *testing.T) {
	creator := &stubEngramCreator{
		issuedIDs: []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000a21"),
			uuid.MustParse("00000000-0000-0000-0000-000000000a22"),
		},
	}
	service := NewService(creator.create)

	first, err := service.Run(
		context.Background(),
		AgentRunRequest{
			ProjectID:           "project-snapshot-resume",
			ThreadID:            "thread-4",
			Objective:           "Resume with snapshots",
			Notes:               []string{"note-1"},
			AutoPersistEngram:   false,
			SnapshotEnabled:     true,
			SnapshotEveryNNotes: 2,
		},
	)
	requireNoWorkflowError(t, err)
	requireEqualWorkflow(t, 0, len(first.SnapshotEngramIDs))

	resumedOne, found, err := service.Resume(
		context.Background(),
		"thread-4",
		AgentResumeRequest{
			Notes:             []string{"note-2"},
			AutoPersistEngram: boolPointer(false),
		},
	)
	requireNoWorkflowError(t, err)
	if !found {
		t.Fatalf("expected first resume to find thread state")
	}
	requireEqualWorkflow(t, 1, len(resumedOne.SnapshotEngramIDs))

	resumedTwo, found, err := service.Resume(
		context.Background(),
		"thread-4",
		AgentResumeRequest{
			Notes:             []string{"note-3", "note-4"},
			AutoPersistEngram: boolPointer(false),
		},
	)
	requireNoWorkflowError(t, err)
	if !found {
		t.Fatalf("expected second resume to find thread state")
	}
	requireEqualWorkflow(t, 2, len(resumedTwo.SnapshotEngramIDs))
	if !containsString(resumedTwo.SnapshotEngramIDs, resumedOne.SnapshotEngramIDs[0]) {
		t.Fatalf("expected previous snapshot ids to be retained, got %#v", resumedTwo.SnapshotEngramIDs)
	}
	requireEqualWorkflow(t, 2, len(creator.calls))
}

func TestGetStateAndResumeNotFound(t *testing.T) {
	service := NewService(nil)

	_, found := service.GetState("missing-thread")
	if found {
		t.Fatalf("expected missing thread to return not found")
	}

	_, found, err := service.Resume(context.Background(), "missing-thread", AgentResumeRequest{Notes: []string{"note"}})
	requireNoWorkflowError(t, err)
	if found {
		t.Fatalf("expected resume to report missing state")
	}
}

type stubCreateCall struct {
	origin  string
	payload models.MemoryEngramCreate
}

type stubEngramCreator struct {
	issuedIDs []uuid.UUID
	calls     []stubCreateCall
}

func (creator *stubEngramCreator) create(
	_ context.Context,
	payload models.MemoryEngramCreate,
	enrichmentOrigin string,
) (*models.EngramCreateResponse, error) {
	creator.calls = append(creator.calls, stubCreateCall{
		origin:  enrichmentOrigin,
		payload: payload,
	})
	issuedID := uuid.New()
	if len(creator.issuedIDs) >= len(creator.calls) {
		issuedID = creator.issuedIDs[len(creator.calls)-1]
	}
	return &models.EngramCreateResponse{
		EngramID:  issuedID,
		CreatedAt: time.Date(2026, 2, 26, 8, 0, 0, 0, time.UTC),
	}, nil
}

func boolPointer(value bool) *bool {
	return &value
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func requireNoWorkflowError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func requireEqualWorkflow[T comparable](t *testing.T, expected T, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}
