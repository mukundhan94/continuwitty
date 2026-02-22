package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const (
	// HeaderAdminActorUserID carries the authenticated actor user id for migration-time admin routes.
	HeaderAdminActorUserID = "X-Engram-Actor-User-ID"
	// HeaderAdminActorRole carries the authenticated actor role for migration-time admin routes.
	HeaderAdminActorRole = "X-Engram-Actor-Role"
)

var errInvalidAdminActor = errors.New("invalid admin actor")

// RequireAdminActorFromHeaders resolves the admin actor from request headers.
func RequireAdminActorFromHeaders(request *http.Request) (AdminActor, error) {
	if request == nil {
		return AdminActor{}, errInvalidAdminActor
	}

	userIDValue := strings.TrimSpace(request.Header.Get(HeaderAdminActorUserID))
	roleValue := strings.ToLower(strings.TrimSpace(request.Header.Get(HeaderAdminActorRole)))
	if userIDValue == "" || roleValue != "admin" {
		return AdminActor{}, errInvalidAdminActor
	}

	userID, err := uuid.Parse(userIDValue)
	if err != nil {
		return AdminActor{}, errInvalidAdminActor
	}

	return AdminActor{
		UserID: userID,
		Role:   roleValue,
	}, nil
}
