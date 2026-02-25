package ingestion

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	minBreakFraction   = 0.6
	defaultSnippetSize = 220
)

var (
	documentNamespace = uuid.MustParse("6e5bc7c9-4930-4d8a-a62e-d5bdabed95e3")
	chunkNamespace    = uuid.MustParse("5ecdb89d-6222-4fbe-8ef2-9298dc5d8f0e")
	doubleNewlineRune = []rune("\n\n")
	singleNewlineRune = []rune("\n")
	spaceRune         = []rune(" ")
)

// ChunkDraft captures deterministic chunk boundaries and metadata for persistence.
type ChunkDraft struct {
	ChunkID       uuid.UUID      `json:"chunk_id"`
	ChunkIndex    int            `json:"chunk_index"`
	ChunkText     string         `json:"chunk_text"`
	Snippet       string         `json:"snippet"`
	CharStart     int            `json:"char_start"`
	CharEnd       int            `json:"char_end"`
	TokenEstimate int            `json:"token_estimate"`
	Metadata      map[string]any `json:"metadata"`
}

// ChunkDocumentInput captures chunking parameters.
type ChunkDocumentInput struct {
	ContentHash       string
	Text              string
	ChunkSizeChars    int
	ChunkOverlapChars int
	SnippetChars      int
}

type chunkDraftBuildInput struct {
	contentHash       string
	chunkIndex        int
	chunkText         string
	charStart         int
	charEnd           int
	chunkSizeChars    int
	chunkOverlapChars int
	snippetChars      int
}

// NormalizeDocumentText normalizes whitespace/newlines to keep hashing and chunk IDs deterministic.
func NormalizeDocumentText(text string) string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.TrimSpace(normalized)
}

// BuildContentHash returns a SHA256 digest for normalized text.
func BuildContentHash(text string) string {
	sum := sha256.Sum256([]byte(NormalizeDocumentText(text)))
	return hex.EncodeToString(sum[:])
}

// BuildDocumentID returns a deterministic UUIDv5-style SHA1 namespace UUID.
func BuildDocumentID(ownerUserID uuid.UUID, projectID string, title string, contentHash string) uuid.UUID {
	fingerprint := fmt.Sprintf("%s:%s:%s:%s", ownerUserID, projectID, strings.ToLower(strings.TrimSpace(title)), contentHash)
	return uuid.NewSHA1(documentNamespace, []byte(fingerprint))
}

// EstimateTokenCount estimates token count using whitespace splitting.
func EstimateTokenCount(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	return len(strings.Fields(trimmed))
}

// ChunkDocumentText builds deterministic chunk boundaries and UUIDs for retrieval chunks.
func ChunkDocumentText(input ChunkDocumentInput) []ChunkDraft {
	normalizedText := NormalizeDocumentText(input.Text)
	if normalizedText == "" || input.ChunkSizeChars <= 0 {
		return []ChunkDraft{}
	}
	runes := []rune(normalizedText)
	chunkOverlap := normalizeChunkOverlap(input.ChunkOverlapChars)
	snippetSize := normalizeSnippetSize(input.SnippetChars)

	chunks := make([]ChunkDraft, 0)
	start := 0
	chunkIndex := 0

	for start < len(runes) {
		hardEnd := minInt(start+input.ChunkSizeChars, len(runes))
		end := pickChunkEnd(runes, start, hardEnd, input.ChunkSizeChars)
		chunkText := strings.TrimSpace(string(runes[start:end]))
		if chunkText != "" {
			chunks = append(chunks, buildChunkDraft(chunkDraftBuildInput{
				contentHash:       input.ContentHash,
				chunkIndex:        chunkIndex,
				chunkText:         chunkText,
				charStart:         start,
				charEnd:           end,
				chunkSizeChars:    input.ChunkSizeChars,
				chunkOverlapChars: chunkOverlap,
				snippetChars:      snippetSize,
			}))
			chunkIndex++
		}
		if end >= len(runes) {
			break
		}
		start = nextChunkStart(start, end, chunkOverlap)
	}
	return chunks
}

func buildChunkDraft(input chunkDraftBuildInput) ChunkDraft {
	chunkID := uuid.NewSHA1(
		chunkNamespace,
		[]byte(fmt.Sprintf("%s:%d:%d:%d", input.contentHash, input.chunkIndex, input.charStart, input.charEnd)),
	)
	return ChunkDraft{
		ChunkID:       chunkID,
		ChunkIndex:    input.chunkIndex,
		ChunkText:     input.chunkText,
		Snippet:       buildSnippet(input.chunkText, input.snippetChars),
		CharStart:     input.charStart,
		CharEnd:       input.charEnd,
		TokenEstimate: EstimateTokenCount(input.chunkText),
		Metadata: map[string]any{
			"content_hash":        input.contentHash,
			"char_start":          input.charStart,
			"char_end":            input.charEnd,
			"chunk_size_chars":    input.chunkSizeChars,
			"chunk_overlap_chars": input.chunkOverlapChars,
		},
	}
}

func buildSnippet(chunkText string, snippetChars int) string {
	chunkRunes := []rune(chunkText)
	snippetEnd := minInt(snippetChars, len(chunkRunes))
	snippet := string(chunkRunes[:snippetEnd])
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	return strings.TrimSpace(snippet)
}

func normalizeChunkOverlap(chunkOverlapChars int) int {
	if chunkOverlapChars < 0 {
		return 0
	}
	return chunkOverlapChars
}

func normalizeSnippetSize(snippetChars int) int {
	if snippetChars <= 0 {
		return defaultSnippetSize
	}
	return snippetChars
}

func nextChunkStart(currentStart int, end int, chunkOverlapChars int) int {
	nextStart := maxInt(0, end-chunkOverlapChars)
	if nextStart <= currentStart {
		return end
	}
	return nextStart
}

func pickChunkEnd(runes []rune, start int, hardEnd int, chunkSizeChars int) int {
	if hardEnd >= len(runes) {
		return len(runes)
	}
	searchStart := start + int(float64(chunkSizeChars)*minBreakFraction)
	if searchStart >= hardEnd {
		return hardEnd
	}

	chosen := maxInt(
		findLastTokenIndex(runes, doubleNewlineRune, searchStart, hardEnd),
		maxInt(
			findLastTokenIndex(runes, singleNewlineRune, searchStart, hardEnd),
			findLastTokenIndex(runes, spaceRune, searchStart, hardEnd),
		),
	)
	if chosen <= start {
		return hardEnd
	}
	if hasTokenAt(runes, chosen, doubleNewlineRune) {
		return chosen
	}
	return chosen + 1
}

func findLastTokenIndex(runes []rune, token []rune, start int, end int) int {
	if len(token) == 0 || end-start < len(token) {
		return -1
	}
	for index := end - len(token); index >= start; index-- {
		if hasTokenAt(runes, index, token) {
			return index
		}
	}
	return -1
}

func hasTokenAt(runes []rune, index int, token []rune) bool {
	if index < 0 || index+len(token) > len(runes) {
		return false
	}
	for tokenIndex := 0; tokenIndex < len(token); tokenIndex++ {
		if runes[index+tokenIndex] != token[tokenIndex] {
			return false
		}
	}
	return true
}

func maxInt(first int, second int) int {
	if first > second {
		return first
	}
	return second
}

func minInt(first int, second int) int {
	if first < second {
		return first
	}
	return second
}
