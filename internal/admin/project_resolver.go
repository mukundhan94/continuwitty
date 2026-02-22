package admin

import (
	"context"
	"strings"
)

// PassthroughProjectResolver accepts explicitly provided project ids for write operations.
type PassthroughProjectResolver struct{}

// ResolveProjectIDForWrite validates and returns a caller-provided project id.
func (PassthroughProjectResolver) ResolveProjectIDForWrite(_ context.Context, input ResolveProjectWriteInput) (ResolveProjectWriteResult, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		return ResolveProjectWriteResult{}, ErrProjectIDRequired
	}
	return ResolveProjectWriteResult{
		ProjectID:          projectID,
		UsedDefaultProject: false,
	}, nil
}
