package mcp

import (
	"errors"
	"fmt"
	"strings"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

type collectionActorRequest struct {
	ActorUserID  uuid.UUID
	ActorRole    models.UserRole
	CollectionID uuid.UUID
}

func parseCollectionActorRequest(
	actor Actor,
	params map[string]any,
) (collectionActorRequest, *toolDispatchError) {
	collectionID, ok := requiredUUIDParam(params, "collection_id")
	if !ok {
		return collectionActorRequest{}, invalidParamError("collection_id")
	}
	return collectionActorRequest{
		ActorUserID:  actor.UserID,
		ActorRole:    normalizedActorRole(actor),
		CollectionID: collectionID,
	}, nil
}

func runCollectionMutationDispatch[Result any](
	collectionID uuid.UUID,
	call func() (*Result, error),
	toPayload func(Result) map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	result, err := call()
	if err != nil {
		return nil, true, mapEngramCollectionMutationError(err, collectionID)
	}
	if result == nil {
		return nil, true, collectionNotFoundDispatchError(collectionID)
	}
	return toPayload(*result), true, nil
}

func mapEngramCollectionMutationError(err error, collectionID uuid.UUID) *toolDispatchError {
	switch {
	case errors.Is(err, admin.ErrCollectionNotFound):
		return collectionNotFoundDispatchError(collectionID)
	case errors.Is(err, admin.ErrCollectionStale):
		return invalidParamsWithStatus(409, admin.ErrCollectionStale.Error())
	case errors.Is(err, repository.ErrCollectionNameExists):
		return invalidParamsWithStatus(409, repository.ErrCollectionNameExists.Error())
	default:
		return internalToolDispatchError()
	}
}

func collectionNotFoundDispatchError(collectionID uuid.UUID) *toolDispatchError {
	return &toolDispatchError{
		code:    -32004,
		message: "Collection not found",
		data: map[string]any{
			"collection_id": collectionID.String(),
		},
	}
}

func optionalUUIDSliceParam(params map[string]any, key string) ([]uuid.UUID, bool) {
	value, found := optionalParamValue(params, key)
	if !found || value == nil {
		return []uuid.UUID{}, true
	}
	return parseUUIDSliceValue(value)
}

func parseUUIDSliceValue(value any) ([]uuid.UUID, bool) {
	switch typed := value.(type) {
	case []uuid.UUID:
		return append([]uuid.UUID(nil), typed...), true
	case []string:
		return parseUUIDSliceFromStrings(typed)
	case []any:
		stringValues := make([]string, 0, len(typed))
		for _, item := range typed {
			stringValues = append(stringValues, fmt.Sprint(item))
		}
		return parseUUIDSliceFromStrings(stringValues)
	default:
		return nil, false
	}
}

func parseUUIDSliceFromStrings(values []string) ([]uuid.UUID, bool) {
	parsed := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		item, ok := parseUUIDSliceItem(value)
		if !ok {
			return nil, false
		}
		parsed = append(parsed, item)
	}
	return parsed, true
}

func parseUUIDSliceItem(value string) (uuid.UUID, bool) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, false
	}
	return parsed, true
}
