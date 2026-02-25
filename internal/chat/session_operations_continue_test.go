package chat

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

type continueSessionCopyFixture struct {
	ActorUserID uuid.UUID
	Session     models.ChatSessionRecord
	Continued   models.ChatSessionRecord
	EngramIDs   []uuid.UUID
	DocumentID  uuid.UUID
}

func TestContinueSessionCopiesPinnedEngramsAndDocuments(t *testing.T) {
	service := sessionOperationsServiceForTest()
	fixture := continueSessionCopyFixture{
		ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000007431"),
		Session:     sessionOperationsFixtureRecord(uuid.MustParse("00000000-0000-0000-0000-000000007431")),
		Continued:   sessionOperationsFixtureRecord(uuid.MustParse("00000000-0000-0000-0000-000000007431")),
		EngramIDs: []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000007433"),
			uuid.MustParse("00000000-0000-0000-0000-000000007434"),
		},
		DocumentID: uuid.MustParse("00000000-0000-0000-0000-000000007435"),
	}
	fixture.Continued.SessionID = uuid.MustParse("00000000-0000-0000-0000-000000007432")
	copiedDocumentIDs := []uuid.UUID{}
	installContinueSessionCopyFixture(service, fixture, &copiedDocumentIDs)

	title := "Continued Session"
	response, err := service.ContinueSession(
		context.Background(),
		fixture.ActorUserID,
		fixture.Session.SessionID,
		models.ContinueSessionRequest{Title: &title},
	)
	if err != nil {
		t.Fatalf("continue session: %v", err)
	}
	requireEqualAnyRuntime(t, "Continued Session", response.Session.Title)
	requireEqualAnyRuntime(t, fixture.EngramIDs, response.CarriedEngramIDs)
	requireEqualAnyRuntime(t, []uuid.UUID{fixture.DocumentID}, copiedDocumentIDs)
}

func TestContinueSessionUsesDefaultTitleWhenBlank(t *testing.T) {
	service := sessionOperationsServiceForTest()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007441")
	session := sessionOperationsFixtureRecord(actorUserID)
	continued := sessionOperationsFixtureRecord(actorUserID)
	continued.SessionID = uuid.MustParse("00000000-0000-0000-0000-000000007442")
	service.deps.getChatSession = func(context.Context, repository.Queryer, repository.ChatSessionGetInput) (*models.ChatSessionRecord, error) {
		record := session
		return &record, nil
	}
	service.deps.createChatSession = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ChatSessionCreateInput,
	) (*models.ChatSessionRecord, error) {
		record := continued
		record.Title = input.Payload.Title
		return &record, nil
	}
	service.deps.listPinnedEngrams = func(context.Context, repository.Queryer, repository.ChatPinnedListInput) ([]models.EngramSummary, error) {
		return []models.EngramSummary{}, nil
	}
	service.deps.listPinnedDocuments = func(context.Context, repository.Queryer, repository.ChatPinnedListInput) ([]models.PinnedDocumentRecord, error) {
		return []models.PinnedDocumentRecord{}, nil
	}

	blankTitle := "   "
	response, err := service.ContinueSession(
		context.Background(),
		actorUserID,
		session.SessionID,
		models.ContinueSessionRequest{Title: &blankTitle},
	)
	if err != nil {
		t.Fatalf("continue session: %v", err)
	}
	requireEqualAnyRuntime(t, session.Title+" (continued)", response.Session.Title)
}

func installContinueSessionCopyFixture(
	service *SessionOperationsService,
	fixture continueSessionCopyFixture,
	copiedDocumentIDs *[]uuid.UUID,
) {
	service.deps.getChatSession = func(context.Context, repository.Queryer, repository.ChatSessionGetInput) (*models.ChatSessionRecord, error) {
		record := fixture.Session
		return &record, nil
	}
	service.deps.createChatSession = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ChatSessionCreateInput,
	) (*models.ChatSessionRecord, error) {
		record := fixture.Continued
		record.Title = input.Payload.Title
		record.ProjectID = input.Payload.ProjectID
		record.Provider = input.Payload.Provider
		record.ModelID = input.Payload.ModelID
		record.SystemPrompt = input.Payload.SystemPrompt
		record.VisibilityScope = input.Payload.VisibilityScope
		record.AutosaveEnabled = input.Payload.AutosaveEnabled
		record.AutosaveStrategy = input.Payload.AutosaveStrategy
		record.AutosaveIntervalMinutes = input.Payload.AutosaveIntervalMinutes
		record.AutosaveMinMessages = input.Payload.AutosaveMinMessages
		record.RetentionDays = input.Payload.RetentionDays
		record.RetentionMaxSnapshots = input.Payload.RetentionMaxSnapshots
		return &record, nil
	}
	service.deps.listPinnedEngrams = func(context.Context, repository.Queryer, repository.ChatPinnedListInput) ([]models.EngramSummary, error) {
		return []models.EngramSummary{
			{EngramID: fixture.EngramIDs[0], ProjectID: fixture.Session.ProjectID, Title: "Pinned A", CreatedAt: time.Now().UTC()},
			{EngramID: fixture.EngramIDs[1], ProjectID: fixture.Session.ProjectID, Title: "Pinned B", CreatedAt: time.Now().UTC()},
		}, nil
	}
	service.deps.pinEngram = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ChatPinEngramInput,
	) (*models.PinnedEngramRecord, error) {
		return &models.PinnedEngramRecord{
			SessionID:      fixture.Continued.SessionID,
			EngramID:       input.EngramID,
			PinnedByUserID: fixture.ActorUserID,
			CreatedAt:      time.Now().UTC(),
		}, nil
	}
	service.deps.listPinnedDocuments = func(context.Context, repository.Queryer, repository.ChatPinnedListInput) ([]models.PinnedDocumentRecord, error) {
		return []models.PinnedDocumentRecord{
			{
				SessionID:      fixture.Session.SessionID,
				DocumentID:     fixture.DocumentID,
				PinnedByUserID: fixture.ActorUserID,
				CreatedAt:      time.Now().UTC(),
			},
		}, nil
	}
	service.deps.pinDocument = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ChatPinDocumentInput,
	) (*models.PinnedDocumentRecord, error) {
		*copiedDocumentIDs = append(*copiedDocumentIDs, input.DocumentID)
		return &models.PinnedDocumentRecord{
			SessionID:      input.SessionID,
			DocumentID:     input.DocumentID,
			PinnedByUserID: fixture.ActorUserID,
			CreatedAt:      time.Now().UTC(),
		}, nil
	}
}
