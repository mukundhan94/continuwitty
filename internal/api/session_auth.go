package api

import (
	"context"
	"net/http"
	"strings"

	"engram/internal/auth"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
)

// SessionUserByUsernameLookup resolves an auth-capable user record by username.
type SessionUserByUsernameLookup func(ctx context.Context, username string) (*models.UserAuthRecord, error)

// SessionAuthDependencies captures required collaborators for session auth routes.
type SessionAuthDependencies struct {
	SessionManager       *auth.SessionManager
	LookupUserByUsername SessionUserByUsernameLookup
	LookupUserByID       SessionUserLookup
	VerifyPassword       func(password, encodedHash string) bool
	GenerateCSRFToken    func() (string, error)
}

type sessionAuthDependencies struct {
	manager              *auth.SessionManager
	lookupUserByUsername SessionUserByUsernameLookup
	lookupUserByID       SessionUserLookup
	verifyPassword       func(password, encodedHash string) bool
	generateCSRFToken    func() (string, error)
}

type sessionLoginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	CSRFToken string `json:"csrf_token"`
}

type sessionLogoutRequest struct {
	CSRFToken string `json:"csrf_token"`
}

// MountSessionAuthRoutes registers session login/logout/csrf/me endpoints.
func MountSessionAuthRoutes(router chi.Router, dependencies SessionAuthDependencies) {
	deps := sessionAuthDependencies{
		manager:              dependencies.SessionManager,
		lookupUserByUsername: dependencies.LookupUserByUsername,
		lookupUserByID:       dependencies.LookupUserByID,
		verifyPassword:       dependencies.VerifyPassword,
		generateCSRFToken:    dependencies.GenerateCSRFToken,
	}
	router.Route("/api/v1/session", func(session chi.Router) {
		session.Get("/csrf", func(writer http.ResponseWriter, request *http.Request) {
			if !deps.validateCoreDependencies(writer) {
				return
			}
			state, _ := deps.manager.DecodeRequest(request)
			if strings.TrimSpace(state.CSRFToken) == "" {
				token, err := deps.generateCSRFToken()
				if err != nil {
					writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to generate csrf token"})
					return
				}
				state.CSRFToken = token
			}
			if !deps.writeSessionCookie(writer, state) {
				return
			}
			writeJSON(writer, http.StatusOK, map[string]string{"csrf_token": state.CSRFToken})
		})

		session.Post("/login", func(writer http.ResponseWriter, request *http.Request) {
			if !deps.validateCoreDependencies(writer) || deps.lookupUserByUsername == nil || deps.verifyPassword == nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "session auth dependencies are not configured"})
				return
			}

			var payload sessionLoginRequest
			if !decodeJSONAllowEmpty(writer, request, &payload) {
				return
			}
			username := strings.TrimSpace(payload.Username)
			password := payload.Password
			if username == "" || password == "" {
				writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "username and password are required"})
				return
			}

			state, err := deps.manager.DecodeRequest(request)
			if err != nil || strings.TrimSpace(state.CSRFToken) == "" || payload.CSRFToken != state.CSRFToken {
				writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "invalid csrf token"})
				return
			}

			record, err := deps.lookupUserByUsername(request.Context(), username)
			if err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
				return
			}
			if record == nil || !record.IsActive || !deps.verifyPassword(password, record.PasswordHash) {
				writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "invalid credentials"})
				return
			}

			nextCSRFToken, err := deps.generateCSRFToken()
			if err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to generate csrf token"})
				return
			}
			nextState := auth.SessionState{
				User: &auth.SessionUser{
					UserID:   record.UserID.String(),
					Username: record.Username,
					Role:     string(record.Role),
				},
				CSRFToken: nextCSRFToken,
			}
			if !deps.writeSessionCookie(writer, nextState) {
				return
			}
			writeJSON(
				writer,
				http.StatusOK,
				map[string]any{
					"user_id":    record.UserID,
					"username":   record.Username,
					"role":       record.Role,
					"is_active":  record.IsActive,
					"csrf_token": nextCSRFToken,
				},
			)
		})

		session.Post("/logout", func(writer http.ResponseWriter, request *http.Request) {
			if !deps.validateCoreDependencies(writer) {
				return
			}

			var payload sessionLogoutRequest
			if !decodeJSONAllowEmpty(writer, request, &payload) {
				return
			}
			state, err := deps.manager.DecodeRequest(request)
			if err == nil && strings.TrimSpace(state.CSRFToken) != "" && payload.CSRFToken != state.CSRFToken {
				writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "invalid csrf token"})
				return
			}

			http.SetCookie(writer, &http.Cookie{
				Name:     deps.manager.CookieName(),
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				MaxAge:   -1,
				SameSite: http.SameSiteLaxMode,
			})
			writeJSON(writer, http.StatusOK, map[string]bool{"logged_out": true})
		})
	})

	router.Get("/api/v1/me", func(writer http.ResponseWriter, request *http.Request) {
		if deps.lookupUserByID == nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "session auth dependencies are not configured"})
			return
		}
		actor, ok := AdminActorFromContext(request.Context())
		if !ok {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "authentication required"})
			return
		}
		record, err := deps.lookupUserByID(request.Context(), actor.UserID)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
			return
		}
		if record == nil || !record.IsActive {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "authentication required"})
			return
		}
		writeJSON(
			writer,
			http.StatusOK,
			map[string]any{
				"user_id":   record.UserID,
				"username":  record.Username,
				"role":      record.Role,
				"is_active": record.IsActive,
			},
		)
	})
}

func (dependencies sessionAuthDependencies) validateCoreDependencies(writer http.ResponseWriter) bool {
	if dependencies.manager == nil || dependencies.generateCSRFToken == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "session auth dependencies are not configured"})
		return false
	}
	return true
}

func (dependencies sessionAuthDependencies) writeSessionCookie(writer http.ResponseWriter, state auth.SessionState) bool {
	token, err := dependencies.manager.Encode(state)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to encode session"})
		return false
	}
	http.SetCookie(writer, &http.Cookie{
		Name:     dependencies.manager.CookieName(),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return true
}
