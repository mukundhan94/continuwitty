package chat

import (
	"context"
	"time"

	"engram/internal/models"
	"engram/internal/providers"

	"github.com/google/uuid"
)

type streamSuccessInput struct {
	ActorUserID       uuid.UUID
	TraceID           string
	Prepared          PreparedGeneration
	StreamResult      StreamChunkResult
	ProviderCandidate ProviderFallbackCandidate
	Events            []StreamEvent
	StreamStartedAt   time.Time
}

type streamPersistenceErrorInput struct {
	Err             error
	Events          []StreamEvent
	Provider        models.ChatProvider
	StreamEvents    []StreamEvent
	StreamStartedAt time.Time
}

type streamFailureInput struct {
	Events          []StreamEvent
	Provider        models.ChatProvider
	Outcome         string
	ErrorCode       string
	Detail          string
	StatusCode      int
	ChunkCount      int
	StreamStartedAt time.Time
}

func (service *ChatService) completeStreamSuccess(
	ctx context.Context,
	input streamSuccessInput,
) ([]StreamEvent, error) {
	assistantMessage, err := service.persistAssistantReplyWithTrace(
		ctx,
		persistAssistantInput{
			ActorUserID: input.ActorUserID,
			Prepared:    input.Prepared,
			Result:      streamAssistantResult(input.ProviderCandidate, input.StreamResult),
			TraceID:     input.TraceID,
			Operation:   chatOperationStream,
			Provider:    input.ProviderCandidate.Provider,
		},
	)
	if err != nil {
		return service.mapStreamPersistenceError(
			streamPersistenceErrorInput{
				Err:             err,
				Events:          input.Events,
				Provider:        input.ProviderCandidate.Provider,
				StreamEvents:    input.StreamResult.Events,
				StreamStartedAt: input.StreamStartedAt,
			},
		)
	}
	service.reinforceLinksWithTrace(ctx, reinforceTraceInput{
		ActorUserID: input.ActorUserID,
		Prepared:    input.Prepared,
		TraceID:     input.TraceID,
		Operation:   chatOperationStream,
		Provider:    input.ProviderCandidate.Provider,
	})
	if err := service.runLifecycleWithTrace(
		ctx,
		lifecycleRunInput{
			ActorUserID: input.ActorUserID,
			Prepared:    input.Prepared,
			TraceID:     input.TraceID,
			Operation:   chatOperationStream,
			Provider:    input.ProviderCandidate.Provider,
		},
	); err != nil {
		return nil, err
	}
	service.recordEngramAccessWithTrace(ctx, accessTraceInput{
		Prepared:  input.Prepared,
		TraceID:   input.TraceID,
		Operation: chatOperationStream,
		Provider:  input.ProviderCandidate.Provider,
	})
	input.Events = append(
		input.Events,
		streamDoneEvent(input.Prepared, *assistantMessage, input.StreamResult.FullText),
	)
	service.recordStreamCompletion(
		input.ProviderCandidate.Provider,
		input.StreamResult.Events,
		input.StreamStartedAt,
	)
	return input.Events, nil
}

func streamAssistantResult(
	providerCandidate ProviderFallbackCandidate,
	streamResult StreamChunkResult,
) providers.ProviderGenerateResult {
	return providers.ProviderGenerateResult{
		Provider:   providerCandidate.Provider,
		ModelID:    providerCandidate.ModelID,
		Text:       streamResult.FullText,
		TokenUsage: map[string]int{},
	}
}

func (service *ChatService) mapStreamPersistenceError(
	input streamPersistenceErrorInput,
) ([]StreamEvent, error) {
	serviceError, ok := input.Err.(*ChatServiceError)
	if !ok {
		return nil, input.Err
	}
	return service.appendStreamFailure(
		streamFailureInput{
			Events:          input.Events,
			Provider:        input.Provider,
			Outcome:         chatStreamOutcomePersistenceFail,
			ErrorCode:       "persistence_error",
			Detail:          serviceError.Detail(),
			StatusCode:      serviceError.StatusCode(),
			ChunkCount:      streamChunkCount(input.StreamEvents),
			StreamStartedAt: input.StreamStartedAt,
		},
	), nil
}

func streamDoneEvent(
	prepared PreparedGeneration,
	assistantMessage models.ChatMessageRecord,
	fullText string,
) StreamEvent {
	return StreamEvent{
		Type:    "done",
		Payload: BuildStreamDonePayload(prepared, assistantMessage, fullText, nil),
	}
}

func (service *ChatService) appendStreamFailure(input streamFailureInput) []StreamEvent {
	input.Events = append(
		input.Events,
		buildStreamErrorEvent(input.Detail, input.StatusCode, input.ErrorCode),
	)
	service.recordStreamHealth(
		StreamHealthSample{
			Provider:   input.Provider,
			Operation:  chatOperationStream,
			Outcome:    input.Outcome,
			ErrorCode:  input.ErrorCode,
			ChunkCount: input.ChunkCount,
			Duration:   service.nowUTC().Sub(input.StreamStartedAt),
		},
	)
	return input.Events
}

func (service *ChatService) recordStreamCompletion(
	provider models.ChatProvider,
	streamEvents []StreamEvent,
	streamStartedAt time.Time,
) {
	service.recordStreamHealth(
		StreamHealthSample{
			Provider:   provider,
			Operation:  chatOperationStream,
			Outcome:    chatStreamOutcomeCompleted,
			ChunkCount: streamChunkCount(streamEvents),
			Duration:   service.nowUTC().Sub(streamStartedAt),
		},
	)
}

func streamChunkCount(events []StreamEvent) int {
	count := 0
	for _, event := range events {
		if event.Type == "chunk" {
			count++
		}
	}
	return count
}

func buildStreamErrorEvent(detail string, statusCode int, errorCode string) StreamEvent {
	return StreamEvent{
		Type: "error",
		Payload: map[string]any{
			"detail":      detail,
			"status_code": statusCode,
			"error_code":  errorCode,
		},
	}
}
