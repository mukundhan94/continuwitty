package main

import (
	"context"

	"engram/internal/mcp"
)

func (adapter mcpEngramAdminAdapter) RestoreEngram(
	ctx context.Context,
	request mcp.EngramRestoreRequest,
) (*mcp.EngramRestoreResponse, error) {
	return runEngramMutation(
		ctx,
		adapter,
		engramMutationRequest{
			ActorUserID: request.ActorUserID,
			ActorRole:   request.ActorRole,
			EngramID:    request.EngramID,
			Kind:        engramMutationRestore,
		},
		func(mutation engramMutationOutcome) *mcp.EngramRestoreResponse {
			return &mcp.EngramRestoreResponse{
				EngramID: mutation.EngramID,
				Restored: mutation.Applied,
			}
		},
	)
}
