package api

import (
	"net/http"

	"engram/internal/chat"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
)

const (
	defaultChatTimelineLimit  = 100
	minChatTimelineLimit      = 1
	maxChatTimelineLimit      = 500
	defaultChatTimelineOffset = 0
)

func getLifecyclePolicyHandler(
	sessionService ChatSessionService,
	requireAPIActor ChatActorResolver,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorID, sessionID, err := resolveChatRouteActorAndSession(request, requireAPIActor)
		if err != nil {
			writeChatError(writer, http.StatusUnauthorized, "Unauthorized")
			return
		}
		policy, err := handleChatServiceError(func() (chat.ChatLifecyclePolicy, error) {
			return sessionService.GetLifecyclePolicy(request.Context(), actorID, sessionID)
		})
		if err != nil {
			writeChatRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, policy)
	}
}

func updateLifecyclePolicyHandler(
	sessionService ChatSessionService,
	requireAPIActor ChatActorResolver,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorID, sessionID, err := resolveChatRouteActorAndSession(request, requireAPIActor)
		if err != nil {
			writeChatError(writer, http.StatusUnauthorized, "Unauthorized")
			return
		}
		payload := chat.ChatLifecyclePolicyUpdateRequest{}
		if err := decodeChatPayload(request, &payload); err != nil {
			writeChatError(writer, http.StatusBadRequest, "Invalid payload")
			return
		}
		updated, err := handleChatServiceError(func() (chat.ChatLifecyclePolicy, error) {
			return sessionService.UpdateLifecyclePolicy(request.Context(), actorID, sessionID, payload)
		})
		if err != nil {
			writeChatRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, updated)
	}
}

func listTimelineEventsHandler(
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
			defaultChatTimelineLimit,
			minChatTimelineLimit,
			maxChatTimelineLimit,
		)
		if err != nil {
			writeChatError(writer, http.StatusBadRequest, "Invalid query parameters")
			return
		}
		offset, err := parseBoundedIntQueryParam(
			request.URL.Query().Get("offset"),
			defaultChatTimelineOffset,
			defaultChatTimelineOffset,
			int(^uint(0)>>1),
		)
		if err != nil {
			writeChatError(writer, http.StatusBadRequest, "Invalid query parameters")
			return
		}
		events, err := handleChatServiceError(func() ([]models.ChatTimelineEvent, error) {
			return sessionService.ListTimelineEvents(
				request.Context(),
				chat.SessionTimelineRequest{
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
		writeJSON(writer, http.StatusOK, events)
	}
}

func registerChatLifecycleRoutes(
	router chi.Router,
	sessionService ChatSessionService,
	requireAPIActor ChatActorResolver,
) {
	router.Get(
		"/api/v1/chat/sessions/{session_id}/lifecycle-policy",
		getLifecyclePolicyHandler(sessionService, requireAPIActor),
	)
	router.Patch(
		"/api/v1/chat/sessions/{session_id}/lifecycle-policy",
		updateLifecyclePolicyHandler(sessionService, requireAPIActor),
	)
	router.Get(
		"/api/v1/chat/sessions/{session_id}/timeline",
		listTimelineEventsHandler(sessionService, requireAPIActor),
	)
}
