package api

import (
	"context"
	"net/http"

	"engram/internal/chat"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type pinnedListOperation[ListResult any] func(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
) (ListResult, error)

type pinnedPinOperation[PinPayload any, PinResult any] func(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload PinPayload,
) (PinResult, error)

type pinnedUnpinOperation func(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	resourceID uuid.UUID,
) error

type pinnedResourceSpec[PinPayload any, PinResult any, ListResult any] struct {
	listPath       string
	pinPath        string
	unpinPath      string
	unpinParam     string
	listOperation  pinnedListOperation[ListResult]
	pinOperation   pinnedPinOperation[PinPayload, PinResult]
	unpinOperation pinnedUnpinOperation
}

func parseChatRouteUUIDParam(request *http.Request, parameterName string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(request, parameterName))
}

func buildPinnedListHandler[ListResult any](
	requireAPIActor ChatActorResolver,
	operation pinnedListOperation[ListResult],
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorID, sessionID, err := resolveChatRouteActorAndSession(request, requireAPIActor)
		if err != nil {
			writeChatError(writer, http.StatusUnauthorized, "Unauthorized")
			return
		}
		listed, err := handleChatServiceError(func() (ListResult, error) {
			return operation(request.Context(), actorID, sessionID)
		})
		if err != nil {
			writeChatRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, listed)
	}
}

func buildPinnedUnpinHandler(
	requireAPIActor ChatActorResolver,
	parameterName string,
	operation pinnedUnpinOperation,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorID, sessionID, err := resolveChatRouteActorAndSession(request, requireAPIActor)
		if err != nil {
			writeChatError(writer, http.StatusUnauthorized, "Unauthorized")
			return
		}
		resourceID, err := parseChatRouteUUIDParam(request, parameterName)
		if err != nil {
			writeChatError(writer, http.StatusBadRequest, "Invalid resource id")
			return
		}
		_, err = handleChatServiceError(func() (struct{}, error) {
			return struct{}{}, operation(request.Context(), actorID, sessionID, resourceID)
		})
		if err != nil {
			writeChatRouteError(writer, err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}
}

func registerPinnedResourceRoutes[PinPayload any, PinResult any, ListResult any](
	router chi.Router,
	requireAPIActor ChatActorResolver,
	spec pinnedResourceSpec[PinPayload, PinResult, ListResult],
) {
	router.Get(spec.listPath, buildPinnedListHandler(requireAPIActor, spec.listOperation))
	router.Post(
		spec.pinPath,
		chatPayloadRouteHandler(
			spec.pinOperation,
			requireAPIActor,
			resolveChatRouteActorAndSession,
			http.StatusOK,
		),
	)
	router.Delete(
		spec.unpinPath,
		buildPinnedUnpinHandler(requireAPIActor, spec.unpinParam, spec.unpinOperation),
	)
}

func buildPinnedSpec[PinPayload any, PinRequest any, PinResult any, ListResult any](
	listPath string,
	pinPath string,
	unpinPath string,
	unpinParam string,
	listOperation pinnedListOperation[ListResult],
	resourceIDFromPayload func(payload PinPayload) uuid.UUID,
	buildPinRequest func(actorUserID uuid.UUID, sessionID uuid.UUID, resourceID uuid.UUID) PinRequest,
	pinRequestOperation func(ctx context.Context, request PinRequest) (PinResult, error),
	unpinRequestOperation func(ctx context.Context, request PinRequest) error,
) pinnedResourceSpec[PinPayload, PinResult, ListResult] {
	return pinnedResourceSpec[PinPayload, PinResult, ListResult]{
		listPath:      listPath,
		pinPath:       pinPath,
		unpinPath:     unpinPath,
		unpinParam:    unpinParam,
		listOperation: listOperation,
		pinOperation: func(
			ctx context.Context,
			actorUserID uuid.UUID,
			sessionID uuid.UUID,
			payload PinPayload,
		) (PinResult, error) {
			request := buildPinRequest(actorUserID, sessionID, resourceIDFromPayload(payload))
			return pinRequestOperation(ctx, request)
		},
		unpinOperation: func(
			ctx context.Context,
			actorUserID uuid.UUID,
			sessionID uuid.UUID,
			resourceID uuid.UUID,
		) error {
			request := buildPinRequest(actorUserID, sessionID, resourceID)
			return unpinRequestOperation(ctx, request)
		},
	}
}

func registerChatPinnedRoutes(
	router chi.Router,
	sessionService ChatSessionService,
	requireAPIActor ChatActorResolver,
) {
	registerPinnedResourceRoutes(
		router,
		requireAPIActor,
		buildPinnedSpec(
			"/api/v1/chat/sessions/{session_id}/engrams",
			"/api/v1/chat/sessions/{session_id}/engrams/pin",
			"/api/v1/chat/sessions/{session_id}/engrams/{engram_id}",
			"engram_id",
			sessionService.ListPinnedEngrams,
			func(payload models.PinEngramRequest) uuid.UUID {
				return payload.EngramID
			},
			func(actorUserID uuid.UUID, sessionID uuid.UUID, engramID uuid.UUID) chat.SessionPinEngramRequest {
				return chat.SessionPinEngramRequest{
					ActorUserID: actorUserID,
					SessionID:   sessionID,
					EngramID:    engramID,
				}
			},
			sessionService.PinEngram,
			sessionService.UnpinEngram,
		),
	)
	registerPinnedResourceRoutes(
		router,
		requireAPIActor,
		buildPinnedSpec(
			"/api/v1/chat/sessions/{session_id}/documents",
			"/api/v1/chat/sessions/{session_id}/documents/pin",
			"/api/v1/chat/sessions/{session_id}/documents/{document_id}",
			"document_id",
			sessionService.ListPinnedDocuments,
			func(payload models.PinDocumentRequest) uuid.UUID {
				return payload.DocumentID
			},
			func(actorUserID uuid.UUID, sessionID uuid.UUID, documentID uuid.UUID) chat.SessionPinDocumentRequest {
				return chat.SessionPinDocumentRequest{
					ActorUserID: actorUserID,
					SessionID:   sessionID,
					DocumentID:  documentID,
				}
			},
			sessionService.PinDocument,
			sessionService.UnpinDocument,
		),
	)
}
