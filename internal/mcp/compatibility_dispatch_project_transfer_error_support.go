package mcp

import (
	"errors"

	internalexport "engram/internal/export"
)

const defaultProjectImportConflictPolicy = "skip"

func runProjectTransferDispatch[Result any](
	call func() (*Result, error),
	toPayload func(Result) map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	result, err := call()
	if err != nil {
		return nil, true, mapProjectTransferError(err)
	}
	if result == nil {
		return nil, true, internalToolDispatchError()
	}
	return toPayload(*result), true, nil
}

func mapProjectTransferError(err error) *toolDispatchError {
	switch {
	case errors.Is(err, internalexport.ErrProjectNotFound):
		return invalidParamsWithStatus(404, "Project not found")
	case errors.Is(err, internalexport.ErrCollectionNotFoundForProject):
		return invalidParamsWithStatus(404, "Collection not found for project")
	case errors.Is(err, internalexport.ErrImportFileEmpty):
		return invalidParamsWithStatus(422, "Import file is empty")
	case errors.Is(err, internalexport.ErrImportZipMissingExportJSON):
		return invalidParamsWithStatus(422, "ZIP missing export.json")
	case errors.Is(err, internalexport.ErrUnsupportedImportFileFormat):
		return invalidParamsWithStatus(422, "Unsupported import file format")
	case errors.Is(err, internalexport.ErrInvalidExportBundle):
		return invalidParamsWithStatus(422, "Invalid export bundle")
	default:
		return internalToolDispatchError()
	}
}
