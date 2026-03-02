package admin

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestPassthroughProjectResolverReturnsTrimmedProjectID(t *testing.T) {
	resolver := PassthroughProjectResolver{}
	result, err := resolver.ResolveProjectIDForWrite(
		context.Background(),
		ResolveProjectWriteInput{
			ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000311"),
			ActorRole:   "admin",
			ProjectID:   "  engram-vault  ",
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ProjectID != "engram-vault" {
		t.Fatalf("expected trimmed project id, got %q", result.ProjectID)
	}
	if result.UsedDefaultProject {
		t.Fatalf("expected explicit project id to avoid default flag")
	}
}

func TestPassthroughProjectResolverRejectsEmptyProjectID(t *testing.T) {
	resolver := PassthroughProjectResolver{}
	_, err := resolver.ResolveProjectIDForWrite(
		context.Background(),
		ResolveProjectWriteInput{
			ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000312"),
			ActorRole:   "admin",
			ProjectID:   "   ",
		},
	)
	if err == nil {
		t.Fatalf("expected error for empty project id")
	}
	if err != ErrProjectIDRequired {
		t.Fatalf("expected ErrProjectIDRequired, got %v", err)
	}
}
