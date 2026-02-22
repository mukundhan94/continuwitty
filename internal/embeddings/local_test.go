package embeddings

import "testing"

func TestEmbedTextLocalIsDeterministicAndFixedDim(t *testing.T) {
	first, err := EmbedTextLocal("hello world", 16)
	if err != nil {
		t.Fatalf("expected first embedding to succeed: %v", err)
	}
	second, err := EmbedTextLocal("hello world", 16)
	if err != nil {
		t.Fatalf("expected second embedding to succeed: %v", err)
	}

	if len(first) != 16 || len(second) != 16 {
		t.Fatalf("expected fixed embedding dimension 16")
	}
	for index, value := range first {
		if value < -1.0 || value > 1.0 {
			t.Fatalf("expected value at index %d to be within [-1,1], got %v", index, value)
		}
		if value != second[index] {
			t.Fatalf("expected deterministic output at index %d", index)
		}
	}
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
