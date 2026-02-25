package chat

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestNormalizeSessionCreatePayloadAppliesAutosavePolicy(t *testing.T) {
	payload := models.ChatSessionCreateRequest{
		ProjectID:               "project-chat",
		Title:                   "Session",
		Provider:                models.ChatProviderOpenAI,
		ModelID:                 "gpt-4o-mini",
		SystemPrompt:            "",
		VisibilityScope:         models.VisibilityScopePrivate,
		AutosaveEnabled:         true,
		AutosaveStrategy:        models.ChatAutosaveStrategyOff,
		AutosaveIntervalMinutes: 30,
		AutosaveMinMessages:     6,
		RetentionDays:           30,
		RetentionMaxSnapshots:   60,
	}

	normalized := NormalizeSessionCreatePayload(payload)

	requireEqualAnyRuntime(t, true, normalized.AutosaveEnabled)
	requireEqualAnyRuntime(t, models.ChatAutosaveStrategyInterval, normalized.AutosaveStrategy)
}

func TestNormalizeSessionUpdatePayloadKeepsNonAutosaveUpdatesUntouched(t *testing.T) {
	title := "Renamed Session"
	normalized := NormalizeSessionUpdatePayload(models.ChatSessionUpdateRequest{Title: &title})
	if normalized.AutosaveEnabled != nil || normalized.AutosaveStrategy != nil {
		t.Fatalf("expected autosave pointers unchanged when autosave fields are absent")
	}
	requireEqualAnyRuntime(t, &title, normalized.Title)
}

func TestNormalizeSessionUpdatePayloadDerivesAutosaveFields(t *testing.T) {
	strategy := models.ChatAutosaveStrategyOff
	normalizedFromStrategy := NormalizeSessionUpdatePayload(
		models.ChatSessionUpdateRequest{AutosaveStrategy: &strategy},
	)
	if normalizedFromStrategy.AutosaveEnabled == nil {
		t.Fatalf("expected autosave_enabled to be derived")
	}
	requireEqualAnyRuntime(t, false, *normalizedFromStrategy.AutosaveEnabled)
	requireEqualAnyRuntime(t, models.ChatAutosaveStrategyOff, *normalizedFromStrategy.AutosaveStrategy)

	enabled := true
	normalizedFromEnabled := NormalizeSessionUpdatePayload(
		models.ChatSessionUpdateRequest{AutosaveEnabled: &enabled},
	)
	if normalizedFromEnabled.AutosaveStrategy == nil {
		t.Fatalf("expected autosave_strategy to be derived")
	}
	requireEqualAnyRuntime(t, models.ChatAutosaveStrategyInterval, *normalizedFromEnabled.AutosaveStrategy)
}

func TestGetSessionReturnsNotFoundErrorWhenMissing(t *testing.T) {
	service := sessionOperationsServiceForTest()
	service.deps.getChatSession = func(context.Context, repository.Queryer, repository.ChatSessionGetInput) (*models.ChatSessionRecord, error) {
		return nil, nil
	}

	_, err := service.GetSession(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-000000007301"),
		uuid.MustParse("00000000-0000-0000-0000-000000007302"),
	)
	serviceErr := requireChatServiceError(t, err)
	requireEqualIntRuntime(t, 404, serviceErr.StatusCode())
}

func TestCreateSessionEnsuresProjectAndCreatesSession(t *testing.T) {
	service := sessionOperationsServiceForTest()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007311")
	capturedEnsure := repository.ProjectEnsureInput{}
	capturedCreate := repository.ChatSessionCreateInput{}
	service.deps.ensureProjectExists = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ProjectEnsureInput,
	) (*models.ProjectRecord, error) {
		capturedEnsure = input
		return &models.ProjectRecord{ProjectID: input.ProjectID, OwnerUserID: input.OwnerUserID}, nil
	}
	service.deps.createChatSession = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ChatSessionCreateInput,
	) (*models.ChatSessionRecord, error) {
		capturedCreate = input
		record := sessionOperationsFixtureRecord(actorUserID)
		return &record, nil
	}
	payload := models.ChatSessionCreateRequest{
		ProjectID:               "project-chat",
		Title:                   "Session",
		Provider:                models.ChatProviderOpenAI,
		ModelID:                 "gpt-4o-mini",
		SystemPrompt:            "",
		VisibilityScope:         models.VisibilityScopePrivate,
		AutosaveEnabled:         true,
		AutosaveStrategy:        models.ChatAutosaveStrategyOff,
		AutosaveIntervalMinutes: 30,
		AutosaveMinMessages:     6,
		RetentionDays:           30,
		RetentionMaxSnapshots:   60,
	}

	created, err := service.CreateSession(context.Background(), actorUserID, payload)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	requireEqualAnyRuntime(t, actorUserID, capturedEnsure.OwnerUserID)
	requireEqualAnyRuntime(t, "project-chat", capturedEnsure.ProjectID)
	requireEqualAnyRuntime(t, actorUserID, capturedCreate.OwnerUserID)
	requireEqualAnyRuntime(t, models.ChatAutosaveStrategyInterval, capturedCreate.Payload.AutosaveStrategy)
	requireEqualAnyRuntime(t, actorUserID, created.OwnerUserID)
}

func TestUpdateLifecyclePolicyReturnsCurrentWhenPayloadIsEmpty(t *testing.T) {
	service := sessionOperationsServiceForTest()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007321")
	session := sessionOperationsFixtureRecord(actorUserID)
	updateCalled := false
	service.deps.getChatSession = func(context.Context, repository.Queryer, repository.ChatSessionGetInput) (*models.ChatSessionRecord, error) {
		copied := session
		return &copied, nil
	}
	service.deps.updateChatSession = func(context.Context, repository.Queryer, repository.ChatSessionUpdateInput) (*models.ChatSessionRecord, error) {
		updateCalled = true
		return nil, nil
	}

	policy, err := service.UpdateLifecyclePolicy(
		context.Background(),
		actorUserID,
		session.SessionID,
		ChatLifecyclePolicyUpdateRequest{},
	)
	if err != nil {
		t.Fatalf("update lifecycle policy: %v", err)
	}
	if updateCalled {
		t.Fatalf("expected empty lifecycle update to avoid update call")
	}
	requireEqualAnyRuntime(t, session.AutosaveStrategy, policy.AutosaveStrategy)
}

func TestUpdateLifecyclePolicyMapsPayloadToSessionUpdate(t *testing.T) {
	service := sessionOperationsServiceForTest()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007331")
	session := sessionOperationsFixtureRecord(actorUserID)
	capturedUpdate := repository.ChatSessionUpdateInput{}
	strategy := models.ChatAutosaveStrategyMessageCount
	interval := 45
	service.deps.updateChatSession = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ChatSessionUpdateInput,
	) (*models.ChatSessionRecord, error) {
		capturedUpdate = input
		updated := session
		updated.AutosaveEnabled = *input.Payload.AutosaveEnabled
		updated.AutosaveStrategy = *input.Payload.AutosaveStrategy
		updated.AutosaveIntervalMinutes = *input.Payload.AutosaveIntervalMinutes
		return &updated, nil
	}

	policy, err := service.UpdateLifecyclePolicy(
		context.Background(),
		actorUserID,
		session.SessionID,
		ChatLifecyclePolicyUpdateRequest{
			AutosaveStrategy:        &strategy,
			AutosaveIntervalMinutes: &interval,
		},
	)
	if err != nil {
		t.Fatalf("update lifecycle policy: %v", err)
	}

	requireEqualAnyRuntime(t, session.SessionID, capturedUpdate.SessionID)
	requireEqualAnyRuntime(t, actorUserID, capturedUpdate.ActorUserID)
	requireEqualAnyRuntime(t, models.ChatAutosaveStrategyMessageCount, *capturedUpdate.Payload.AutosaveStrategy)
	requireEqualAnyRuntime(t, models.ChatAutosaveStrategyMessageCount, policy.AutosaveStrategy)
	requireEqualAnyRuntime(t, 45, policy.AutosaveIntervalMinutes)
}

func sessionOperationsServiceForTest() *SessionOperationsService {
	service := &SessionOperationsService{db: nil, deps: defaultSessionOperationsDeps()}
	service.deps.ensureProjectExists = func(context.Context, repository.Queryer, repository.ProjectEnsureInput) (*models.ProjectRecord, error) {
		return &models.ProjectRecord{}, nil
	}
	service.deps.createChatSession = func(context.Context, repository.Queryer, repository.ChatSessionCreateInput) (*models.ChatSessionRecord, error) {
		record := sessionOperationsFixtureRecord(uuid.MustParse("00000000-0000-0000-0000-000000007399"))
		return &record, nil
	}
	service.deps.listChatSessions = func(context.Context, repository.Queryer, repository.ChatSessionListInput) ([]models.ChatSessionRecord, error) {
		return []models.ChatSessionRecord{}, nil
	}
	service.deps.getChatSession = func(context.Context, repository.Queryer, repository.ChatSessionGetInput) (*models.ChatSessionRecord, error) {
		record := sessionOperationsFixtureRecord(uuid.MustParse("00000000-0000-0000-0000-000000007399"))
		return &record, nil
	}
	service.deps.updateChatSession = func(context.Context, repository.Queryer, repository.ChatSessionUpdateInput) (*models.ChatSessionRecord, error) {
		record := sessionOperationsFixtureRecord(uuid.MustParse("00000000-0000-0000-0000-000000007399"))
		return &record, nil
	}
	return service
}

func sessionOperationsFixtureRecord(ownerUserID uuid.UUID) models.ChatSessionRecord {
	now := time.Now().UTC()
	return models.ChatSessionRecord{
		SessionID:               uuid.MustParse("00000000-0000-0000-0000-000000007390"),
		OwnerUserID:             ownerUserID,
		ProjectID:               "project-chat",
		Title:                   "Session",
		Provider:                models.ChatProviderOpenAI,
		ModelID:                 "gpt-4o-mini",
		SystemPrompt:            "be helpful",
		VisibilityScope:         models.VisibilityScopePrivate,
		AutosaveEnabled:         true,
		AutosaveStrategy:        models.ChatAutosaveStrategyInterval,
		AutosaveIntervalMinutes: 30,
		AutosaveMinMessages:     6,
		RetentionDays:           30,
		RetentionMaxSnapshots:   60,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}
