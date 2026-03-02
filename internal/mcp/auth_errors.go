package mcp

import "errors"

var (
	errBearerTokenMalformed = errors.New("authorization header must use bearer scheme")
	errBearerTokenMissing   = errors.New("bearer token missing")
)
