package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DocumentSourceType identifies how a document was ingested.
type DocumentSourceType string

const (
	DocumentSourceTypeText DocumentSourceType = "text"
	DocumentSourceTypeFile DocumentSourceType = "file"
)

// ParseDocumentSourceType validates document source type inputs.
func ParseDocumentSourceType(value string) (DocumentSourceType, error) {
	switch DocumentSourceType(value) {
	case DocumentSourceTypeText, DocumentSourceTypeFile:
		return DocumentSourceType(value), nil
	default:
		return "", fmt.Errorf("unsupported document source type %q", value)
	}
}

// DocumentRecord models persisted document rows.
type DocumentRecord struct {
	DocumentID      uuid.UUID          `json:"document_id"`
	OwnerUserID     uuid.UUID          `json:"owner_user_id"`
	ProjectID       string             `json:"project_id"`
	Title           string             `json:"title"`
	SourceType      DocumentSourceType `json:"source_type"`
	SourceName      *string            `json:"source_name,omitempty"`
	MimeType        *string            `json:"mime_type,omitempty"`
	VisibilityScope VisibilityScope    `json:"visibility_scope"`
	ContentHash     string             `json:"content_hash"`
	ChunkCount      int                `json:"chunk_count"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// DocumentChunkQueryRequest models retrieval constraints for document chunk search.
type DocumentChunkQueryRequest struct {
	Query       string      `json:"query"`
	ProjectID   *string     `json:"project_id,omitempty"`
	DocumentIDs []uuid.UUID `json:"document_ids,omitempty"`
	TopK        int         `json:"top_k"`
}

// DocumentChunkQueryResult models ranked retrieval rows for document chunks.
type DocumentChunkQueryResult struct {
	ChunkID         uuid.UUID       `json:"chunk_id"`
	DocumentID      uuid.UUID       `json:"document_id"`
	ProjectID       string          `json:"project_id"`
	Title           string          `json:"title"`
	SourceName      *string         `json:"source_name,omitempty"`
	ChunkIndex      int             `json:"chunk_index"`
	Snippet         string          `json:"snippet"`
	CreatedAt       time.Time       `json:"created_at"`
	VisibilityScope VisibilityScope `json:"visibility_scope"`
	Distance        float64         `json:"distance"`
}
