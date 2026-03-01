package mcp

import (
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceTokenProjectScopeRejectsEngramLinkUpdateOutsideAllowlist(t *testing.T) {
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000007200")
	updateService := &fakeEngramLinkUpdateService{
		updated: &models.EngramLinkRecord{LinkID: linkID},
	}
	service := NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			EngramLinkGet: &fakeEngramLinkGetService{
				link: &models.EngramLinkRecord{
					LinkID:    linkID,
					ProjectID: "project-disallowed",
				},
			},
			EngramLinkUpdate: updateService,
		},
	)
	request := tokenScopedDirectRequest(tokenScopedRequest{
		actorID:  "00000000-0000-0000-0000-000000007201",
		toolName: "engram.link_update",
		params: map[string]any{
			"link_id":    linkID.String(),
			"confidence": 0.2,
		},
		scope:             models.MCPTokenScopeWrite,
		allowedProjectIDs: []string{"project-allowed"},
	})

	assertTokenScopeDeniedProject(t, service, request, "project-disallowed")
	if updateService.call.LinkID != uuid.Nil {
		t.Fatalf("expected link update service not to be called when project is disallowed")
	}
}
