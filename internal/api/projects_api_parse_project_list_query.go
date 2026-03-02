package api

import "net/http"

func parseProjectListQuery(
	request *http.Request,
) (includeArchived bool, limit int, offset int, err error) {
	includeArchived, err = parseProjectIncludeArchivedQuery(request.URL.Query().Get("include_archived"))
	if err != nil {
		return false, 0, 0, err
	}
	limit, err = parseBoundedIntQueryParam(
		request.URL.Query().Get("limit"),
		defaultProjectListLimit,
		minProjectListLimit,
		maxProjectListLimit,
	)
	if err != nil {
		return false, 0, 0, errInvalidProjectQueryParam
	}
	offset, err = parseBoundedIntQueryParam(
		request.URL.Query().Get("offset"),
		defaultProjectListOffset,
		defaultProjectListOffset,
		int(^uint(0)>>1),
	)
	if err != nil {
		return false, 0, 0, errInvalidProjectQueryParam
	}
	return includeArchived, limit, offset, nil
}
