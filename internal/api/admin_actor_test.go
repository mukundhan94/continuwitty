package api

import (
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRequireAdminActorFromHeadersReturnsActor(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin/memory/sessions", nil)
	request.Header.Set(HeaderAdminActorUserID, "00000000-0000-0000-0000-000000000411")
	request.Header.Set(HeaderAdminActorRole, "admin")

	actor, err := RequireAdminActorFromHeaders(request)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if actor.UserID != uuid.MustParse("00000000-0000-0000-0000-000000000411") {
		t.Fatalf("unexpected actor user id: %s", actor.UserID)
	}
	if actor.Role != "admin" {
		t.Fatalf("unexpected actor role: %s", actor.Role)
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
