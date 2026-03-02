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

type snippetBuildInput struct {
	chunkText    string
	snippetChars int
}

type nextChunkStartInput struct {
	currentStart     int
	end              int
	chunkOverlapChar int
}

type chunkEndPickInput struct {
	runes          []rune
	start          int
	hardEnd        int
	chunkSizeChars int
}

type tokenSearchInput struct {
	runes []rune
	token []rune
	start int
	end   int
}

type tokenAtInput struct {
	runes []rune
	index int
	token []rune
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
		end := pickChunkEnd(chunkEndPickInput{
			runes:          runes,
			start:          start,
			hardEnd:        hardEnd,
			chunkSizeChars: input.ChunkSizeChars,
		})
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
		start = nextChunkStart(nextChunkStartInput{
			currentStart:     start,
			end:              end,
			chunkOverlapChar: chunkOverlap,
		})
	}
	return chunks
}

func buildChunkDraft(input chunkDraftBuildInput) ChunkDraft {
	chunkID := uuid.NewSHA1(
		chunkNamespace,
		[]byte(fmt.Sprintf("%s:%d:%d:%d", input.contentHash, input.chunkIndex, input.charStart, input.charEnd)),
	)
	return ChunkDraft{
		ChunkID:    chunkID,
		ChunkIndex: input.chunkIndex,
		ChunkText:  input.chunkText,
		Snippet: buildSnippet(snippetBuildInput{
			chunkText:    input.chunkText,
			snippetChars: input.snippetChars,
		}),
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

func buildSnippet(input snippetBuildInput) string {
	chunkRunes := []rune(input.chunkText)
	snippetEnd := minInt(input.snippetChars, len(chunkRunes))
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

func nextChunkStart(input nextChunkStartInput) int {
	nextStart := maxInt(0, input.end-input.chunkOverlapChar)
	if nextStart <= input.currentStart {
		return input.end
	}
	return nextStart
}

func pickChunkEnd(input chunkEndPickInput) int {
	if input.hardEnd >= len(input.runes) {
		return len(input.runes)
	}
	searchStart := input.start + int(float64(input.chunkSizeChars)*minBreakFraction)
	if searchStart >= input.hardEnd {
		return input.hardEnd
	}

	chosen := maxInt(
		findLastTokenIndex(tokenSearchInput{
			runes: input.runes, token: doubleNewlineRune, start: searchStart, end: input.hardEnd,
		}),
		maxInt(
			findLastTokenIndex(tokenSearchInput{
				runes: input.runes, token: singleNewlineRune, start: searchStart, end: input.hardEnd,
			}),
			findLastTokenIndex(tokenSearchInput{
				runes: input.runes, token: spaceRune, start: searchStart, end: input.hardEnd,
			}),
		),
	)
	if chosen <= input.start {
		return input.hardEnd
	}
	if hasTokenAt(tokenAtInput{
		runes: input.runes, index: chosen, token: doubleNewlineRune,
	}) {
		return chosen
	}
	return chosen + 1
}

func findLastTokenIndex(input tokenSearchInput) int {
	if len(input.token) == 0 || input.end-input.start < len(input.token) {
		return -1
	}
	for index := input.end - len(input.token); index >= input.start; index-- {
		if hasTokenAt(tokenAtInput{
			runes: input.runes, index: index, token: input.token,
		}) {
			return index
		}
	}
	return -1
}

func hasTokenAt(input tokenAtInput) bool {
	if input.index < 0 || input.index+len(input.token) > len(input.runes) {
		return false
	}
	for tokenIndex := 0; tokenIndex < len(input.token); tokenIndex++ {
		if input.runes[input.index+tokenIndex] != input.token[tokenIndex] {
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
