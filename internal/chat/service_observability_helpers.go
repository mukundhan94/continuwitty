package chat

import (
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

type traceRecordInput struct {
	TraceID   string
	Operation string
	Stage     string
	Prepared  *PreparedGeneration
	Provider  models.ChatProvider
	ErrorCode string
	Duration  time.Duration
}

func (service *ChatService) recordProviderFailure(
	provider models.ChatProvider,
	operation string,
	errorCode string,
) {
	service.observability.RecordProviderFailure(
		ProviderFailureSample{
			Provider:  provider,
			Operation: operation,
			ErrorCode: errorCode,
		},
	)
}

func (service *ChatService) recordStreamHealth(sample StreamHealthSample) {
	service.observability.RecordStreamHealth(sample)
}

func (service *ChatService) recordLifecycleTrace(input traceRecordInput) {
	sessionID := uuid.Nil
	messageID := uuid.Nil
	if input.Prepared != nil {
		sessionID = input.Prepared.Session.SessionID
		messageID = input.Prepared.UserMessage.MessageID
	}
	service.observability.RecordLifecycleTrace(
		LifecycleTraceSample{
			TraceID:   input.TraceID,
			Operation: input.Operation,
			Stage:     input.Stage,
			Provider:  input.Provider,
			SessionID: sessionID,
			MessageID: messageID,
			ErrorCode: input.ErrorCode,
			Duration:  input.Duration,
		},
	)
}
