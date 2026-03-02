package mcp

func invalidParamError(field string) *toolDispatchError {
	return &toolDispatchError{
		code:    -32602,
		message: "Invalid params",
		data:    map[string]any{"invalid": field},
	}
}

func missingParamError(field string) *toolDispatchError {
	return &toolDispatchError{
		code:    -32602,
		message: "Invalid params",
		data:    map[string]any{"missing": field},
	}
}

func internalToolDispatchError() *toolDispatchError {
	return &toolDispatchError{
		code:    -32603,
		message: "Internal error",
	}
}

func invalidParamsWithStatus(statusCode int, detail string) *toolDispatchError {
	return &toolDispatchError{
		code:    -32602,
		message: "Invalid params",
		data: map[string]any{
			"status_code": statusCode,
			"detail":      detail,
		},
	}
}
