package embeddings

// ProviderError marks provider execution failures that may trigger fallback.
type ProviderError struct {
	Message string
}

func (e *ProviderError) Error() string {
	return e.Message
}
