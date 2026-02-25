package ingestion

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestChunkDocumentTextIsDeterministicForSameInput(t *testing.T) {
	text := strings.Join([]string{
		"Incident started at 14:32 UTC with elevated latency across write path.",
		"Mitigation introduced read-only cache and connection pool cap.",
		"Service recovered by 17:02 UTC after query rollback and index hint.",
	}, "\n\n")
	contentHash := BuildContentHash(text)
	input := ChunkDocumentInput{
		ContentHash:       contentHash,
		Text:              text,
		ChunkSizeChars:    90,
		ChunkOverlapChars: 18,
	}

	first := ChunkDocumentText(input)
	second := ChunkDocumentText(input)

	if len(first) < 2 {
		t.Fatalf("expected at least two chunks, got %d", len(first))
	}
	requireEqualStringSlice(t, collectChunkIDs(first), collectChunkIDs(second))
	requireEqualStringSlice(t, collectChunkTexts(first), collectChunkTexts(second))
}

func TestBuildDocumentIDIsStableForSameFingerprint(t *testing.T) {
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	contentHash := BuildContentHash("same content")

	first := BuildDocumentID(ownerID, "engram-vault", "Runbook", contentHash)
	second := BuildDocumentID(ownerID, "engram-vault", "Runbook", contentHash)
	if first != second {
		t.Fatalf("expected stable document id, got %s and %s", first, second)
	}
}

func TestNormalizeDocumentTextFlattensMixedNewlines(t *testing.T) {
	source := "Line A\r\nLine B\rLine C\n"
	normalized := NormalizeDocumentText(source)
	if normalized != "Line A\nLine B\nLine C" {
		t.Fatalf("expected normalized text, got %q", normalized)
	}
}

func collectChunkIDs(chunks []ChunkDraft) []string {
	ids := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		ids = append(ids, chunk.ChunkID.String())
	}
	return ids
}

func collectChunkTexts(chunks []ChunkDraft) []string {
	texts := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		texts = append(texts, chunk.ChunkText)
	}
	return texts
}

func requireEqualStringSlice(t *testing.T, expected []string, actual []string) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Fatalf("expected slice length %d, got %d", len(expected), len(actual))
	}
	for index := 0; index < len(expected); index++ {
		if expected[index] != actual[index] {
			t.Fatalf("expected element %d to be %q, got %q", index, expected[index], actual[index])
		}
	}
}
