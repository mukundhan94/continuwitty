package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"engram/internal/auth"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const sessionUITestPassword = "StrongPassword-12345"

var csrfTokenInputPattern = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)

func TestMountSessionUIRoutesHomeRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	handler, _, _ := buildSessionUITestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", response.Code)
	}
	if response.Header().Get("Location") != "/login" {
		t.Fatalf("expected redirect to /login, got %q", response.Header().Get("Location"))
	}
}

func TestMountSessionUIRoutesDashboardRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	handler, _, _ := buildSessionUITestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/ui", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", response.Code)
	}
	if response.Header().Get("Location") != "/login" {
		t.Fatalf("expected redirect to /login, got %q", response.Header().Get("Location"))
	}
}

func TestMountSessionUIRoutesLoginRejectsInvalidCredentials(t *testing.T) {
	handler, manager, _ := buildSessionUITestHandler(t)
	csrfToken, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	values := url.Values{}
	values.Set("username", "wrong")
	values.Set("password", "wrong")
	values.Set("csrf_token", csrfToken)

	request := newSessionUIFormRequest(t, http.MethodPost, "/login", values, loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "Invalid username or password.") {
		t.Fatalf("expected invalid credentials message in response body")
	}
}

func TestMountSessionUIRoutesLoginRejectsInvalidCSRF(t *testing.T) {
	handler, manager, record := buildSessionUITestHandler(t)
	_, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	values := url.Values{}
	values.Set("username", record.Username)
	values.Set("password", sessionUITestPassword)
	values.Set("csrf_token", "bad-token")

	request := newSessionUIFormRequest(t, http.MethodPost, "/login", values, loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", response.Code)
	}
}

func TestMountSessionUIRoutesLoginAndLogoutWorkflow(t *testing.T) {
	handler, manager, record := buildSessionUITestHandler(t)
	csrfToken, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	loginValues := url.Values{}
	loginValues.Set("username", record.Username)
	loginValues.Set("password", sessionUITestPassword)
	loginValues.Set("csrf_token", csrfToken)

	loginRequest := newSessionUIFormRequest(t, http.MethodPost, "/login", loginValues, loginCookie)
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)

	if loginResponse.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", loginResponse.Code)
	}
	if loginResponse.Header().Get("Location") != "/ui" {
		t.Fatalf("expected redirect to /ui, got %q", loginResponse.Header().Get("Location"))
	}
	authenticatedCookie := findResponseCookie(loginResponse, manager.CookieName())
	if authenticatedCookie == nil {
		t.Fatalf("expected session cookie after login")
	}

	dashboardRequest := httptest.NewRequest(http.MethodGet, "/ui", nil)
	dashboardRequest.AddCookie(authenticatedCookie)
	dashboardResponse := httptest.NewRecorder()
	handler.ServeHTTP(dashboardResponse, dashboardRequest)
	if dashboardResponse.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", dashboardResponse.Code)
	}
	if !strings.Contains(dashboardResponse.Body.String(), "Engram Vault Test UI") {
		t.Fatalf("expected dashboard title in response body")
	}
	if !strings.Contains(dashboardResponse.Body.String(), record.Username) {
		t.Fatalf("expected username %q in dashboard", record.Username)
	}
	logoutCSRFToken := extractCSRFTokenFromHTML(t, dashboardResponse.Body.String())
	dashboardCookie := findResponseCookie(dashboardResponse, manager.CookieName())
	if dashboardCookie != nil {
		authenticatedCookie = dashboardCookie
	}

	logoutValues := url.Values{}
	logoutValues.Set("csrf_token", logoutCSRFToken)
	logoutRequest := newSessionUIFormRequest(t, http.MethodPost, "/logout", logoutValues, authenticatedCookie)
	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", logoutResponse.Code)
	}
	if logoutResponse.Header().Get("Location") != "/login" {
		t.Fatalf("expected redirect to /login, got %q", logoutResponse.Header().Get("Location"))
	}
	clearedCookie := findResponseCookie(logoutResponse, manager.CookieName())
	if clearedCookie == nil {
		t.Fatalf("expected clear-session cookie after logout")
	}

	blockedDashboardRequest := httptest.NewRequest(http.MethodGet, "/ui", nil)
	blockedDashboardRequest.AddCookie(clearedCookie)
	blockedDashboardResponse := httptest.NewRecorder()
	handler.ServeHTTP(blockedDashboardResponse, blockedDashboardRequest)
	if blockedDashboardResponse.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", blockedDashboardResponse.Code)
	}
	if blockedDashboardResponse.Header().Get("Location") != "/login" {
		t.Fatalf("expected redirect to /login, got %q", blockedDashboardResponse.Header().Get("Location"))
	}
}

func TestMountSessionUIRoutesLoginRedirectPathSanitization(t *testing.T) {
	testCases := []struct {
		name             string
		nextPath         string
		expectedRedirect string
	}{
		{name: "trusted relative path", nextPath: "/ui/admin", expectedRedirect: "/ui/admin"},
		{name: "absolute url rejected", nextPath: "https://malicious.example/phish", expectedRedirect: "/ui"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler, manager, record := buildSessionUITestHandler(t)
			csrfToken, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

			values := url.Values{}
			values.Set("username", record.Username)
			values.Set("password", sessionUITestPassword)
			values.Set("csrf_token", csrfToken)
			values.Set("next_path", testCase.nextPath)

			request := newSessionUIFormRequest(t, http.MethodPost, "/login", values, loginCookie)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusSeeOther {
				t.Fatalf("expected status 303, got %d", response.Code)
			}
			if response.Header().Get("Location") != testCase.expectedRedirect {
				t.Fatalf("expected redirect to %q, got %q", testCase.expectedRedirect, response.Header().Get("Location"))
			}
		})
	}
}

func TestMountSessionUIRoutesLogoutRejectsInvalidCSRF(t *testing.T) {
	handler, manager, record := buildSessionUITestHandler(t)
	csrfToken, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	loginValues := url.Values{}
	loginValues.Set("username", record.Username)
	loginValues.Set("password", sessionUITestPassword)
	loginValues.Set("csrf_token", csrfToken)

	loginRequest := newSessionUIFormRequest(t, http.MethodPost, "/login", loginValues, loginCookie)
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	authenticatedCookie := findResponseCookie(loginResponse, manager.CookieName())
	if authenticatedCookie == nil {
		t.Fatalf("expected session cookie after login")
	}

	badLogoutValues := url.Values{}
	badLogoutValues.Set("csrf_token", "bad-token")
	badLogoutRequest := newSessionUIFormRequest(t, http.MethodPost, "/logout", badLogoutValues, authenticatedCookie)
	badLogoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(badLogoutResponse, badLogoutRequest)
	if badLogoutResponse.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", badLogoutResponse.Code)
	}

	stillAuthenticatedRequest := httptest.NewRequest(http.MethodGet, "/ui", nil)
	stillAuthenticatedRequest.AddCookie(authenticatedCookie)
	stillAuthenticatedResponse := httptest.NewRecorder()
	handler.ServeHTTP(stillAuthenticatedResponse, stillAuthenticatedRequest)
	if stillAuthenticatedResponse.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", stillAuthenticatedResponse.Code)
	}
}

func buildSessionUITestHandler(t *testing.T) (http.Handler, *auth.SessionManager, *models.UserAuthRecord) {
	t.Helper()
	manager, err := auth.NewSessionManager("dev-session-secret-for-tests", auth.DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}
	passwordHash, err := auth.HashPassword(sessionUITestPassword, []byte{0x01, 0x02, 0x03, 0x04})
	if err != nil {
		t.Fatalf("expected password hashing to succeed: %v", err)
	}
	record := &models.UserAuthRecord{
		UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000712"),
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
	dependencies := SessionAuthDependencies{
		SessionManager:       manager,
		LookupUserByUsername: lookupByUsername,
		LookupUserByID:       lookupByID,
		VerifyPassword:       auth.VerifyPassword,
		GenerateCSRFToken:    auth.GenerateCSRFToken,
	}
	MountSessionAuthRoutes(router, dependencies)
	MountSessionUIRoutes(router, dependencies)
	return SessionActorMiddleware(manager, lookupByID)(router), manager, record
}

func fetchLoginCSRFTokenAndCookie(
	t *testing.T,
	handler http.Handler,
	manager *auth.SessionManager,
	cookie *http.Cookie,
) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	csrfToken := extractCSRFTokenFromHTML(t, response.Body.String())
	nextCookie := findResponseCookie(response, manager.CookieName())
	if nextCookie == nil {
		nextCookie = cookie
	}
	if nextCookie == nil {
		t.Fatalf("expected session cookie from login page")
	}
	return csrfToken, nextCookie
}

func extractCSRFTokenFromHTML(t *testing.T, html string) string {
	t.Helper()
	matches := csrfTokenInputPattern.FindStringSubmatch(html)
	if len(matches) != 2 {
		t.Fatalf("expected csrf token input in html response")
	}
	return matches[1]
}

func newSessionUIFormRequest(
	t *testing.T,
	method string,
	path string,
	values url.Values,
	cookie *http.Cookie,
) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	return request
}
