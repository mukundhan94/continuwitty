package embeddings

import "testing"

func TestEmbedTextLocalIsDeterministicAndFixedDim(t *testing.T) {
	first := mustEmbedLocal(t, "hello world", 16, "first")
	second := mustEmbedLocal(t, "hello world", 16, "second")
	assertVectorDimension(t, first, 16)
	assertVectorDimension(t, second, 16)
	assertVectorValuesInRange(t, first)
	assertVectorsMatch(t, first, second)
}

func TestEmbedTextLocalHandlesEmptyText(t *testing.T) {
	vector, err := EmbedTextLocal("", 8)
	if err != nil {
		t.Fatalf("expected embedding empty text to succeed: %v", err)
	}
	if len(vector) != 8 {
		t.Fatalf("expected dimension 8, got %d", len(vector))
	}
}

func TestEmbedTextLocalRejectsInvalidDim(t *testing.T) {
	_, err := EmbedTextLocal("abc", 0)
	if err == nil {
		t.Fatalf("expected invalid dim to fail")
	}
}

func mustEmbedLocal(t *testing.T, text string, dim int, label string) []float64 {
	t.Helper()
	vector, err := EmbedTextLocal(text, dim)
	if err != nil {
		t.Fatalf("expected %s embedding to succeed: %v", label, err)
	}
	return vector
}

func assertVectorDimension(t *testing.T, vector []float64, expected int) {
	t.Helper()
	if len(vector) != expected {
		t.Fatalf("expected dimension %d, got %d", expected, len(vector))
	}
}

func assertVectorValuesInRange(t *testing.T, vector []float64) {
	t.Helper()
	for index, value := range vector {
		if value < -1.0 || value > 1.0 {
			t.Fatalf("expected value at index %d to be within [-1,1], got %v", index, value)
		}
	}
}

func assertVectorsMatch(t *testing.T, expected []float64, actual []float64) {
	t.Helper()
	for index, value := range expected {
		if value != actual[index] {
			t.Fatalf("expected deterministic output at index %d", index)
		}
	}
}
