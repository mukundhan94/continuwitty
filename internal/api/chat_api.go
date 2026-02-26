package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"engram/internal/chat"
	"engram/internal/models"

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

// ChatSessionDerivativeService captures save/continue session behavior.
type ChatSessionDerivativeService interface {
	SaveSessionAsEngram(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload models.SaveSessionAsEngramRequest,
	) (models.SaveSessionAsEngramResponse, error)
	ContinueSession(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload models.ContinueSessionRequest,
	) (models.ContinueSessionResponse, error)
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

func writeChatError(writer http.ResponseWriter, statusCode int, detail string) {
	writeJSON(writer, statusCode, map[string]string{"detail": detail})
}

func writeChatRouteError(writer http.ResponseWriter, err error) {
	var httpError *chatHTTPError
	if errors.As(err, &httpError) {
		writeChatError(writer, httpError.StatusCode(), httpError.Detail())
		return
	}
	writeChatError(writer, http.StatusInternalServerError, "Internal server error")
}

func resolveChatRouteActorAndSession(
	request *http.Request,
	requireAPIActor ChatActorResolver,
) (uuid.UUID, uuid.UUID, error) {
	sessionID, err := uuid.Parse(chi.URLParam(request, "session_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	actorID, err := actorUserID(request, requireAPIActor)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return actorID, sessionID, nil
}

func decodeChatPayload[T any](request *http.Request, payload *T) error {
	return json.NewDecoder(request.Body).Decode(payload)
}

func streamMessageHandler(
	chatService ChatStreamService,
	requireAPIActor ChatActorResolver,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorID, sessionID, err := resolveChatRouteActorAndSession(request, requireAPIActor)
		if err != nil {
			writeChatError(writer, http.StatusUnauthorized, "Unauthorized")
			return
		}
		payload := chat.ChatMessageCreateRequest{}
		if err := decodeChatPayload(request, &payload); err != nil {
			writeChatError(writer, http.StatusBadRequest, "Invalid payload")
			return
		}
		events, err := handleChatServiceError(func() ([]chat.StreamEvent, error) {
			return chatService.StreamMessageEvents(request.Context(), actorID, sessionID, payload)
		})
		if err != nil {
			writeChatRouteError(writer, err)
			return
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(streamSSEEvents(events))
	}
}

func handleChatCreatedRoute[Payload any, Result any](
	writer http.ResponseWriter,
	request *http.Request,
	requireAPIActor ChatActorResolver,
	operation func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload Payload,
	) (Result, error),
) {
	actorID, sessionID, err := resolveChatRouteActorAndSession(request, requireAPIActor)
	if err != nil {
		writeChatError(writer, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var payload Payload
	if err := decodeChatPayload(request, &payload); err != nil {
		writeChatError(writer, http.StatusBadRequest, "Invalid payload")
		return
	}
	result, err := handleChatServiceError(func() (Result, error) {
		return operation(request.Context(), actorID, sessionID, payload)
	})
	if err != nil {
		writeChatRouteError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, result)
}

func chatCreatedRouteHandler[Payload any, Result any](
	requireAPIActor ChatActorResolver,
	operation func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload Payload,
	) (Result, error),
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		handleChatCreatedRoute(
			writer,
			request,
			requireAPIActor,
			operation,
		)
	}
}

// CreateChatRouter mounts chat API routes for migration parity.
func CreateChatRouter(
	streamService ChatStreamService,
	sessionService ChatSessionDerivativeService,
	requireAPIActor ChatActorResolver,
) chi.Router {
	router := chi.NewRouter()
	if streamService != nil {
		router.Post(
			"/api/v1/chat/sessions/{session_id}/messages/stream",
			streamMessageHandler(streamService, requireAPIActor),
		)
	}
	if sessionService != nil {
		router.Post(
			"/api/v1/chat/sessions/{session_id}/save-engram",
			chatCreatedRouteHandler(
				requireAPIActor,
				sessionService.SaveSessionAsEngram,
			),
		)
		router.Post(
			"/api/v1/chat/sessions/{session_id}/continue",
			chatCreatedRouteHandler(
				requireAPIActor,
				sessionService.ContinueSession,
			),
		)
	}
	return router
}
