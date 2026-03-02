package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestAdminActorResolversReturnActor(t *testing.T) {
	testCases := []struct {
		name           string
		requestFactory func() *http.Request
		resolver       func(request *http.Request) (AdminActor, error)
		expectedUserID uuid.UUID
	}{
		{
			name: "headers",
			requestFactory: func() *http.Request {
				request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)
				request.Header.Set(HeaderAdminActorUserID, "00000000-0000-0000-0000-000000000411")
				request.Header.Set(HeaderAdminActorRole, "admin")
				return request
			},
			resolver:       RequireAdminActorFromHeaders,
			expectedUserID: uuid.MustParse("00000000-0000-0000-0000-000000000411"),
		},
		{
			name: "context",
			requestFactory: func() *http.Request {
				request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)
				return WithAdminActor(
					request,
					AdminActor{
						UserID: uuid.MustParse("00000000-0000-0000-0000-000000000421"),
						Role:   "admin",
					},
				)
			},
			resolver:       RequireAdminActorFromContext,
			expectedUserID: uuid.MustParse("00000000-0000-0000-0000-000000000421"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actor, err := testCase.resolver(testCase.requestFactory())
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if actor.UserID != testCase.expectedUserID {
				t.Fatalf("unexpected actor user id: %s", actor.UserID)
			}
			if actor.Role != "admin" {
				t.Fatalf("unexpected actor role: %s", actor.Role)
			}
		})
	}
}

func TestRequireAdminActorFromHeadersRejectsMissingHeaders(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)

	_, err := RequireAdminActorFromHeaders(request)
	if err == nil {
		t.Fatalf("expected error for missing actor headers")
	}
}

func TestRequireAdminActorFromHeadersRejectsInvalidUserID(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)
	request.Header.Set(HeaderAdminActorUserID, "not-a-uuid")
	request.Header.Set(HeaderAdminActorRole, "admin")

	_, err := RequireAdminActorFromHeaders(request)
	if err == nil {
		t.Fatalf("expected error for invalid actor user id")
	}
}

func TestRequireAdminActorFromHeadersRejectsNonAdminRole(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)
	request.Header.Set(HeaderAdminActorUserID, "00000000-0000-0000-0000-000000000412")
	request.Header.Set(HeaderAdminActorRole, "viewer")

	_, err := RequireAdminActorFromHeaders(request)
	if err == nil {
		t.Fatalf("expected error for non-admin role")
	}
}

func TestRequireAdminActorFromContextRejectsMissingActor(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)

	_, err := RequireAdminActorFromContext(request)
	if err == nil {
		t.Fatalf("expected error for missing context actor")
	}
}

func TestRequireAdminActorFromContextRejectsNonAdminRole(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)
	request = WithAdminActor(request, AdminActor{
		UserID: uuid.MustParse("00000000-0000-0000-0000-000000000422"),
		Role:   "viewer",
	})

	_, err := RequireAdminActorFromContext(request)
	if err == nil {
		t.Fatalf("expected error for non-admin context actor")
	}
}

func TestAdminActorHeaderBridgeResolvesActor(t *testing.T) {
	testCases := []struct {
		name            string
		requestFactory  func() *http.Request
		expectedActorID uuid.UUID
	}{
		{
			name: "injects from headers",
			requestFactory: func() *http.Request {
				request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)
				request.Header.Set(HeaderAdminActorUserID, "00000000-0000-0000-0000-000000000423")
				request.Header.Set(HeaderAdminActorRole, "admin")
				return request
			},
			expectedActorID: uuid.MustParse("00000000-0000-0000-0000-000000000423"),
		},
		{
			name: "preserves existing context actor",
			requestFactory: func() *http.Request {
				request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)
				request = WithAdminActor(
					request,
					AdminActor{
						UserID: uuid.MustParse("00000000-0000-0000-0000-000000000424"),
						Role:   "admin",
					},
				)
				request.Header.Set(HeaderAdminActorUserID, "00000000-0000-0000-0000-000000000425")
				request.Header.Set(HeaderAdminActorRole, "admin")
				return request
			},
			expectedActorID: uuid.MustParse("00000000-0000-0000-0000-000000000424"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := AdminActorHeaderBridge(
				http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					actor, err := RequireAdminActorFromContext(request)
					if err != nil {
						writer.WriteHeader(http.StatusForbidden)
						return
					}
					if actor.UserID != testCase.expectedActorID {
						t.Fatalf("unexpected actor user id: %s", actor.UserID)
					}
					writer.WriteHeader(http.StatusNoContent)
				}),
			)

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, testCase.requestFactory())
			if response.Code != http.StatusNoContent {
				t.Fatalf("expected status 204, got %d", response.Code)
			}
		})
	}
}

func TestAdminActorHeaderBridgeLeavesRequestUnauthenticatedWhenHeadersMissing(t *testing.T) {
	handler := AdminActorHeaderBridge(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, err := RequireAdminActorFromContext(request)
			if err != nil {
				writer.WriteHeader(http.StatusForbidden)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		}),
	)

	request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", response.Code)
	}
}
