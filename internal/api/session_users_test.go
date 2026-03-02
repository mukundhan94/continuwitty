package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"engram/internal/auth"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type sessionUserRoutesHandlerOptions struct {
	actor      *models.UserAuthRecord
	listUsers  func(ctx context.Context, limit, offset int) ([]models.UserRecord, error)
	createUser func(ctx context.Context, input SessionUserCreateInput) (*models.UserRecord, error)
	updateUser func(ctx context.Context, userID uuid.UUID, input SessionUserUpdateInput) (*models.UserRecord, error)
}

type sessionUserLoginCredentials struct {
	username string
	password string
}

type sessionUserCallTracker struct {
	createCalls int
	updateCalls int
}

type adminManageUsersFixture struct {
	actor         *models.UserAuthRecord
	createdUserID uuid.UUID
	createdAt     time.Time
	callTracker   *sessionUserCallTracker
}

func TestMountSessionAuthRoutesAdminCanManageUsers(t *testing.T) {
	fixture := adminManageUsersFixture{
		actor:         newSessionRoutesTestActor(t, models.UserRoleAdmin),
		createdUserID: uuid.MustParse("00000000-0000-0000-0000-000000000811"),
		createdAt:     time.Date(2026, 2, 22, 4, 0, 0, 0, time.UTC),
		callTracker:   &sessionUserCallTracker{},
	}

	handler, manager := buildSessionUserRoutesTestHandler(
		t,
		newAdminManageUsersHandlerOptions(t, fixture),
	)
	loginCookie := loginSessionUserRoutesActor(
		t,
		handler,
		manager,
		sessionUserLoginCredentials{
			username: fixture.actor.Username,
			password: "StrongPassword-12345",
		},
	)

	assertAdminUsersListIncludesActor(t, handler, loginCookie, fixture.actor.Username)
	assertAdminCreateUserSucceeds(t, handler, loginCookie)
	assertAdminUpdateUserSucceeds(t, handler, loginCookie, fixture.createdUserID)

	if fixture.callTracker.createCalls != 1 {
		t.Fatalf("expected exactly one create call, got %d", fixture.callTracker.createCalls)
	}
	if fixture.callTracker.updateCalls != 1 {
		t.Fatalf("expected exactly one update call, got %d", fixture.callTracker.updateCalls)
	}
}

func newAdminManageUsersHandlerOptions(t *testing.T, fixture adminManageUsersFixture) sessionUserRoutesHandlerOptions {
	t.Helper()
	return sessionUserRoutesHandlerOptions{
		actor: fixture.actor,
		listUsers: func(_ context.Context, limit, offset int) ([]models.UserRecord, error) {
			if limit != 200 || offset != 0 {
				t.Fatalf("expected default pagination 200/0, got %d/%d", limit, offset)
			}
			return []models.UserRecord{
				{
					UserID:    fixture.actor.UserID,
					Username:  fixture.actor.Username,
					Role:      fixture.actor.Role,
					IsActive:  fixture.actor.IsActive,
					CreatedAt: fixture.actor.CreatedAt,
				},
			}, nil
		},
		createUser: func(_ context.Context, input SessionUserCreateInput) (*models.UserRecord, error) {
			fixture.callTracker.createCalls++
			assertAdminCreateInput(t, input)
			return &models.UserRecord{
				UserID:    fixture.createdUserID,
				Username:  input.Username,
				Role:      input.Role,
				IsActive:  input.IsActive,
				CreatedAt: fixture.createdAt,
			}, nil
		},
		updateUser: func(_ context.Context, userID uuid.UUID, input SessionUserUpdateInput) (*models.UserRecord, error) {
			fixture.callTracker.updateCalls++
			assertAdminUpdateInput(t, fixture.createdUserID, userID, input)
			return &models.UserRecord{
				UserID:    fixture.createdUserID,
				Username:  "analyst_01",
				Role:      *input.Role,
				IsActive:  true,
				CreatedAt: fixture.createdAt,
			}, nil
		},
	}
}

func assertAdminCreateInput(t *testing.T, input SessionUserCreateInput) {
	t.Helper()
	if input.Username != "analyst_01" {
		t.Fatalf("expected create username analyst_01, got %q", input.Username)
	}
	if input.Role != models.UserRoleAnalyst {
		t.Fatalf("expected create role analyst, got %s", input.Role)
	}
	if !input.IsActive {
		t.Fatalf("expected created user to be active")
	}
	if !auth.VerifyPassword("StrongPass123", input.PasswordHash) {
		t.Fatalf("expected password hash to verify input password")
	}
}

func assertAdminUpdateInput(
	t *testing.T,
	expectedUserID uuid.UUID,
	actualUserID uuid.UUID,
	input SessionUserUpdateInput,
) {
	t.Helper()
	if actualUserID != expectedUserID {
		t.Fatalf("expected update user id %s, got %s", expectedUserID, actualUserID)
	}
	if input.Role == nil || *input.Role != models.UserRoleViewer {
		t.Fatalf("expected update role viewer, got %#v", input.Role)
	}
	if input.PasswordHash != nil {
		t.Fatalf("expected password hash to be nil for role-only update")
	}
}

func assertAdminUsersListIncludesActor(
	t *testing.T,
	handler http.Handler,
	loginCookie *http.Cookie,
	username string,
) {
	t.Helper()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listRequest.AddCookie(loginCookie)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected users list status 200, got %d", listResponse.Code)
	}
	var listed []map[string]any
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode users list response: %v", err)
	}
	if len(listed) != 1 || listed[0]["username"] != username {
		t.Fatalf("expected list to include actor username %q, got %#v", username, listed)
	}
}

func assertAdminCreateUserSucceeds(t *testing.T, handler http.Handler, loginCookie *http.Cookie) {
	t.Helper()
	createBody := map[string]any{
		"username": "analyst_01",
		"password": "StrongPass123",
		"role":     "analyst",
	}
	createPayload, _ := json.Marshal(createBody)
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(createPayload))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.AddCookie(loginCookie)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createResponse.Code)
	}
	var created map[string]any
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created["username"] != "analyst_01" {
		t.Fatalf("expected created username analyst_01, got %#v", created["username"])
	}
	if created["role"] != "analyst" {
		t.Fatalf("expected created role analyst, got %#v", created["role"])
	}
}

func assertAdminUpdateUserSucceeds(
	t *testing.T,
	handler http.Handler,
	loginCookie *http.Cookie,
	userID uuid.UUID,
) {
	t.Helper()
	updateBody := map[string]any{"role": "viewer"}
	updatePayload, _ := json.Marshal(updateBody)
	updateRequest := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/users/"+userID.String(),
		bytes.NewReader(updatePayload),
	)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRequest.AddCookie(loginCookie)
	updateResponse := httptest.NewRecorder()
	handler.ServeHTTP(updateResponse, updateRequest)

	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d", updateResponse.Code)
	}
	var updated map[string]any
	if err := json.Unmarshal(updateResponse.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated["role"] != "viewer" {
		t.Fatalf("expected updated role viewer, got %#v", updated["role"])
	}
}

func TestMountSessionAuthRoutesNonAdminCannotAccessAdminRoutes(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleViewer)
	listCalled := false
	handler, manager := buildSessionUserRoutesTestHandler(
		t,
		sessionUserRoutesHandlerOptions{
			actor: actor,
			listUsers: func(_ context.Context, _ int, _ int) ([]models.UserRecord, error) {
				listCalled = true
				return []models.UserRecord{}, nil
			},
		},
	)
	loginCookie := loginSessionUserRoutesActor(
		t,
		handler,
		manager,
		sessionUserLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meRequest.AddCookie(loginCookie)
	meResponse := httptest.NewRecorder()
	handler.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("expected me status 200, got %d", meResponse.Code)
	}
	var mePayload map[string]any
	if err := json.Unmarshal(meResponse.Body.Bytes(), &mePayload); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if mePayload["role"] != "viewer" {
		t.Fatalf("expected me role viewer, got %#v", mePayload["role"])
	}

	usersRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	usersRequest.AddCookie(loginCookie)
	usersResponse := httptest.NewRecorder()
	handler.ServeHTTP(usersResponse, usersRequest)
	if usersResponse.Code != http.StatusForbidden {
		t.Fatalf("expected users status 403, got %d", usersResponse.Code)
	}
	if listCalled {
		t.Fatalf("expected users list dependency not to be called for non-admin actor")
	}

	adminUIRequest := httptest.NewRequest(http.MethodGet, "/ui/admin", nil)
	adminUIRequest.AddCookie(loginCookie)
	adminUIResponse := httptest.NewRecorder()
	handler.ServeHTTP(adminUIResponse, adminUIRequest)
	if adminUIResponse.Code != http.StatusForbidden {
		t.Fatalf("expected admin ui status 403, got %d", adminUIResponse.Code)
	}
}

func TestMountSessionAuthRoutesCreateUserReturnsConflictForDuplicateUsername(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAdmin)
	handler, manager := buildSessionUserRoutesTestHandler(
		t,
		sessionUserRoutesHandlerOptions{
			actor: actor,
			createUser: func(_ context.Context, _ SessionUserCreateInput) (*models.UserRecord, error) {
				return nil, repository.ErrUsernameExists
			},
		},
	)
	loginCookie := loginSessionUserRoutesActor(
		t,
		handler,
		manager,
		sessionUserLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	createBody := map[string]any{
		"username": "duplicate_user",
		"password": "StrongPass123",
	}
	payload, _ := json.Marshal(createBody)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected create status 409, got %d", response.Code)
	}
}

func TestMountSessionAuthRoutesUpdateUserRequiresFields(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAdmin)
	updateCalled := false
	handler, manager := buildSessionUserRoutesTestHandler(
		t,
		sessionUserRoutesHandlerOptions{
			actor: actor,
			updateUser: func(_ context.Context, _ uuid.UUID, _ SessionUserUpdateInput) (*models.UserRecord, error) {
				updateCalled = true
				return nil, nil
			},
		},
	)
	loginCookie := loginSessionUserRoutesActor(
		t,
		handler,
		manager,
		sessionUserLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/users/00000000-0000-0000-0000-000000000999",
		bytes.NewReader([]byte(`{}`)),
	)
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected update status 400, got %d", response.Code)
	}
	if updateCalled {
		t.Fatalf("expected update dependency not to be called")
	}
}

func buildSessionUserRoutesTestHandler(
	t *testing.T,
	options sessionUserRoutesHandlerOptions,
) (http.Handler, *auth.SessionManager) {
	t.Helper()

	actor := options.actor
	if actor == nil {
		actor = newSessionRoutesTestActor(t, models.UserRoleAdmin)
	}

	manager, err := auth.NewSessionManager("dev-session-secret-for-tests", auth.DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}

	lookupByUsername := func(_ context.Context, username string) (*models.UserAuthRecord, error) {
		if stringsEqualFoldTrimmed(username, actor.Username) {
			return actor, nil
		}
		return nil, nil
	}
	lookupByID := func(_ context.Context, userID uuid.UUID) (*models.UserAuthRecord, error) {
		if userID == actor.UserID {
			return actor, nil
		}
		return nil, nil
	}
	listUsers := options.listUsers
	if listUsers == nil {
		listUsers = func(_ context.Context, _ int, _ int) ([]models.UserRecord, error) {
			return []models.UserRecord{}, nil
		}
	}
	createUser := options.createUser
	if createUser == nil {
		createUser = func(_ context.Context, _ SessionUserCreateInput) (*models.UserRecord, error) {
			return nil, errors.New("create user dependency not configured")
		}
	}
	updateUser := options.updateUser
	if updateUser == nil {
		updateUser = func(_ context.Context, _ uuid.UUID, _ SessionUserUpdateInput) (*models.UserRecord, error) {
			return nil, errors.New("update user dependency not configured")
		}
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
			HashPassword: func(password string) (string, error) {
				return auth.HashPassword(password, nil)
			},
			ListUsers:  listUsers,
			CreateUser: createUser,
			UpdateUser: updateUser,
		},
	)
	MountSessionUIRoutes(
		router,
		SessionAuthDependencies{
			SessionManager:       manager,
			LookupUserByUsername: lookupByUsername,
			LookupUserByID:       lookupByID,
			VerifyPassword:       auth.VerifyPassword,
			GenerateCSRFToken:    auth.GenerateCSRFToken,
			CookieSecure:         false,
		},
	)
	return SessionActorMiddleware(manager, lookupByID)(router), manager
}

func loginSessionUserRoutesActor(
	t *testing.T,
	handler http.Handler,
	manager *auth.SessionManager,
	credentials sessionUserLoginCredentials,
) *http.Cookie {
	t.Helper()

	csrfToken, sessionCookie := fetchCSRFTokenAndCookie(t, handler, manager)
	loginBody := map[string]string{
		"username":   credentials.username,
		"password":   credentials.password,
		"csrf_token": csrfToken,
	}
	payload, _ := json.Marshal(loginBody)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(sessionCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", response.Code)
	}
	loginCookie := findResponseCookie(response, manager.CookieName())
	if loginCookie == nil {
		t.Fatalf("expected login cookie")
	}
	return loginCookie
}

func newSessionRoutesTestActor(t *testing.T, role models.UserRole) *models.UserAuthRecord {
	t.Helper()
	passwordHash, err := auth.HashPassword("StrongPassword-12345", []byte{0x01, 0x02, 0x03, 0x05})
	if err != nil {
		t.Fatalf("expected password hashing to succeed: %v", err)
	}
	return &models.UserAuthRecord{
		UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000710"),
		Username:     "session_actor",
		PasswordHash: passwordHash,
		Role:         role,
		IsActive:     true,
		CreatedAt:    time.Date(2026, 2, 22, 3, 0, 0, 0, time.UTC),
	}
}

func stringsEqualFoldTrimmed(left, right string) bool {
	return strings.TrimSpace(left) == strings.TrimSpace(right)
}
