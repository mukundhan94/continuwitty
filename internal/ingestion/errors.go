package ingestion

import "fmt"

// ServiceError mirrors Python's ingestion service error shape.
type ServiceError struct {
	Detail     string
	StatusCode int
}

func (err ServiceError) Error() string {
	if err.Detail == "" {
		return "ingestion service error"
	}
	return err.Detail
}

func newServiceError(detail string, statusCode int) error {
	return ServiceError{Detail: detail, StatusCode: statusCode}
}

func newDefaultServiceError(detail string) error {
	return newServiceError(detail, 400)
}

func invalidChunkShapeError() error {
	return newServiceError("chunk_overlap_chars must be smaller than chunk_size_chars", 422)
}

func oversizedTextError(maxTextChars int) error {
	return newServiceError(
		fmt.Sprintf("Document text exceeds max allowed size of %d characters", maxTextChars),
		413,
	)
}

func oversizedFileError(maxFileBytes int) error {
	return newServiceError(
		fmt.Sprintf("Uploaded file exceeds max allowed size of %d bytes", maxFileBytes),
		413,
	)
}
