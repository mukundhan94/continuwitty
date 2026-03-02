package api

import (
	"context"
	"net/http"

	"engram/internal/admin"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func mountMemoryAdminCollectionRoutes(memory chi.Router, service MemoryAdminService, requireAdminActor RequireAdminActor) {
	memory.Get("/collections", listMemoryAdminCollectionsRoute(service, requireAdminActor))
	memory.Post("/collections", createMemoryAdminCollectionRoute(service, requireAdminActor))
	memory.Patch(
		"/collections/{collection_id}",
		collectionRoute(
			requireAdminActor,
			http.StatusOK,
			func(writer http.ResponseWriter, request *http.Request, _ AdminActor) (collectionPathPayloadRouteInput[admin.CollectionUpdateRequest], bool) {
				pathID, ok := parsePathUUID(writer, request, "collection_id")
				if !ok {
					return collectionPathPayloadRouteInput[admin.CollectionUpdateRequest]{}, false
				}
				payload, ok := parseCollectionPayload[admin.CollectionUpdateRequest](writer, request)
				if !ok {
					return collectionPathPayloadRouteInput[admin.CollectionUpdateRequest]{}, false
				}
				return collectionPathPayloadRouteInput[admin.CollectionUpdateRequest]{pathID: pathID, payload: payload}, true
			},
			func(ctx context.Context, input collectionPathPayloadRouteInput[admin.CollectionUpdateRequest]) (any, error) {
				return service.UpdateCollection(ctx, input.pathID, input.payload)
			},
		),
	)
	memory.Delete(
		"/collections/{collection_id}",
		collectionRoute(
			requireAdminActor,
			http.StatusOK,
			func(writer http.ResponseWriter, request *http.Request, actor AdminActor) (collectionActorPathPayloadRouteInput[admin.CollectionDeleteRequest], bool) {
				pathID, ok := parsePathUUID(writer, request, "collection_id")
				if !ok {
					return collectionActorPathPayloadRouteInput[admin.CollectionDeleteRequest]{}, false
				}
				payload, ok := parseCollectionPayload[admin.CollectionDeleteRequest](writer, request)
				if !ok {
					return collectionActorPathPayloadRouteInput[admin.CollectionDeleteRequest]{}, false
				}
				return collectionActorPathPayloadRouteInput[admin.CollectionDeleteRequest]{
					actor:   actor,
					pathID:  pathID,
					payload: payload,
				}, true
			},
			func(ctx context.Context, input collectionActorPathPayloadRouteInput[admin.CollectionDeleteRequest]) (any, error) {
				return service.DeleteCollection(ctx, input.pathID, input.actor.UserID, input.payload)
			},
		),
	)
	memory.Post(
		"/collections/{collection_id}/items",
		collectionRoute(
			requireAdminActor,
			http.StatusOK,
			func(writer http.ResponseWriter, request *http.Request, actor AdminActor) (collectionActorPathPayloadRouteInput[admin.CollectionItemsUpdateRequest], bool) {
				pathID, ok := parsePathUUID(writer, request, "collection_id")
				if !ok {
					return collectionActorPathPayloadRouteInput[admin.CollectionItemsUpdateRequest]{}, false
				}
				payload, ok := parseCollectionPayload[admin.CollectionItemsUpdateRequest](writer, request)
				if !ok {
					return collectionActorPathPayloadRouteInput[admin.CollectionItemsUpdateRequest]{}, false
				}
				return collectionActorPathPayloadRouteInput[admin.CollectionItemsUpdateRequest]{
					actor:   actor,
					pathID:  pathID,
					payload: payload,
				}, true
			},
			func(ctx context.Context, input collectionActorPathPayloadRouteInput[admin.CollectionItemsUpdateRequest]) (any, error) {
				return service.AddCollectionItems(ctx, input.pathID, input.actor.UserID, input.payload)
			},
		),
	)
	memory.Delete("/collections/{collection_id}/items/{engram_id}", removeMemoryAdminCollectionItemRoute(service, requireAdminActor))
}

func listMemoryAdminCollectionsRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if _, ok := requireActor(writer, request, requireAdminActor); !ok {
			return
		}
		listRequest, ok := parseMemoryAdminListRequest(writer, request, false)
		if !ok {
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.ListCollections(request.Context(), listRequest)
		})
	}
}

type collectionCreateRouteInput struct {
	actor   AdminActor
	payload admin.CollectionCreateRequest
}

func createMemoryAdminCollectionRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return collectionRoute(
		requireAdminActor,
		http.StatusCreated,
		func(writer http.ResponseWriter, request *http.Request, actor AdminActor) (collectionCreateRouteInput, bool) {
			payload, ok := parseCollectionPayload[admin.CollectionCreateRequest](writer, request)
			if !ok {
				return collectionCreateRouteInput{}, false
			}
			return collectionCreateRouteInput{actor: actor, payload: payload}, true
		},
		func(ctx context.Context, input collectionCreateRouteInput) (any, error) {
			return service.CreateCollection(ctx, input.actor.UserID, input.actor.Role, input.payload)
		},
	)
}

type collectionPathPayloadRouteInput[Payload any] struct {
	pathID  uuid.UUID
	payload Payload
}

type collectionActorPathPayloadRouteInput[Payload any] struct {
	actor   AdminActor
	pathID  uuid.UUID
	payload Payload
}

type collectionPathPairRouteInput struct {
	collectionID uuid.UUID
	engramID     uuid.UUID
}

func removeMemoryAdminCollectionItemRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return collectionRoute(
		requireAdminActor,
		http.StatusOK,
		func(writer http.ResponseWriter, request *http.Request, _ AdminActor) (collectionPathPairRouteInput, bool) {
			collectionID, ok := parsePathUUID(writer, request, "collection_id")
			if !ok {
				return collectionPathPairRouteInput{}, false
			}
			engramID, ok := parsePathUUID(writer, request, "engram_id")
			if !ok {
				return collectionPathPairRouteInput{}, false
			}
			return collectionPathPairRouteInput{collectionID: collectionID, engramID: engramID}, true
		},
		func(ctx context.Context, input collectionPathPairRouteInput) (any, error) {
			return service.RemoveCollectionItem(ctx, input.collectionID, input.engramID)
		},
	)
}

func collectionRoute[Input any](
	requireAdminActor RequireAdminActor,
	successStatus int,
	parse func(writer http.ResponseWriter, request *http.Request, actor AdminActor) (Input, bool),
	execute func(ctx context.Context, input Input) (any, error),
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actor, ok := requireActor(writer, request, requireAdminActor)
		if !ok {
			return
		}
		input, ok := parse(writer, request, actor)
		if !ok {
			return
		}
		writeServiceCall(writer, successStatus, func() (any, error) {
			return execute(request.Context(), input)
		})
	}
}

func parseCollectionPayload[Payload any](writer http.ResponseWriter, request *http.Request) (Payload, bool) {
	var payload Payload
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return payload, false
	}
	return payload, true
}
