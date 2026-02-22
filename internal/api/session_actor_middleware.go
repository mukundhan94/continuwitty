package api

import (
	"context"
	"net/http"
	"strings"

	"engram/internal/auth"
	"engram/internal/models"

	"github.com/google/uuid"
)

// SessionUserLookup loads a canonical user auth record for a session-bound user id.
type SessionUserLookup func(ctx context.Context, userID uuid.UUID) (*models.UserAuthRecord, error)

// SessionActorMiddleware populates request actor context from a signed session cookie.
func SessionActorMiddleware(manager *auth.SessionManager, lookup SessionUserLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "handler missing"})
			})
		}
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if manager != nil && lookup != nil {
				actor, ok := resolveSessionActor(request.Context(), request, manager, lookup)
				if ok {
					request = WithAdminActor(request, actor)
				}
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func resolveSessionActor(
	ctx context.Context,
	request *http.Request,
	manager *auth.SessionManager,
	lookup SessionUserLookup,
) (AdminActor, bool) {
	if request == nil {
		return AdminActor{}, false
	}
	state, err := manager.DecodeRequest(request)
	if err != nil || state.User == nil {
		return AdminActor{}, false
	}

	userID, err := uuid.Parse(strings.TrimSpace(state.User.UserID))
	if err != nil {
		return AdminActor{}, false
	}
	record, err := lookup(ctx, userID)
	if err != nil || record == nil || !record.IsActive {
		return AdminActor{}, false
	}
	role := strings.ToLower(strings.TrimSpace(string(record.Role)))
	if role == "" {
		return AdminActor{}, false
	}
	return AdminActor{
		UserID: record.UserID,
		Role:   role,
	}, true
}
