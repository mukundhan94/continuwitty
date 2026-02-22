package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/auth"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestMountSessionAuthRoutesCSRFIssuesTokenAndCookie(t *testing.T) {
	handler, manager, _ := buildSessionAuthTestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/session/csrf", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["csrf_token"] == "" {
		t.Fatalf("expected csrf token in response")
	}
	cookie := findResponseCookie(response, manager.CookieName())
	if cookie == nil || cookie.Value == "" {
		t.Fatalf("expected session cookie to be set")
	}
}

func TestMountSessionAuthRoutesLoginSetsSessionCookie(t *testing.T) {
	handler, manager, record := buildSessionAuthTestHandler(t)

	csrfToken, sessionCookie := fetchCSRFTokenAndCookie(t, handler, manager)
	loginBody := map[string]string{
		"username":   record.Username,
		"password":   "StrongPassword-12345",
		"csrf_token": csrfToken,
	}
	bodyBytes, _ := json.Marshal(loginBody)
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", bytes.NewReader(bodyBytes))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRequest.AddCookie(sessionCookie)

	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", loginResponse.Code)
	}
	loginCookie := findResponseCookie(loginResponse, manager.CookieName())
	if loginCookie == nil || loginCookie.Value == "" {
		t.Fatalf("expected login response to set session cookie")
	}
	state, err := manager.Decode(loginCookie.Value)
	if err != nil {
		t.Fatalf("expected login cookie to decode: %v", err)
	}
	if state.User == nil || state.User.UserID != record.UserID.String() {
		t.Fatalf("unexpected session user after login: %#v", state.User)
	}
	if state.CSRFToken == "" {
		t.Fatalf("expected rotated csrf token after login")
	}
}

func TestMountSessionAuthRoutesLoginRejectsInvalidCSRF(t *testing.T) {
	handler, manager, record := buildSessionAuthTestHandler(t)

	_, sessionCookie := fetchCSRFTokenAndCookie(t, handler, manager)
	loginBody := map[string]string{
		"username":   record.Username,
		"password":   "StrongPassword-12345",
		"csrf_token": "invalid-token",
	}
	bodyBytes, _ := json.Marshal(loginBody)
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", bytes.NewReader(bodyBytes))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRequest.AddCookie(sessionCookie)

	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", loginResponse.Code)
	}
}

func TestMountSessionAuthRoutesMeUsesSessionActorMiddleware(t *testing.T) {
	handler, manager, record := buildSessionAuthTestHandler(t)

	csrfToken, sessionCookie := fetchCSRFTokenAndCookie(t, handler, manager)
	loginBody := map[string]string{
		"username":   record.Username,
		"password":   "StrongPassword-12345",
		"csrf_token": csrfToken,
	}
	bodyBytes, _ := json.Marshal(loginBody)
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", bytes.NewReader(bodyBytes))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRequest.AddCookie(sessionCookie)
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	loginCookie := findResponseCookie(loginResponse, manager.CookieName())
	if loginCookie == nil {
		t.Fatalf("expected session cookie after login")
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meRequest.AddCookie(loginCookie)
	meResponse := httptest.NewRecorder()
	handler.ServeHTTP(meResponse, meRequest)

	if meResponse.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", meResponse.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(meResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if payload["username"] != record.Username {
		t.Fatalf("expected username %q, got %#v", record.Username, payload["username"])
	}
}

func buildSessionAuthTestHandler(t *testing.T) (http.Handler, *auth.SessionManager, *models.UserAuthRecord) {
	t.Helper()

	manager, err := auth.NewSessionManager("dev-session-secret-for-tests", auth.DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}
	passwordHash, err := auth.HashPassword("StrongPassword-12345", []byte{0x01, 0x02, 0x03, 0x04})
	if err != nil {
		t.Fatalf("expected password hashing to succeed: %v", err)
	}
	record := &models.UserAuthRecord{
		UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000711"),
		Username:     "admin",
		PasswordHash: passwordHash,
		Role:         models.UserRoleAdmin,
		IsActive:     true,
	}

	lookupByUsername := func(_ context.Context, username string) (*models.UserAuthRecord, error) {
		if username != record.Username {
			return nil, nil
		}
		return record, nil
	}
	lookupByID := func(_ context.Context, userID uuid.UUID) (*models.UserAuthRecord, error) {
		if userID != record.UserID {
			return nil, nil
		}
		return record, nil
	}

	router := chi.NewRouter()
	MountSessionAuthRoutes(
		router,
		SessionAuthDependencies{
			SessionManager:       manager,
			LookupUserByUsername: lookupByUsername,
			LookupUserByID:       lookupByID,
			VerifyPassword:       auth.VerifyPassword,
			GenerateCSRFToken:    auth.GenerateCSRFToken,
		},
	)
	return SessionActorMiddleware(manager, lookupByID)(router), manager, record
}

func fetchCSRFTokenAndCookie(t *testing.T, handler http.Handler, manager *auth.SessionManager) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/session/csrf", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode csrf response: %v", err)
	}
	cookie := findResponseCookie(response, manager.CookieName())
	if cookie == nil {
		t.Fatalf("expected session cookie from csrf route")
	}
	return payload["csrf_token"], cookie
}

func findResponseCookie(response *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
