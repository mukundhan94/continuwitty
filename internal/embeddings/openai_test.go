package embeddings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestOpenAIEmbeddingProviderEmbedManyOrdersByIndex(t *testing.T) {
	captured := openAIEmbeddingRequest{}
	server := newOpenAIEmbeddingCaptureServer(t, &captured)
	defer server.Close()

	provider, err := NewOpenAIEmbeddingProvider(
		OpenAIEmbeddingProviderInput{
			APIKey:     "sk-test",
			BaseURL:    server.URL,
			Model:      "text-embedding-3-small",
			HTTPClient: server.Client(),
		},
	)
	if err != nil {
		t.Fatalf("build openai embedding provider: %v", err)
	}
	vectors, err := provider.EmbedMany([]string{"alpha", "beta"}, 2)
	if err != nil {
		t.Fatalf("embed many: %v", err)
	}
	assertOpenAIEmbeddingRequestPayload(t, captured)
	assertOpenAIEmbeddingRequestInputs(t, captured.Input)
	assertOpenAIEmbeddingVectors(t, vectors)
}

func TestOpenAIEmbeddingProviderReturnsProviderErrorOnHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{"error":{"message":"upstream unavailable"}}`))
	}))
	defer server.Close()

	provider, err := NewOpenAIEmbeddingProvider(
		OpenAIEmbeddingProviderInput{
			APIKey:     "sk-test",
			BaseURL:    server.URL,
			Model:      "text-embedding-3-small",
			HTTPClient: server.Client(),
		},
	)
	if err != nil {
		t.Fatalf("build openai embedding provider: %v", err)
	}
	_, err = provider.Embed("alpha", 2)
	if err == nil {
		t.Fatalf("expected openai provider failure")
	}
	providerErr, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if providerErr.Message != "upstream unavailable" {
		t.Fatalf("expected provider error message, got %q", providerErr.Message)
	}
}

func newOpenAIEmbeddingCaptureServer(
	t *testing.T,
	captured *openAIEmbeddingRequest,
) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assertOpenAIEmbeddingsHTTPRequest(t, request)
		if err := json.NewDecoder(request.Body).Decode(captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"index":1,"embedding":[1.0,1.1]},{"index":0,"embedding":[0.0,0.1]}]}`))
	}))
}

func assertOpenAIEmbeddingsHTTPRequest(t *testing.T, request *http.Request) {
	t.Helper()
	if request.Method != http.MethodPost {
		t.Fatalf("expected POST method, got %s", request.Method)
	}
	if request.URL.Path != "/v1/embeddings" {
		t.Fatalf("expected embeddings path, got %s", request.URL.Path)
	}
	if request.Header.Get("Authorization") != "Bearer sk-test" {
		t.Fatalf("expected bearer auth header")
	}
}

func assertOpenAIEmbeddingRequestPayload(t *testing.T, payload openAIEmbeddingRequest) {
	t.Helper()
	if payload.Model != "text-embedding-3-small" {
		t.Fatalf("expected model text-embedding-3-small, got %q", payload.Model)
	}
	if payload.Dimensions != 2 {
		t.Fatalf("expected dimensions=2, got %d", payload.Dimensions)
	}
}

func assertOpenAIEmbeddingRequestInputs(t *testing.T, input any) {
	t.Helper()
	inputs, ok := input.([]any)
	if !ok || len(inputs) != 2 {
		t.Fatalf("expected []input with 2 items, got %#v", input)
	}
}

func assertOpenAIEmbeddingVectors(t *testing.T, vectors [][]float64) {
	t.Helper()
	if !reflect.DeepEqual(vectors, [][]float64{{0.0, 0.1}, {1.0, 1.1}}) {
		t.Fatalf("unexpected vectors order/content: %#v", vectors)
	}
}
