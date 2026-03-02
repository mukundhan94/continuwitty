package chat

import (
	"context"
	"strings"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestSaveSessionAsEngramSetsSourceSessionID(t *testing.T) {
	service := sessionOperationsServiceForTest()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007401")
	session := sessionOperationsFixtureRecord(actorUserID)
	captured := repository.CreateEngramInput{}
	service.deps.getChatSession = func(context.Context, repository.Queryer, repository.ChatSessionGetInput) (*models.ChatSessionRecord, error) {
		record := session
		return &record, nil
	}
	service.deps.listChatMessages = func(context.Context, repository.Queryer, repository.ChatMessageListInput) ([]models.ChatMessageRecord, error) {
		return []models.ChatMessageRecord{
			sessionMessageFixture(session.SessionID, "user", "Question"),
			sessionMessageFixture(session.SessionID, "assistant", "Answer"),
		}, nil
	}
	service.deps.createEngram = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.CreateEngramInput,
	) (*models.EngramCreateResponse, error) {
		captured = input
		return &models.EngramCreateResponse{
			EngramID:  uuid.MustParse("00000000-0000-0000-0000-000000007402"),
			CreatedAt: time.Now().UTC(),
		}, nil
	}

	response, err := service.SaveSessionAsEngram(
		context.Background(),
		actorUserID,
		session.SessionID,
		models.SaveSessionAsEngramRequest{
			Title:           "Saved Session",
			Abstract:        "Session summary",
			VisibilityScope: models.VisibilityScopeProject,
			Tags:            []string{"chat"},
			Keywords:        []string{"continuity"},
		},
	)
	if err != nil {
		t.Fatalf("save session as engram: %v", err)
	}
	requireEqualAnyRuntime(t, session.SessionID, response.SessionID)
	requireEqualIntRuntime(t, 256, captured.EmbeddingDim)
	if captured.OwnerUserID == nil {
		t.Fatalf("expected owner user id")
	}
	requireEqualAnyRuntime(t, actorUserID, *captured.OwnerUserID)
	requireEqualAnyRuntime(t, "chat.save_as_engram", captured.EnrichmentOrigin)
	if captured.Payload.SourceSessionID == nil {
		t.Fatalf("expected source session id")
	}
	requireEqualAnyRuntime(t, session.SessionID, *captured.Payload.SourceSessionID)
	requireEqualAnyRuntime(t, string(models.VisibilityScopeProject), captured.Payload.VisibilityScope)
	if captured.Payload.ThreadID == nil {
		t.Fatalf("expected thread id")
	}
	requireEqualAnyRuntime(t, "chat-session:"+session.SessionID.String(), *captured.Payload.ThreadID)
}

func TestSaveSessionAsEngramDerivesAbstractFromLatestAssistant(t *testing.T) {
	service := sessionOperationsServiceForTest()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007411")
	session := sessionOperationsFixtureRecord(actorUserID)
	assistantText := "Payment outage caused by DB saturation; rollback ineffective due to query-plan drift."
	captured := repository.CreateEngramInput{}
	service.deps.getChatSession = func(context.Context, repository.Queryer, repository.ChatSessionGetInput) (*models.ChatSessionRecord, error) {
		record := session
		return &record, nil
	}
	service.deps.listChatMessages = func(context.Context, repository.Queryer, repository.ChatMessageListInput) ([]models.ChatMessageRecord, error) {
		return []models.ChatMessageRecord{
			sessionMessageFixture(session.SessionID, "user", "Need a clean incident summary"),
			sessionMessageFixture(session.SessionID, "assistant", assistantText),
		}, nil
	}
	service.deps.createEngram = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.CreateEngramInput,
	) (*models.EngramCreateResponse, error) {
		captured = input
		return &models.EngramCreateResponse{
			EngramID:  uuid.MustParse("00000000-0000-0000-0000-000000007412"),
			CreatedAt: time.Now().UTC(),
		}, nil
	}

	_, err := service.SaveSessionAsEngram(
		context.Background(),
		actorUserID,
		session.SessionID,
		models.SaveSessionAsEngramRequest{
			Title:           "Saved Session",
			Abstract:        "Snapshot from active chat session.",
			VisibilityScope: models.VisibilityScopeProject,
			Tags:            []string{"chat"},
			Keywords:        []string{"continuity"},
		},
	)
	if err != nil {
		t.Fatalf("save session as engram: %v", err)
	}
	if !strings.HasPrefix(captured.Payload.Abstract, "Payment outage caused by DB saturation") {
		t.Fatalf("expected assistant-derived abstract, got %q", captured.Payload.Abstract)
	}
}

func TestSaveSessionAsEngramRejectsEmptySession(t *testing.T) {
	service := sessionOperationsServiceForTest()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007421")
	session := sessionOperationsFixtureRecord(actorUserID)
	service.deps.getChatSession = func(context.Context, repository.Queryer, repository.ChatSessionGetInput) (*models.ChatSessionRecord, error) {
		record := session
		return &record, nil
	}
	service.deps.listChatMessages = func(context.Context, repository.Queryer, repository.ChatMessageListInput) ([]models.ChatMessageRecord, error) {
		return []models.ChatMessageRecord{}, nil
	}

	_, err := service.SaveSessionAsEngram(
		context.Background(),
		actorUserID,
		session.SessionID,
		models.SaveSessionAsEngramRequest{
			Title:           "Saved Session",
			Abstract:        "Session summary",
			VisibilityScope: models.VisibilityScopeProject,
			Tags:            []string{"chat"},
			Keywords:        []string{"continuity"},
		},
	)
	serviceErr := requireChatServiceError(t, err)
	requireEqualIntRuntime(t, 400, serviceErr.StatusCode())
	requireEqualAnyRuntime(t, "Cannot save an empty chat session as engram", serviceErr.Detail())
}

func sessionMessageFixture(sessionID uuid.UUID, role string, content string) models.ChatMessageRecord {
	return models.ChatMessageRecord{
		MessageID:      uuid.New(),
		SessionID:      sessionID,
		Role:           role,
		ContentText:    content,
		TokenUsageJSON: map[string]any{},
		UsedEngramIDs:  []uuid.UUID{},
		CreatedAt:      time.Now().UTC(),
	}
}
