package models

import (
	"time"

	"github.com/google/uuid"
)

// SupportingSource tracks citation evidence associated with a claim.
type SupportingSource struct {
	URL        string    `json:"url"`
	Title      *string   `json:"title,omitempty"`
	Snippet    string    `json:"snippet"`
	CapturedAt time.Time `json:"captured_at"`
}

// Claim captures a statement plus supporting evidence.
type Claim struct {
	Claim             string             `json:"claim"`
	SupportingSources []SupportingSource `json:"supporting_sources,omitempty"`
}

// Decision captures a decision and rationale pair.
type Decision struct {
	Decision  string `json:"decision"`
	Rationale string `json:"rationale"`
}

// ArtifactIn models artifact metadata attached to an engram.
type ArtifactIn struct {
	ArtifactType string         `json:"artifact_type"`
	StorageURI   string         `json:"storage_uri"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// MemoryEngramCreate represents the engram creation payload shape.
type MemoryEngramCreate struct {
	ProjectID               string       `json:"project_id"`
	ThreadID                *string      `json:"thread_id,omitempty"`
	Title                   string       `json:"title"`
	Abstract                string       `json:"abstract"`
	DetailedSummaryMarkdown string       `json:"detailed_summary_markdown"`
	Decisions               []Decision   `json:"decisions,omitempty"`
	Assumptions             []string     `json:"assumptions,omitempty"`
	OpenQuestions           []string     `json:"open_questions,omitempty"`
	Claims                  []Claim      `json:"claims,omitempty"`
	Tags                    []string     `json:"tags,omitempty"`
	Keywords                []string     `json:"keywords,omitempty"`
	Artifacts               []ArtifactIn `json:"artifacts,omitempty"`
	RetrievalText           *string      `json:"retrieval_text,omitempty"`
	VisibilityScope         string       `json:"visibility_scope"`
	SourceSessionID         *uuid.UUID   `json:"source_session_id,omitempty"`
}

// EngramQueryRequest models vector/lexical query constraints.
type EngramQueryRequest struct {
	Query         string     `json:"query"`
	TopK          int        `json:"top_k"`
	ProjectID     *string    `json:"project_id,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	Keywords      []string   `json:"keywords,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
}

// RehydrationCitation models a source snippet used in rehydration output.
type RehydrationCitation struct {
	URL        string    `json:"url"`
	Title      *string   `json:"title,omitempty"`
	Snippet    string    `json:"snippet"`
	CapturedAt time.Time `json:"captured_at"`
}

// EngramSummary models list-engrams response rows.
type EngramSummary struct {
	EngramID        uuid.UUID  `json:"engram_id"`
	ProjectID       string     `json:"project_id"`
	ThreadID        *string    `json:"thread_id,omitempty"`
	Title           string     `json:"title"`
	Abstract        string     `json:"abstract"`
	CreatedAt       time.Time  `json:"created_at"`
	Tags            []string   `json:"tags,omitempty"`
	Keywords        []string   `json:"keywords,omitempty"`
	OwnerUserID     *uuid.UUID `json:"owner_user_id,omitempty"`
	VisibilityScope string     `json:"visibility_scope"`
}

// EngramQueryResult models query-engrams response rows.
type EngramQueryResult struct {
	EngramID        uuid.UUID  `json:"engram_id"`
	ProjectID       string     `json:"project_id"`
	Title           string     `json:"title"`
	Abstract        string     `json:"abstract"`
	CreatedAt       time.Time  `json:"created_at"`
	Tags            []string   `json:"tags,omitempty"`
	Keywords        []string   `json:"keywords,omitempty"`
	OwnerUserID     *uuid.UUID `json:"owner_user_id,omitempty"`
	VisibilityScope string     `json:"visibility_scope"`
	Distance        float64    `json:"distance"`
}

// RehydrationBundle models context returned for rehydration consumers.
type RehydrationBundle struct {
	EngramID                uuid.UUID             `json:"engram_id"`
	ProjectID               string                `json:"project_id"`
	Title                   string                `json:"title"`
	CompactSummary          string                `json:"compact_summary"`
	DetailedSummaryMarkdown string                `json:"detailed_summary_markdown"`
	KeyDecisions            []map[string]any      `json:"key_decisions,omitempty"`
	OpenQuestions           []string              `json:"open_questions,omitempty"`
	TopCitations            []RehydrationCitation `json:"top_citations,omitempty"`
	ContextMarkdown         string                `json:"context_markdown"`
	OwnerUserID             *uuid.UUID            `json:"owner_user_id,omitempty"`
	VisibilityScope         string                `json:"visibility_scope"`
}

// EngramSourceRecord models source rows tied to an engram.
type EngramSourceRecord struct {
	SourceID   uuid.UUID `json:"source_id"`
	EngramID   uuid.UUID `json:"engram_id"`
	CapturedAt time.Time `json:"captured_at"`
	URL        string    `json:"url"`
	Title      *string   `json:"title,omitempty"`
	Snippet    *string   `json:"snippet,omitempty"`
}
