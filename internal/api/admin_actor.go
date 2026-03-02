package api

import (
	"context"
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

type adminActorContextKey struct{}

// WithAdminActor stores an authenticated actor in the request context.
func WithAdminActor(request *http.Request, actor AdminActor) *http.Request {
	if request == nil {
		return nil
	}
	ctx := context.WithValue(request.Context(), adminActorContextKey{}, actor)
	return request.WithContext(ctx)
}

// AdminActorFromContext extracts the authenticated actor from request context.
func AdminActorFromContext(ctx context.Context) (AdminActor, bool) {
	if ctx == nil {
		return AdminActor{}, false
	}
	actor, ok := ctx.Value(adminActorContextKey{}).(AdminActor)
	if !ok || actor.UserID == uuid.Nil {
		return AdminActor{}, false
	}
	return actor, true
}

// RequireAdminActorFromContext resolves the admin actor from request context.
func RequireAdminActorFromContext(request *http.Request) (AdminActor, error) {
	if request == nil {
		return AdminActor{}, errInvalidAdminActor
	}
	actor, ok := AdminActorFromContext(request.Context())
	if !ok || strings.ToLower(strings.TrimSpace(actor.Role)) != "admin" {
		return AdminActor{}, errInvalidAdminActor
	}
	return actor, nil
}

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

// AdminActorHeaderBridge injects a context actor when migration-time headers are present.
func AdminActorHeaderBridge(next http.Handler) http.Handler {
	if next == nil {
		return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "handler missing"})
		})
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if _, exists := AdminActorFromContext(request.Context()); exists {
			next.ServeHTTP(writer, request)
			return
		}
		actor, err := RequireAdminActorFromHeaders(request)
		if err == nil {
			request = WithAdminActor(request, actor)
		}
		next.ServeHTTP(writer, request)
	})
}
