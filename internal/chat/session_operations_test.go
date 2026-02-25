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

func TestListMessagesRequiresVisibleSession(t *testing.T) {
	service := sessionOperationsServiceForTest()
	listCalled := false
	service.deps.getChatSession = func(context.Context, repository.Queryer, repository.ChatSessionGetInput) (*models.ChatSessionRecord, error) {
		return nil, nil
	}
	service.deps.listChatMessages = func(context.Context, repository.Queryer, repository.ChatMessageListInput) ([]models.ChatMessageRecord, error) {
		listCalled = true
		return []models.ChatMessageRecord{}, nil
	}

	_, err := service.ListMessages(
		context.Background(),
		SessionMessagesRequest{
			ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000007341"),
			SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000007342"),
			Limit:       100,
			Offset:      0,
		},
	)
	_ = requireChatServiceError(t, err)
	if listCalled {
		t.Fatalf("expected list messages to short-circuit on missing session")
	}
}

func TestPinDocumentReturnsPinnedRecord(t *testing.T) {
	service := sessionOperationsServiceForTest()
	expected := &models.PinnedDocumentRecord{
		SessionID:      uuid.MustParse("00000000-0000-0000-0000-000000007362"),
		DocumentID:     uuid.MustParse("00000000-0000-0000-0000-000000007363"),
		PinnedByUserID: uuid.MustParse("00000000-0000-0000-0000-000000007361"),
		CreatedAt:      time.Now().UTC(),
	}
	service.deps.pinDocument = func(context.Context, repository.Queryer, repository.ChatPinDocumentInput) (*models.PinnedDocumentRecord, error) {
		record := *expected
		return &record, nil
	}

	record, err := service.PinDocument(
		context.Background(),
		SessionPinDocumentRequest{
			ActorUserID: expected.PinnedByUserID,
			SessionID:   expected.SessionID,
			DocumentID:  expected.DocumentID,
		},
	)
	if err != nil {
		t.Fatalf("pin document: %v", err)
	}
	requireEqualAnyRuntime(t, expected.DocumentID, record.DocumentID)
}

func TestPinEngramReturnsValidationErrorWhenNotAccessible(t *testing.T) {
	service := sessionOperationsServiceForTest()
	service.deps.pinEngram = func(
		context.Context,
		repository.Queryer,
		repository.ChatPinEngramInput,
	) (*models.PinnedEngramRecord, error) {
		return nil, nil
	}

	_, err := service.PinEngram(
		context.Background(),
		SessionPinEngramRequest{
			ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000007351"),
			SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000007352"),
			EngramID:    uuid.MustParse("00000000-0000-0000-0000-000000007353"),
		},
	)
	serviceErr := requireChatServiceError(t, err)
	requireEqualIntRuntime(t, 400, serviceErr.StatusCode())
}

func TestUnpinReturnsNotFoundWhenMissing(t *testing.T) {
	testCases := []struct {
		name           string
		configure      func(*SessionOperationsService)
		invoke         func(*SessionOperationsService) error
		expectedDetail string
	}{
		{
			name: "engram",
			configure: func(service *SessionOperationsService) {
				service.deps.unpinEngram = func(context.Context, repository.Queryer, repository.ChatPinEngramInput) (bool, error) {
					return false, nil
				}
			},
			invoke: func(service *SessionOperationsService) error {
				return service.UnpinEngram(
					context.Background(),
					SessionPinEngramRequest{
						ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000007371"),
						SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000007372"),
						EngramID:    uuid.MustParse("00000000-0000-0000-0000-000000007373"),
					},
				)
			},
			expectedDetail: "Pinned engram not found for session",
		},
		{
			name: "document",
			configure: func(service *SessionOperationsService) {
				service.deps.unpinDocument = func(context.Context, repository.Queryer, repository.ChatPinDocumentInput) (bool, error) {
					return false, nil
				}
			},
			invoke: func(service *SessionOperationsService) error {
				return service.UnpinDocument(
					context.Background(),
					SessionPinDocumentRequest{
						ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000007381"),
						SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000007382"),
						DocumentID:  uuid.MustParse("00000000-0000-0000-0000-000000007383"),
					},
				)
			},
			expectedDetail: "Pinned document not found for session",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			service := sessionOperationsServiceForTest()
			testCase.configure(service)

			serviceErr := requireChatServiceError(t, testCase.invoke(service))
			requireEqualIntRuntime(t, 404, serviceErr.StatusCode())
			requireEqualAnyRuntime(t, testCase.expectedDetail, serviceErr.Detail())
		})
	}
}

func TestListPinnedEngramsAndDocuments(t *testing.T) {
	service := sessionOperationsServiceForTest()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007391")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000007392")
	service.deps.getChatSession = func(context.Context, repository.Queryer, repository.ChatSessionGetInput) (*models.ChatSessionRecord, error) {
		record := sessionOperationsFixtureRecord(actorUserID)
		record.SessionID = sessionID
		return &record, nil
	}
	service.deps.listPinnedEngrams = func(context.Context, repository.Queryer, repository.ChatPinnedListInput) ([]models.EngramSummary, error) {
		return []models.EngramSummary{{EngramID: uuid.MustParse("00000000-0000-0000-0000-000000007393"), ProjectID: "project-chat", Title: "Pinned", CreatedAt: time.Now().UTC()}}, nil
	}
	service.deps.listPinnedDocuments = func(context.Context, repository.Queryer, repository.ChatPinnedListInput) ([]models.PinnedDocumentRecord, error) {
		return []models.PinnedDocumentRecord{{SessionID: sessionID, DocumentID: uuid.MustParse("00000000-0000-0000-0000-000000007394"), PinnedByUserID: actorUserID, CreatedAt: time.Now().UTC()}}, nil
	}

	engrams, err := service.ListPinnedEngrams(context.Background(), actorUserID, sessionID)
	if err != nil {
		t.Fatalf("list pinned engrams: %v", err)
	}
	documents, err := service.ListPinnedDocuments(context.Background(), actorUserID, sessionID)
	if err != nil {
		t.Fatalf("list pinned documents: %v", err)
	}
	requireEqualIntRuntime(t, 1, len(engrams))
	requireEqualIntRuntime(t, 1, len(documents))
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
	service.deps.listChatMessages = func(context.Context, repository.Queryer, repository.ChatMessageListInput) ([]models.ChatMessageRecord, error) {
		return []models.ChatMessageRecord{}, nil
	}
	service.deps.listPinnedEngrams = func(context.Context, repository.Queryer, repository.ChatPinnedListInput) ([]models.EngramSummary, error) {
		return []models.EngramSummary{}, nil
	}
	service.deps.listPinnedDocuments = func(context.Context, repository.Queryer, repository.ChatPinnedListInput) ([]models.PinnedDocumentRecord, error) {
		return []models.PinnedDocumentRecord{}, nil
	}
	service.deps.pinEngram = func(context.Context, repository.Queryer, repository.ChatPinEngramInput) (*models.PinnedEngramRecord, error) {
		record := models.PinnedEngramRecord{SessionID: uuid.MustParse("00000000-0000-0000-0000-000000007390"), EngramID: uuid.MustParse("00000000-0000-0000-0000-000000007398"), PinnedByUserID: uuid.MustParse("00000000-0000-0000-0000-000000007399"), CreatedAt: time.Now().UTC()}
		return &record, nil
	}
	service.deps.pinDocument = func(context.Context, repository.Queryer, repository.ChatPinDocumentInput) (*models.PinnedDocumentRecord, error) {
		record := models.PinnedDocumentRecord{SessionID: uuid.MustParse("00000000-0000-0000-0000-000000007390"), DocumentID: uuid.MustParse("00000000-0000-0000-0000-000000007397"), PinnedByUserID: uuid.MustParse("00000000-0000-0000-0000-000000007399"), CreatedAt: time.Now().UTC()}
		return &record, nil
	}
	service.deps.unpinEngram = func(context.Context, repository.Queryer, repository.ChatPinEngramInput) (bool, error) {
		return true, nil
	}
	service.deps.unpinDocument = func(context.Context, repository.Queryer, repository.ChatPinDocumentInput) (bool, error) {
		return true, nil
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
