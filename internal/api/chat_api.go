package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"engram/internal/chat"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var errInvalidChatActor = errors.New("invalid chat actor")

// ChatActorResolver resolves the authenticated API actor for chat routes.
type ChatActorResolver func(request *http.Request) (map[string]any, error)

// ChatStreamService captures stream-message behavior used by chat routes.
type ChatStreamService interface {
	StreamMessageEvents(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload chat.ChatMessageCreateRequest,
	) ([]chat.StreamEvent, error)
}

type chatHTTPError struct {
	statusCode int
	detail     string
}

func (err *chatHTTPError) Error() string {
	return err.detail
}

func (err *chatHTTPError) StatusCode() int {
	return err.statusCode
}

func (err *chatHTTPError) Detail() string {
	return err.detail
}

func toChatHTTPError(serviceError *chat.ChatServiceError) *chatHTTPError {
	return &chatHTTPError{statusCode: serviceError.StatusCode(), detail: serviceError.Detail()}
}

func sseEvent(event string, payload map[string]any) string {
	encoded, _ := json.Marshal(payload)
	return "event: " + event + "\ndata: " + string(encoded) + "\n\n"
}

func actorUserID(request *http.Request, resolver ChatActorResolver) (uuid.UUID, error) {
	actor, err := resolver(request)
	if err != nil {
		return uuid.Nil, err
	}
	rawUserID, ok := actor["user_id"]
	if !ok {
		return uuid.Nil, errInvalidChatActor
	}
	userIDText, ok := rawUserID.(string)
	if !ok {
		return uuid.Nil, errInvalidChatActor
	}
	parsedUserID, err := uuid.Parse(userIDText)
	if err != nil {
		return uuid.Nil, errInvalidChatActor
	}
	return parsedUserID, nil
}

func handleChatServiceError[T any](operation func() (T, error)) (T, error) {
	result, err := operation()
	if err == nil {
		return result, nil
	}
	var serviceError *chat.ChatServiceError
	if errors.As(err, &serviceError) {
		var zero T
		return zero, toChatHTTPError(serviceError)
	}
	var zero T
	return zero, err
}

func streamSSEEvents(events []chat.StreamEvent) []byte {
	encoded := make([]byte, 0)
	for _, event := range events {
		encoded = append(encoded, []byte(sseEvent(event.Type, event.Payload))...)
	}
	return encoded
}

// CreateChatRouter mounts the initial chat stream route baseline for migration parity.
func CreateChatRouter(chatService ChatStreamService, requireAPIActor ChatActorResolver) chi.Router {
	router := chi.NewRouter()
	router.Post("/api/v1/chat/sessions/{session_id}/messages/stream", func(writer http.ResponseWriter, request *http.Request) {
		sessionID, err := uuid.Parse(chi.URLParam(request, "session_id"))
		if err != nil {
			http.Error(writer, "Invalid session id", http.StatusBadRequest)
			return
		}
		actorID, err := actorUserID(request, requireAPIActor)
		if err != nil {
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}
		payload := chat.ChatMessageCreateRequest{}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(writer, "Invalid payload", http.StatusBadRequest)
			return
		}
		events, err := handleChatServiceError(func() ([]chat.StreamEvent, error) {
			return chatService.StreamMessageEvents(request.Context(), actorID, sessionID, payload)
		})
		if err != nil {
			var httpError *chatHTTPError
			if errors.As(err, &httpError) {
				http.Error(writer, httpError.Detail(), httpError.StatusCode())
				return
			}
			http.Error(writer, "Internal server error", http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(streamSSEEvents(events))
	})
	return router
}
