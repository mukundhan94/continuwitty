package chat

import (
	"context"

	"engram/internal/models"
	"engram/internal/providers"

	"github.com/google/uuid"
)

type prepareGenerationInput struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	Payload     ChatMessageCreateRequest
	TraceID     string
	Operation   string
}

type persistAssistantInput struct {
	ActorUserID uuid.UUID
	Prepared    PreparedGeneration
	Result      providers.ProviderGenerateResult
	TraceID     string
	Operation   string
	Provider    models.ChatProvider
}

type lifecycleRunInput struct {
	ActorUserID uuid.UUID
	Prepared    PreparedGeneration
	TraceID     string
	Operation   string
	Provider    models.ChatProvider
}

type preparedTraceInput struct {
	Prepared  PreparedGeneration
	TraceID   string
	Operation string
	Stage     string
	Provider  models.ChatProvider
	ErrorCode string
}

type preparedActionTraceInput struct {
	Prepared     PreparedGeneration
	TraceID      string
	Operation    string
	Provider     models.ChatProvider
	SuccessStage string
	FailureStage string
	FailureCode  string
	Action       func() error
}

func (service *ChatService) prepareGenerationWithTrace(
	ctx context.Context,
	input prepareGenerationInput,
) (PreparedGeneration, error) {
	startedAt := service.nowUTC()
	service.recordLifecycleTrace(traceRecordInput{
		TraceID:   input.TraceID,
		Operation: input.Operation,
		Stage:     chatTracePrepareStart,
	})
	prepared, err := service.runtime.PrepareGeneration(
		ctx,
		input.ActorUserID,
		input.SessionID,
		input.Payload,
	)
	if err != nil {
		service.recordLifecycleTrace(traceRecordInput{
			TraceID:   input.TraceID,
			Operation: input.Operation,
			Stage:     chatTracePrepareFailed,
			Duration:  service.nowUTC().Sub(startedAt),
		})
		return PreparedGeneration{}, err
	}
	service.recordLifecycleTrace(traceRecordInput{
		TraceID:   input.TraceID,
		Operation: input.Operation,
		Stage:     chatTracePrepareDone,
		Prepared:  &prepared,
		Provider:  prepared.Session.Provider,
		Duration:  service.nowUTC().Sub(startedAt),
	})
	return prepared, nil
}

func (service *ChatService) persistAssistantReplyWithTrace(
	ctx context.Context,
	input persistAssistantInput,
) (*models.ChatMessageRecord, error) {
	var assistantMessage *models.ChatMessageRecord
	err := service.runPreparedActionWithTrace(
		preparedActionTraceInput{
			Prepared:     input.Prepared,
			TraceID:      input.TraceID,
			Operation:    input.Operation,
			Provider:     input.Provider,
			SuccessStage: chatTracePersistOK,
			FailureStage: chatTracePersistFail,
			Action: func() error {
				record, actionErr := service.runtime.PersistAssistantReply(
					ctx,
					input.ActorUserID,
					input.Prepared,
					input.Result,
				)
				assistantMessage = record
				return actionErr
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return assistantMessage, nil
}

func (service *ChatService) runLifecycleWithTrace(
	ctx context.Context,
	input lifecycleRunInput,
) error {
	return service.runPreparedActionWithTrace(
		preparedActionTraceInput{
			Prepared:     input.Prepared,
			TraceID:      input.TraceID,
			Operation:    input.Operation,
			Provider:     input.Provider,
			SuccessStage: chatTraceLifecycleOK,
			FailureStage: chatTraceLifecycleFail,
			Action: func() error {
				return service.runSessionLifecycleMaintenance(
					ctx,
					input.ActorUserID,
					input.Prepared.Session,
				)
			},
		},
	)
}

func (service *ChatService) reinforceLinksWithTrace(ctx context.Context, input reinforceTraceInput) {
	if len(input.Prepared.Context.UsedEngramLinkIDs) == 0 {
		return
	}
	_ = service.runPreparedActionWithTrace(
		preparedActionTraceInput{
			Prepared:     input.Prepared,
			TraceID:      input.TraceID,
			Operation:    input.Operation,
			Provider:     input.Provider,
			SuccessStage: chatTraceReinforceOK,
			FailureStage: chatTraceReinforceFail,
			FailureCode:  "link_reinforce_error",
			Action: func() error {
				return service.reinforceEngramLinks(
					ctx,
					input.ActorUserID,
					input.Prepared.Context.UsedEngramLinkIDs,
				)
			},
		},
	)
}

func (service *ChatService) recordEngramAccessWithTrace(ctx context.Context, input accessTraceInput) {
	engramIDs := dedupeUUIDs(input.Prepared.Context.UsedEngramIDs)
	if len(engramIDs) == 0 {
		return
	}
	accessSource := resolveEngramAccessSource(input.Operation)
	_ = service.runPreparedActionWithTrace(
		preparedActionTraceInput{
			Prepared:     input.Prepared,
			TraceID:      input.TraceID,
			Operation:    input.Operation,
			Provider:     input.Provider,
			SuccessStage: chatTraceAccessOK,
			FailureStage: chatTraceAccessFail,
			FailureCode:  "engram_access_record_error",
			Action: func() error {
				return service.recordEngramAccess(
					ctx,
					input.Prepared.Session.SessionID,
					accessSource,
					engramIDs,
				)
			},
		},
	)
}

func resolveEngramAccessSource(operation string) string {
	switch operation {
	case chatOperationStream:
		return "chat_stream"
	default:
		return "chat_send"
	}
}

func dedupeUUIDs(values []uuid.UUID) []uuid.UUID {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(values))
	deduped := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	return deduped
}

func (service *ChatService) recordPreparedLifecycleTrace(input preparedTraceInput) {
	service.recordLifecycleTrace(traceRecordInput{
		TraceID:   input.TraceID,
		Operation: input.Operation,
		Stage:     input.Stage,
		Prepared:  &input.Prepared,
		Provider:  input.Provider,
		ErrorCode: input.ErrorCode,
	})
}

func (service *ChatService) runPreparedActionWithTrace(input preparedActionTraceInput) error {
	if err := input.Action(); err != nil {
		service.recordPreparedLifecycleTrace(
			preparedTraceInput{
				Prepared:  input.Prepared,
				TraceID:   input.TraceID,
				Operation: input.Operation,
				Stage:     input.FailureStage,
				Provider:  input.Provider,
				ErrorCode: input.FailureCode,
			},
		)
		return err
	}
	service.recordPreparedLifecycleTrace(
		preparedTraceInput{
			Prepared:  input.Prepared,
			TraceID:   input.TraceID,
			Operation: input.Operation,
			Stage:     input.SuccessStage,
			Provider:  input.Provider,
		},
	)
	return nil
}
