package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"engram/internal/chat"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	defaultChatSessionListLimit  = 50
	minChatSessionListLimit      = 1
	maxChatSessionListLimit      = 200
	defaultChatSessionListOffset = 0
)

var errInvalidChatQueryParam = errors.New("invalid chat query parameter")

// ChatSessionService captures chat session CRUD behavior used by API routes.
type ChatSessionService interface {
	CreateSession(
		ctx context.Context,
		actorUserID uuid.UUID,
		payload models.ChatSessionCreateRequest,
	) (*models.ChatSessionRecord, error)
	ListSessions(
		ctx context.Context,
		request chat.SessionListRequest,
	) ([]models.ChatSessionRecord, error)
	GetSession(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
	) (*models.ChatSessionRecord, error)
	UpdateSession(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload models.ChatSessionUpdateRequest,
	) (*models.ChatSessionRecord, error)
	GetLifecyclePolicy(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
	) (chat.ChatLifecyclePolicy, error)
	UpdateLifecyclePolicy(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload chat.ChatLifecyclePolicyUpdateRequest,
	) (chat.ChatLifecyclePolicy, error)
	ListTimelineEvents(
		ctx context.Context,
		request chat.SessionTimelineRequest,
	) ([]models.ChatTimelineEvent, error)
	ListMessages(
		ctx context.Context,
		request chat.SessionMessagesRequest,
	) ([]models.ChatMessageRecord, error)
}

type chatSessionListQuery struct {
	projectID *string
	limit     int
	offset    int
}

func parseBoundedIntQueryParam(
	rawValue string,
	defaultValue int,
	minimumValue int,
	maximumValue int,
) (int, error) {
	if rawValue == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, errInvalidChatQueryParam
	}
	if parsed < minimumValue || parsed > maximumValue {
		return 0, errInvalidChatQueryParam
	}
	return parsed, nil
}

func parseChatSessionListQuery(request *http.Request) (chatSessionListQuery, error) {
	limit, err := parseBoundedIntQueryParam(
		request.URL.Query().Get("limit"),
		defaultChatSessionListLimit,
		minChatSessionListLimit,
		maxChatSessionListLimit,
	)
	if err != nil {
		return chatSessionListQuery{}, err
	}
	offset, err := parseBoundedIntQueryParam(
		request.URL.Query().Get("offset"),
		defaultChatSessionListOffset,
		defaultChatSessionListOffset,
		int(^uint(0)>>1),
	)
	if err != nil {
		return chatSessionListQuery{}, err
	}
	var projectID *string
	if rawProjectID := request.URL.Query().Get("project_id"); rawProjectID != "" {
		projectID = &rawProjectID
	}
	return chatSessionListQuery{
		projectID: projectID,
		limit:     limit,
		offset:    offset,
	}, nil
}

func listSessionsHandler(
	sessionService ChatSessionService,
	requireAPIActor ChatActorResolver,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorID, err := actorUserID(request, requireAPIActor)
		if err != nil {
			writeChatError(writer, http.StatusUnauthorized, "Unauthorized")
			return
		}
		query, err := parseChatSessionListQuery(request)
		if err != nil {
			writeChatError(writer, http.StatusBadRequest, "Invalid query parameters")
			return
		}
		listed, err := sessionService.ListSessions(
			request.Context(),
			chat.SessionListRequest{
				ActorUserID: actorID,
				ProjectID:   query.projectID,
				Limit:       query.limit,
				Offset:      query.offset,
			},
		)
		if err != nil {
			writeChatRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, listed)
	}
}

func getSessionHandler(
	sessionService ChatSessionService,
	requireAPIActor ChatActorResolver,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorID, sessionID, err := resolveChatRouteActorAndSession(request, requireAPIActor)
		if err != nil {
			writeChatError(writer, http.StatusUnauthorized, "Unauthorized")
			return
		}
		session, err := handleChatServiceError(func() (*models.ChatSessionRecord, error) {
			return sessionService.GetSession(request.Context(), actorID, sessionID)
		})
		if err != nil {
			writeChatRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, session)
	}
}

type chatRouteResolver func(
	request *http.Request,
	requireAPIActor ChatActorResolver,
) (uuid.UUID, uuid.UUID, error)

func resolveChatRouteActor(
	request *http.Request,
	requireAPIActor ChatActorResolver,
) (uuid.UUID, uuid.UUID, error) {
	actorID, err := actorUserID(request, requireAPIActor)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return actorID, uuid.Nil, nil
}

func chatPayloadRouteHandler[Payload any, Result any](
	operation func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload Payload,
	) (Result, error),
	requireAPIActor ChatActorResolver,
	resolveRouteContext chatRouteResolver,
	successStatusCode int,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorID, sessionID, err := resolveRouteContext(request, requireAPIActor)
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
		writeJSON(writer, successStatusCode, result)
	}
}

func registerChatSessionRoutes(
	router chi.Router,
	sessionService ChatSessionService,
	requireAPIActor ChatActorResolver,
) {
	if sessionService == nil {
		return
	}
	router.Post(
		"/api/v1/chat/sessions",
		chatPayloadRouteHandler(
			func(
				ctx context.Context,
				actorUserID uuid.UUID,
				_ uuid.UUID,
				payload models.ChatSessionCreateRequest,
			) (*models.ChatSessionRecord, error) {
				return sessionService.CreateSession(ctx, actorUserID, payload)
			},
			requireAPIActor,
			resolveChatRouteActor,
			http.StatusCreated,
		),
	)
	router.Get("/api/v1/chat/sessions", listSessionsHandler(sessionService, requireAPIActor))
	router.Get("/api/v1/chat/sessions/{session_id}", getSessionHandler(sessionService, requireAPIActor))
	router.Patch(
		"/api/v1/chat/sessions/{session_id}",
		chatPayloadRouteHandler(
			sessionService.UpdateSession,
			requireAPIActor,
			resolveChatRouteActorAndSession,
			http.StatusOK,
		),
	)
	registerChatLifecycleRoutes(router, sessionService, requireAPIActor)
}
