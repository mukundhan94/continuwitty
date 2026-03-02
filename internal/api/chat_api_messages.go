package api

import (
	"net/http"

	"engram/internal/chat"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
)

const (
	defaultChatMessageListLimit  = 200
	minChatMessageListLimit      = 1
	maxChatMessageListLimit      = 500
	defaultChatMessageListOffset = 0
)

func sendMessageHandler(
	messageService ChatStreamService,
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
		sent, err := handleChatServiceError(func() (chat.ChatSendResponse, error) {
			return messageService.SendMessage(request.Context(), actorID, sessionID, payload)
		})
		if err != nil {
			writeChatRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusCreated, sent)
	}
}

func listMessagesHandler(
	sessionService ChatSessionService,
	requireAPIActor ChatActorResolver,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorID, sessionID, err := resolveChatRouteActorAndSession(request, requireAPIActor)
		if err != nil {
			writeChatError(writer, http.StatusUnauthorized, "Unauthorized")
			return
		}
		limit, err := parseBoundedIntQueryParam(
			request.URL.Query().Get("limit"),
			defaultChatMessageListLimit,
			minChatMessageListLimit,
			maxChatMessageListLimit,
		)
		if err != nil {
			writeChatError(writer, http.StatusBadRequest, "Invalid query parameters")
			return
		}
		offset, err := parseBoundedIntQueryParam(
			request.URL.Query().Get("offset"),
			defaultChatMessageListOffset,
			defaultChatMessageListOffset,
			int(^uint(0)>>1),
		)
		if err != nil {
			writeChatError(writer, http.StatusBadRequest, "Invalid query parameters")
			return
		}
		messages, err := handleChatServiceError(func() ([]models.ChatMessageRecord, error) {
			return sessionService.ListMessages(
				request.Context(),
				chat.SessionMessagesRequest{
					ActorUserID: actorID,
					SessionID:   sessionID,
					Limit:       limit,
					Offset:      offset,
				},
			)
		})
		if err != nil {
			writeChatRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, messages)
	}
}

func registerChatMessageRoutes(
	router chi.Router,
	sessionService ChatSessionService,
	messageService ChatStreamService,
	requireAPIActor ChatActorResolver,
) {
	if sessionService != nil {
		router.Get(
			"/api/v1/chat/sessions/{session_id}/messages",
			listMessagesHandler(sessionService, requireAPIActor),
		)
	}
	if messageService != nil {
		router.Post(
			"/api/v1/chat/sessions/{session_id}/messages",
			sendMessageHandler(messageService, requireAPIActor),
		)
		router.Post(
			"/api/v1/chat/sessions/{session_id}/messages/stream",
			streamMessageHandler(messageService, requireAPIActor),
		)
	}
}
