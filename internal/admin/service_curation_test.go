package admin

import (
	"context"
	"errors"
	"testing"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestListMemoryCurationSuggestionsUsesRepositoryRequestObject(t *testing.T) {
	service := NewService(nil, 256, nil)
	projectID := "engram-vault"
	sessionID := uuid.MustParse("00000000-0000-0000-0000-00000000c011")
	suggestionType := models.MemoryCurationSuggestionTypeLink
	status := models.MemoryCurationSuggestionStatusSuggested
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000c012")
	captured := repository.MemoryCurationSuggestionListInput{}
	service.deps.listMemoryCurationSuggestions = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionListInput,
	) ([]models.MemoryCurationSuggestion, error) {
		captured = input
		return []models.MemoryCurationSuggestion{
			{
				SuggestionID:   suggestionID,
				ProjectID:      projectID,
				SessionID:      &sessionID,
				SuggestionType: suggestionType,
				Status:         status,
			},
		}, nil
	}

	listed, err := service.ListMemoryCurationSuggestions(
		context.Background(),
		MemoryCurationSuggestionListRequest{
			ProjectID:      &projectID,
			SessionID:      &sessionID,
			SuggestionType: &suggestionType,
			Status:         &status,
			Limit:          25,
			Offset:         4,
		},
	)
	requireNoError(t, err)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, sessionID, *captured.SessionID)
	requireEqual(t, suggestionType, *captured.SuggestionType)
	requireEqual(t, status, *captured.Status)
	requireEqual(t, 25, captured.Limit)
	requireEqual(t, 4, captured.Offset)
	requireEqual(t, 1, len(listed))
	requireEqual(t, suggestionID, listed[0].SuggestionID)
}

func TestActionMemoryCurationSuggestionUsesRepositoryRequestObject(t *testing.T) {
	service := NewService(nil, 256, nil)
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000c021")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000c022")
	projectID := "engram-vault"
	captured := repository.MemoryCurationSuggestionActionInput{}
	service.deps.applyMemoryCurationSuggestionAction = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionActionInput,
	) (*models.MemoryCurationSuggestion, error) {
		captured = input
		return &models.MemoryCurationSuggestion{
			SuggestionID: suggestionID,
			ProjectID:    projectID,
			Status:       models.MemoryCurationSuggestionStatusAccepted,
		}, nil
	}

	updated, err := service.ActionMemoryCurationSuggestion(
		context.Background(),
		suggestionID,
		actorUserID,
		MemoryCurationSuggestionActionRequest{
			ProjectID: &projectID,
			Status:    models.MemoryCurationSuggestionStatusAccepted,
		},
	)
	requireNoError(t, err)
	requireEqual(t, suggestionID, captured.SuggestionID)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, actorUserID, captured.ActorUserID)
	requireEqual(t, models.MemoryCurationSuggestionStatusAccepted, captured.Status)
	if updated == nil {
		t.Fatalf("expected updated memory curation suggestion")
	}
	requireEqual(t, suggestionID, updated.SuggestionID)
}

func TestActionMemoryCurationSuggestionRejectsInvalidStatus(t *testing.T) {
	service := NewService(nil, 256, nil)
	_, err := service.ActionMemoryCurationSuggestion(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-00000000c031"),
		uuid.MustParse("00000000-0000-0000-0000-00000000c032"),
		MemoryCurationSuggestionActionRequest{
			Status: models.MemoryCurationSuggestionStatusSuggested,
		},
	)
	if !errors.Is(err, ErrMemoryCurationSuggestionActionInvalid) {
		t.Fatalf("expected ErrMemoryCurationSuggestionActionInvalid, got %v", err)
	}
}

func TestActionMemoryCurationSuggestionReturnsNotFound(t *testing.T) {
	service := NewService(nil, 256, nil)
	service.deps.applyMemoryCurationSuggestionAction = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MemoryCurationSuggestionActionInput,
	) (*models.MemoryCurationSuggestion, error) {
		return nil, nil
	}

	_, err := service.ActionMemoryCurationSuggestion(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-00000000c041"),
		uuid.MustParse("00000000-0000-0000-0000-00000000c042"),
		MemoryCurationSuggestionActionRequest{Status: models.MemoryCurationSuggestionStatusRejected},
	)
	if !errors.Is(err, ErrMemoryCurationSuggestionNotFound) {
		t.Fatalf("expected ErrMemoryCurationSuggestionNotFound, got %v", err)
	}
}

func TestActionMemoryCurationSuggestionAppliedDispatchesConsolidationMerge(t *testing.T) {
	runAppliedCurationSuggestionSideEffectCase(
		t,
		appliedCurationSuggestionSideEffectCase{
			suggestionType: models.MemoryCurationSuggestionTypeConsolidate,
			payloadKey:     "consolidation_suggestion_id",
		},
	)
}

func TestActionMemoryCurationSuggestionAppliedDispatchesContradictionResolve(t *testing.T) {
	runAppliedCurationSuggestionSideEffectCase(
		t,
		appliedCurationSuggestionSideEffectCase{
			suggestionType: models.MemoryCurationSuggestionTypeContradiction,
			payloadKey:     "contradiction_alert_id",
		},
	)
}

func TestActionMemoryCurationSuggestionAppliedRejectsInvalidPayload(t *testing.T) {
	runAppliedCurationSuggestionRejectCase(
		t,
		appliedCurationSuggestionRejectCase{
			suggestionType: models.MemoryCurationSuggestionTypeConsolidate,
			payload:        map[string]any{},
			expectedErr:    ErrMemoryCurationSuggestionPayloadInvalid,
		},
	)
}

func TestActionMemoryCurationSuggestionAppliedDispatchesLinkArchive(t *testing.T) {
	service := NewService(nil, 256, nil)
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000c081")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000c082")
	projectID := "engram-vault"
	linkID := uuid.MustParse("00000000-0000-0000-0000-00000000c083")
	archiveCalled := false
	actionCalled := false
	service.deps.getMemoryCurationSuggestion = func(
		_ context.Context,
		_ repository.Queryer,
		_ uuid.UUID,
		_ *string,
	) (*models.MemoryCurationSuggestion, error) {
		return &models.MemoryCurationSuggestion{
			SuggestionID:   suggestionID,
			ProjectID:      projectID,
			SuggestionType: models.MemoryCurationSuggestionTypeLink,
			PayloadJSON: map[string]any{
				"suggested_action": "archive_stale_low_value",
				"link_id":          linkID.String(),
			},
		}, nil
	}
	service.deps.archiveEngramLink = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.EngramLinkArchiveInput,
	) (*models.EngramLinkRecord, error) {
		archiveCalled = true
		requireEqual(t, linkID, input.LinkID)
		requireEqual(t, actorUserID, input.ActorUserID)
		return &models.EngramLinkRecord{LinkID: linkID}, nil
	}
	service.deps.applyMemoryCurationSuggestionAction = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MemoryCurationSuggestionActionInput,
	) (*models.MemoryCurationSuggestion, error) {
		actionCalled = true
		return &models.MemoryCurationSuggestion{
			SuggestionID: suggestionID,
			ProjectID:    projectID,
			Status:       models.MemoryCurationSuggestionStatusApplied,
		}, nil
	}

	updated, err := service.ActionMemoryCurationSuggestion(
		context.Background(),
		suggestionID,
		actorUserID,
		MemoryCurationSuggestionActionRequest{
			ProjectID: &projectID,
			Status:    models.MemoryCurationSuggestionStatusApplied,
		},
	)
	requireNoError(t, err)
	if !archiveCalled {
		t.Fatalf("expected link archive side effect to run")
	}
	if !actionCalled {
		t.Fatalf("expected curation status update after link archive side effect")
	}
	assertAppliedCurationSuggestionStatus(t, updated)
}

func TestActionMemoryCurationSuggestionAppliedRejectsUnsupportedLinkAction(t *testing.T) {
	runAppliedCurationSuggestionRejectCase(
		t,
		appliedCurationSuggestionRejectCase{
			suggestionType: models.MemoryCurationSuggestionTypeLink,
			payload: map[string]any{
				"suggested_action": "review_relation_conflict",
				"link_id":          "00000000-0000-0000-0000-00000000c093",
			},
			expectedErr: ErrMemoryCurationSuggestionApplyUnsupported,
		},
	)
}

type appliedCurationSuggestionSideEffectCase struct {
	suggestionType models.MemoryCurationSuggestionType
	payloadKey     string
}

type appliedCurationSuggestionSetupInput struct {
	suggestionID   uuid.UUID
	projectID      string
	suggestionType models.MemoryCurationSuggestionType
	payloadKey     string
	relatedID      uuid.UUID
	actionCalled   *bool
}

type appliedCurationSuggestionRejectCase struct {
	suggestionType models.MemoryCurationSuggestionType
	payload        map[string]any
	expectedErr    error
}

func runAppliedCurationSuggestionSideEffectCase(
	t *testing.T,
	testCase appliedCurationSuggestionSideEffectCase,
) {
	t.Helper()
	service := NewService(nil, 256, nil)
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000c051")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000c052")
	projectID := "engram-vault"
	relatedID := uuid.MustParse("00000000-0000-0000-0000-00000000c053")
	sideEffectCalled := false
	actionCalled := false
	configureAppliedCurationSuggestion(
		service,
		appliedCurationSuggestionSetupInput{
			suggestionID:   suggestionID,
			projectID:      projectID,
			suggestionType: testCase.suggestionType,
			payloadKey:     testCase.payloadKey,
			relatedID:      relatedID,
			actionCalled:   &actionCalled,
		},
	)
	service.deps.applyConsolidationSuggestionAction = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ConsolidationSuggestionActionInput,
	) (*models.EngramConsolidationSuggestion, error) {
		sideEffectCalled = true
		requireEqual(t, relatedID, input.SuggestionID)
		requireEqual(t, models.ConsolidationSuggestionStatusMerged, input.Status)
		requireEqual(t, actorUserID, input.ActorUserID)
		return &models.EngramConsolidationSuggestion{SuggestionID: input.SuggestionID}, nil
	}
	service.deps.resolveContradictionAlert = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ContradictionAlertResolveInput,
	) (*models.EngramContradictionAlert, error) {
		sideEffectCalled = true
		requireEqual(t, relatedID, input.AlertID)
		requireEqual(t, models.ContradictionAlertStatusResolved, input.Status)
		requireEqual(t, actorUserID, input.ResolvedBy)
		return &models.EngramContradictionAlert{AlertID: input.AlertID}, nil
	}

	updated, err := service.ActionMemoryCurationSuggestion(
		context.Background(),
		suggestionID,
		actorUserID,
		MemoryCurationSuggestionActionRequest{
			ProjectID: &projectID,
			Status:    models.MemoryCurationSuggestionStatusApplied,
		},
	)
	requireNoError(t, err)
	if !sideEffectCalled {
		t.Fatalf("expected side effect to run")
	}
	if !actionCalled {
		t.Fatalf("expected curation suggestion action to run")
	}
	assertAppliedCurationSuggestionStatus(t, updated)
}

func runAppliedCurationSuggestionRejectCase(
	t *testing.T,
	testCase appliedCurationSuggestionRejectCase,
) {
	t.Helper()
	service := NewService(nil, 256, nil)
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000c071")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000c072")
	actionCalled := false
	service.deps.getMemoryCurationSuggestion = func(
		_ context.Context,
		_ repository.Queryer,
		_ uuid.UUID,
		_ *string,
	) (*models.MemoryCurationSuggestion, error) {
		return &models.MemoryCurationSuggestion{
			SuggestionID:   suggestionID,
			ProjectID:      "engram-vault",
			SuggestionType: testCase.suggestionType,
			PayloadJSON:    testCase.payload,
		}, nil
	}
	service.deps.applyMemoryCurationSuggestionAction = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MemoryCurationSuggestionActionInput,
	) (*models.MemoryCurationSuggestion, error) {
		actionCalled = true
		return nil, nil
	}

	_, err := service.ActionMemoryCurationSuggestion(
		context.Background(),
		suggestionID,
		actorUserID,
		MemoryCurationSuggestionActionRequest{
			Status: models.MemoryCurationSuggestionStatusApplied,
		},
	)
	if !errors.Is(err, testCase.expectedErr) {
		t.Fatalf("expected %v, got %v", testCase.expectedErr, err)
	}
	if actionCalled {
		t.Fatalf("expected curation status update to be skipped for rejected apply side effect")
	}
}

func configureAppliedCurationSuggestion(
	service *Service,
	input appliedCurationSuggestionSetupInput,
) {
	service.deps.getMemoryCurationSuggestion = func(
		_ context.Context,
		_ repository.Queryer,
		_ uuid.UUID,
		_ *string,
	) (*models.MemoryCurationSuggestion, error) {
		return &models.MemoryCurationSuggestion{
			SuggestionID:   input.suggestionID,
			ProjectID:      input.projectID,
			SuggestionType: input.suggestionType,
			PayloadJSON: map[string]any{
				input.payloadKey: input.relatedID.String(),
			},
		}, nil
	}
	service.deps.applyMemoryCurationSuggestionAction = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MemoryCurationSuggestionActionInput,
	) (*models.MemoryCurationSuggestion, error) {
		*input.actionCalled = true
		return &models.MemoryCurationSuggestion{
			SuggestionID: input.suggestionID,
			ProjectID:    input.projectID,
			Status:       models.MemoryCurationSuggestionStatusApplied,
		}, nil
	}
}

func assertAppliedCurationSuggestionStatus(
	t *testing.T,
	updated *models.MemoryCurationSuggestion,
) {
	t.Helper()
	if updated == nil {
		t.Fatalf("expected updated memory curation suggestion")
	}
	requireEqual(t, models.MemoryCurationSuggestionStatusApplied, updated.Status)
}
