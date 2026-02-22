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
	userID, ok := resolveSessionUserID(request, manager)
	if !ok {
		return AdminActor{}, false
	}
	record, ok := lookupActiveSessionUserRecord(ctx, lookup, userID)
	if !ok {
		return AdminActor{}, false
	}
	role, ok := normalizeActorRole(record.Role)
	if !ok {
		return AdminActor{}, false
	}
	return AdminActor{
		UserID: record.UserID,
		Role:   role,
	}, true
}

func resolveSessionUserID(request *http.Request, manager *auth.SessionManager) (uuid.UUID, bool) {
	state, err := manager.DecodeRequest(request)
	if err != nil || state.User == nil {
		return uuid.Nil, false
	}
	return parseSessionActorUserID(state.User.UserID)
}

func parseSessionActorUserID(value string) (uuid.UUID, bool) {
	parsedUserID, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, false
	}
	return parsedUserID, true
}

func lookupActiveSessionUserRecord(
	ctx context.Context,
	lookup SessionUserLookup,
	userID uuid.UUID,
) (*models.UserAuthRecord, bool) {
	record, err := lookup(ctx, userID)
	if err != nil || record == nil || !record.IsActive {
		return nil, false
	}
	return record, true
}

func normalizeActorRole(role models.UserRole) (string, bool) {
	normalizedRole := strings.ToLower(strings.TrimSpace(string(role)))
	if normalizedRole == "" {
		return "", false
	}
	return normalizedRole, true
}
