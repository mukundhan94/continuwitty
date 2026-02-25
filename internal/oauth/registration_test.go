package oauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"engram/internal/config"
	"engram/internal/models"
	"engram/internal/repository"
)

func TestDedupStringListTrimsAndDeduplicates(t *testing.T) {
	actual := DedupStringList([]string{"  one  ", "", "one", " two ", "two"})
	expected := []string{"one", "two"}
	requireEqualStringSlice(t, expected, actual)
}

func TestHandleRegisterNormalizesListsAndIssuesSecret(t *testing.T) {
	captured := repository.OAuthClientCreateInput{}
	service := NewRegistrationService(nil)
	service.deps.buildClientID = func() string { return "engram_client_static" }
	service.deps.issueClientSecret = func() (string, error) { return "secret-value", nil }
	service.deps.oauthSecretHash = func(_ SecretHashInput) string { return "hashed-secret" }
	service.deps.createOAuthClient = func(_ context.Context, _ repository.Queryer, input repository.OAuthClientCreateInput) (*models.OAuthClientRecord, error) {
		captured = input
		return &models.OAuthClientRecord{
			ClientID:                input.ClientID,
			ClientName:              input.ClientName,
			RedirectURIs:            input.RedirectURIs,
			GrantTypes:              input.GrantTypes,
			ResponseTypes:           input.ResponseTypes,
			TokenEndpointAuthMethod: input.TokenEndpointAuthMethod,
			ClientSecretHash:        input.ClientSecretHash,
			MetadataJSON:            input.MetadataJSON,
			CreatedAt:               time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC),
		}, nil
	}

	response, err := service.HandleRegister(
		context.Background(),
		registrationSettings(),
		RegistrationRequest{
			ClientName:              "  Test Client  ",
			RedirectURIs:            []string{" https://example.com/callback ", "", "https://example.com/callback"},
			GrantTypes:              []string{},
			ResponseTypes:           []string{},
			TokenEndpointAuthMethod: "client_secret_post",
		},
		&SessionUser{UserID: "user-1", Username: "admin", Role: "admin"},
	)
	requireNoErrorRegistration(t, err)
	requireEqualStringRegistration(t, "engram_client_static", response.ClientID)
	requireEqualStringRegistration(t, "secret-value", valueOrEmpty(response.ClientSecret))
	requireEqualStringRegistration(t, "client_secret_post", response.TokenEndpointAuthMethod)
	requireEqualStringRegistration(t, "Test Client", captured.ClientName)
	requireEqualStringSlice(t, []string{"https://example.com/callback"}, captured.RedirectURIs)
	requireEqualStringSlice(t, []string{"authorization_code"}, captured.GrantTypes)
	requireEqualStringSlice(t, []string{"code"}, captured.ResponseTypes)
	requireEqualStringRegistration(t, "hashed-secret", valueOrEmpty(captured.ClientSecretHash))
}

func TestHandleRegisterRejectsUnauthorizedSessionUser(t *testing.T) {
	testCases := []struct {
		name           string
		sessionUser    *SessionUser
		expectedStatus int
		expectedError  string
	}{
		{name: "missing session", sessionUser: nil, expectedStatus: 401, expectedError: "invalid_client"},
		{name: "insufficient role", sessionUser: &SessionUser{UserID: "user-1", Username: "analyst", Role: "analyst"}, expectedStatus: 403, expectedError: "insufficient_privilege"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			service := NewRegistrationService(nil)
			_, err := service.HandleRegister(
				context.Background(),
				registrationSettings(),
				RegistrationRequest{RedirectURIs: []string{"https://example.com/callback"}},
				testCase.sessionUser,
			)
			registrationErr := RegistrationError{}
			if !errors.As(err, &registrationErr) {
				t.Fatalf("expected registration error, got %v", err)
			}
			requireEqualInt(t, testCase.expectedStatus, registrationErr.StatusCode)
			requireEqualStringRegistration(t, testCase.expectedError, registrationErr.ErrorCode)
		})
	}
}

func registrationSettings() config.Settings {
	settings := config.Settings{}
	settings.OAuthEnabled = true
	settings.OAuthRequireProtectedRegistration = true
	settings.OAuthClientSecretPepper = "oauth-pepper"
	return settings
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func requireNoErrorRegistration(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func requireEqualStringRegistration[T ~string](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

func requireEqualInt(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}

func requireEqualStringSlice(t *testing.T, expected []string, actual []string) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Fatalf("expected length %d, got %d", len(expected), len(actual))
	}
	for index := range expected {
		if expected[index] != actual[index] {
			t.Fatalf("expected[%d]=%q, got %q", index, expected[index], actual[index])
		}
	}
}
