package api

import "net/http"

func parseProjectMemberListQuery(request *http.Request) (includeRevoked bool, limit int, offset int, err error) {
	includeRevoked, err = parseProjectIncludeArchivedQuery(request.URL.Query().Get("include_revoked"))
	if err != nil {
		return false, 0, 0, errInvalidProjectQueryParam
	}
	limit, err = parseBoundedIntQueryParam(
		request.URL.Query().Get("limit"),
		defaultProjectMemberLimit,
		minProjectListLimit,
		maxProjectMemberLimit,
	)
	if err != nil {
		return false, 0, 0, errInvalidProjectQueryParam
	}
	offset, err = parseBoundedIntQueryParam(
		request.URL.Query().Get("offset"),
		defaultProjectMemberOffset,
		defaultProjectMemberOffset,
		int(^uint(0)>>1),
	)
	if err != nil {
		return false, 0, 0, errInvalidProjectQueryParam
	}
	return includeRevoked, limit, offset, nil
}
