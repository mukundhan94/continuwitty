package api

import (
	"context"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/google/uuid"
)

func (f *fakeMemoryAdminService) ActionEngramConsolidationSuggestion(
	ctx context.Context,
	suggestionID uuid.UUID,
	actorUserID uuid.UUID,
	request admin.EngramConsolidationSuggestionActionRequest,
) (*models.EngramConsolidationSuggestion, error) {
	return requireFakeAdminHandler("ActionEngramConsolidationSuggestion", f.actionConsolidationFn)(
		ctx,
		suggestionID,
		actorUserID,
		request,
	)
}

func (f *fakeMemoryAdminService) ListMemoryCurationSuggestions(
	ctx context.Context,
	request admin.MemoryCurationSuggestionListRequest,
) ([]models.MemoryCurationSuggestion, error) {
	return requireFakeAdminHandler("ListMemoryCurationSuggestions", f.listCurationFn)(ctx, request)
}

func (f *fakeMemoryAdminService) ActionMemoryCurationSuggestion(
	ctx context.Context,
	suggestionID uuid.UUID,
	actorUserID uuid.UUID,
	request admin.MemoryCurationSuggestionActionRequest,
) (*models.MemoryCurationSuggestion, error) {
	return requireFakeAdminHandler("ActionMemoryCurationSuggestion", f.actionCurationFn)(
		ctx,
		suggestionID,
		actorUserID,
		request,
	)
}
