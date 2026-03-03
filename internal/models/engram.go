package models

import (
	"fmt"
	"strings"
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
	Query                   string                  `json:"query"`
	TopK                    int                     `json:"top_k"`
	ProjectID               *string                 `json:"project_id,omitempty"`
	Tags                    []string                `json:"tags,omitempty"`
	Keywords                []string                `json:"keywords,omitempty"`
	CreatedAfter            *time.Time              `json:"created_after,omitempty"`
	CreatedBefore           *time.Time              `json:"created_before,omitempty"`
	UsefulCountMin          *int                    `json:"useful_count_min,omitempty"`
	UsefulCountMax          *int                    `json:"useful_count_max,omitempty"`
	AccessCountMin          *int                    `json:"access_count_min,omitempty"`
	AccessCountMax          *int                    `json:"access_count_max,omitempty"`
	FeedbackCountMin        *int                    `json:"feedback_count_min,omitempty"`
	FeedbackCountMax        *int                    `json:"feedback_count_max,omitempty"`
	ContradictionCountMin   *int                    `json:"contradiction_count_min,omitempty"`
	ContradictionCountMax   *int                    `json:"contradiction_count_max,omitempty"`
	ContradictionRatioMin   *float64                `json:"contradiction_feedback_ratio_min,omitempty"`
	ContradictionRatioMax   *float64                `json:"contradiction_feedback_ratio_max,omitempty"`
	FreshnessScoreMin       *float64                `json:"freshness_score_min,omitempty"`
	FreshnessScoreMax       *float64                `json:"freshness_score_max,omitempty"`
	UsefulFeedbackRatioMin  *float64                `json:"useful_feedback_ratio_min,omitempty"`
	UsefulFeedbackRatioMax  *float64                `json:"useful_feedback_ratio_max,omitempty"`
	AvgRelevanceFeedbackMin *float64                `json:"avg_relevance_feedback_min,omitempty"`
	AvgRelevanceFeedbackMax *float64                `json:"avg_relevance_feedback_max,omitempty"`
	SourceSessionQualityMin *float64                `json:"source_session_quality_min,omitempty"`
	SourceSessionQualityMax *float64                `json:"source_session_quality_max,omitempty"`
	LastAccessedAfter       *time.Time              `json:"last_accessed_after,omitempty"`
	LastAccessedBefore      *time.Time              `json:"last_accessed_before,omitempty"`
	FreshnessComputedAfter  *time.Time              `json:"freshness_computed_after,omitempty"`
	FreshnessComputedBefore *time.Time              `json:"freshness_computed_before,omitempty"`
	RelationType            *EngramLinkRelationType `json:"relation_type,omitempty"`
	TraceDepth              *int                    `json:"trace_depth,omitempty"`
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
	EngramID                   uuid.UUID  `json:"engram_id"`
	ProjectID                  string     `json:"project_id"`
	Title                      string     `json:"title"`
	Abstract                   string     `json:"abstract"`
	CreatedAt                  time.Time  `json:"created_at"`
	Tags                       []string   `json:"tags,omitempty"`
	Keywords                   []string   `json:"keywords,omitempty"`
	OwnerUserID                *uuid.UUID `json:"owner_user_id,omitempty"`
	VisibilityScope            string     `json:"visibility_scope"`
	AccessCount                int        `json:"access_count"`
	FreshnessScore             float64    `json:"freshness_score"`
	FeedbackCount              int        `json:"feedback_count"`
	UsefulCount                int        `json:"useful_count"`
	AvgRelevanceFeedback       float64    `json:"avg_relevance_feedback"`
	UsefulFeedbackRatio        float64    `json:"useful_feedback_ratio"`
	ContradictionCount         int        `json:"contradiction_count"`
	ContradictionFeedbackRatio float64    `json:"contradiction_feedback_ratio"`
	SourceSessionQualityScore  float64    `json:"source_session_quality_score"`
	Distance                   float64    `json:"distance"`
}

// EngramFeedbackType identifies explicit feedback semantics for one engram.
type EngramFeedbackType string

const (
	EngramFeedbackTypeUseful        EngramFeedbackType = "useful"
	EngramFeedbackTypeContradiction EngramFeedbackType = "contradiction"
)

// EngramFeedbackIntegrationDepth captures how deeply a recalled engram was integrated.
type EngramFeedbackIntegrationDepth string

const (
	EngramFeedbackIntegrationDepthMentioned    EngramFeedbackIntegrationDepth = "mentioned"
	EngramFeedbackIntegrationDepthElaborated   EngramFeedbackIntegrationDepth = "elaborated"
	EngramFeedbackIntegrationDepthContradicted EngramFeedbackIntegrationDepth = "contradicted"
	EngramFeedbackIntegrationDepthIgnored      EngramFeedbackIntegrationDepth = "ignored"
)

// ParseEngramFeedbackType normalizes an engram feedback type and validates it.
func ParseEngramFeedbackType(value string) (EngramFeedbackType, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	switch EngramFeedbackType(trimmed) {
	case EngramFeedbackTypeUseful:
		return EngramFeedbackTypeUseful, nil
	case EngramFeedbackTypeContradiction:
		return EngramFeedbackTypeContradiction, nil
	default:
		return "", fmt.Errorf("unsupported engram feedback type %q", value)
	}
}

// ParseEngramFeedbackIntegrationDepth normalizes an integration-depth value and validates it.
func ParseEngramFeedbackIntegrationDepth(value string) (EngramFeedbackIntegrationDepth, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	switch EngramFeedbackIntegrationDepth(trimmed) {
	case EngramFeedbackIntegrationDepthMentioned:
		return EngramFeedbackIntegrationDepthMentioned, nil
	case EngramFeedbackIntegrationDepthElaborated:
		return EngramFeedbackIntegrationDepthElaborated, nil
	case EngramFeedbackIntegrationDepthContradicted:
		return EngramFeedbackIntegrationDepthContradicted, nil
	case EngramFeedbackIntegrationDepthIgnored:
		return EngramFeedbackIntegrationDepthIgnored, nil
	default:
		return "", fmt.Errorf("unsupported engram feedback integration depth %q", value)
	}
}

// EngramFeedbackRecord captures one persisted explicit feedback event and updated counters.
type EngramFeedbackRecord struct {
	FeedbackID           uuid.UUID                       `json:"feedback_id"`
	EngramID             uuid.UUID                       `json:"engram_id"`
	SessionID            *uuid.UUID                      `json:"session_id,omitempty"`
	ActorUserID          uuid.UUID                       `json:"actor_user_id"`
	FeedbackType         EngramFeedbackType              `json:"feedback_type"`
	IntegrationDepth     *EngramFeedbackIntegrationDepth `json:"integration_depth,omitempty"`
	Note                 string                          `json:"note"`
	RelevanceScore       *int                            `json:"relevance_score,omitempty"`
	CreatedAt            time.Time                       `json:"created_at"`
	UsefulCount          int                             `json:"useful_count"`
	FeedbackCount        int                             `json:"feedback_count"`
	AvgRelevanceFeedback *float64                        `json:"avg_relevance_feedback,omitempty"`
	ContradictionCount   int                             `json:"contradiction_count"`
}

// EngramFeedbackCreateRequest models explicit feedback submission payload.
type EngramFeedbackCreateRequest struct {
	FeedbackType     string  `json:"feedback_type"`
	IntegrationDepth *string `json:"integration_depth,omitempty"`
	Note             *string `json:"note,omitempty"`
	RelevanceScore   *int    `json:"relevance_score,omitempty"`
	SessionID        *string `json:"session_id,omitempty"`
}

// EngramCreateResponse is returned when a new engram is persisted.
type EngramCreateResponse struct {
	EngramID           uuid.UUID `json:"engram_id"`
	CreatedAt          time.Time `json:"created_at"`
	ResolvedProjectID  *string   `json:"resolved_project_id,omitempty"`
	UsedDefaultProject bool      `json:"used_default_project"`
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

// AdminEngramSourceInput models source replacement payloads in memory-admin flows.
type AdminEngramSourceInput struct {
	CapturedAt  time.Time `json:"captured_at"`
	URL         string    `json:"url"`
	Title       *string   `json:"title,omitempty"`
	Snippet     *string   `json:"snippet,omitempty"`
	ContentText *string   `json:"content_text,omitempty"`
	ContentHash *string   `json:"content_hash,omitempty"`
}

// AdminEngramSourceRecord models admin source rows associated with an engram.
type AdminEngramSourceRecord struct {
	SourceID    uuid.UUID `json:"source_id"`
	CapturedAt  time.Time `json:"captured_at"`
	URL         string    `json:"url"`
	Title       *string   `json:"title,omitempty"`
	Snippet     *string   `json:"snippet,omitempty"`
	ContentText *string   `json:"content_text,omitempty"`
	ContentHash *string   `json:"content_hash,omitempty"`
}

// AdminEngramRecord models admin-level engram rows with lifecycle metadata.
type AdminEngramRecord struct {
	EngramID                uuid.UUID                 `json:"engram_id"`
	ProjectID               string                    `json:"project_id"`
	ThreadID                *string                   `json:"thread_id,omitempty"`
	Title                   string                    `json:"title"`
	Abstract                string                    `json:"abstract"`
	DetailedSummaryMarkdown string                    `json:"detailed_summary_markdown"`
	Tags                    []string                  `json:"tags,omitempty"`
	Keywords                []string                  `json:"keywords,omitempty"`
	OwnerUserID             *uuid.UUID                `json:"owner_user_id,omitempty"`
	VisibilityScope         VisibilityScope           `json:"visibility_scope"`
	SourceSessionID         *uuid.UUID                `json:"source_session_id,omitempty"`
	CreatedAt               time.Time                 `json:"created_at"`
	UpdatedAt               time.Time                 `json:"updated_at"`
	DeletedAt               *time.Time                `json:"deleted_at"`
	DeletedByUserID         *uuid.UUID                `json:"deleted_by_user_id,omitempty"`
	DeleteReason            *string                   `json:"delete_reason,omitempty"`
	Sources                 []AdminEngramSourceRecord `json:"sources,omitempty"`
}
