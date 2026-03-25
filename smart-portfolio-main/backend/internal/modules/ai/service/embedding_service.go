package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/config"
	"github.com/rs/zerolog/log"
)

// EmbeddingService handles communication with the Jina Embeddings API (which
// exposes an OpenAI-compatible /v1/embeddings endpoint). It converts text into
// dense vector representations used by the RAG pipeline and semantic cache.
//
// The service is safe for concurrent use — multiple goroutines can call Embed
// and EmbedBatch simultaneously. A sync.Pool is used to recycle HTTP request
// buffers and reduce GC pressure under high concurrency.
type EmbeddingService interface {
	// Embed returns the embedding vector for a single piece of text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch returns embedding vectors for multiple texts in a single API
	// call. The returned slice is in the same order as the input texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the expected dimensionality of the embedding vectors
	// produced by the configured model.
	Dimensions() int
}

// embeddingService is the concrete implementation of EmbeddingService backed
// by the Jina (OpenAI-compatible) embeddings API.
type embeddingService struct {
	apiKey     string
	baseURL    string
	model      string
	dimensions int
	client     *http.Client
	bufPool    sync.Pool
}

// NewEmbeddingService creates a new EmbeddingService configured with the
// provided embedding settings. It validates that the API key is present and
// initialises an HTTP client with sensible timeouts for embedding requests.
func NewEmbeddingService(cfg config.EmbeddingConfig) (EmbeddingService, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("embedding_service: JINA_API_KEY is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.jina.ai/v1"
	}

	model := cfg.Model
	if model == "" {
		model = "jina-embeddings-v2-base-en"
	}

	dimensions := cfg.Dimensions
	if dimensions <= 0 {
		dimensions = 768
	}

	svc := &embeddingService{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		model:      model,
		dimensions: dimensions,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		bufPool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}

	log.Info().
		Str("base_url", baseURL).
		Str("model", model).
		Int("dimensions", dimensions).
		Msg("embedding_service: initialized")

	return svc, nil
}

// Dimensions returns the expected vector dimensionality for the configured
// embedding model.
func (s *embeddingService) Dimensions() int {
	return s.dimensions
}

// Embed generates the embedding vector for a single text string by delegating
// to EmbedBatch with a single-element slice. This avoids code duplication
// while keeping the single-text API ergonomic.
func (s *embeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("embedding_service.Embed: text must not be empty")
	}

	results, err := s.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("embedding_service.Embed: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("embedding_service.Embed: API returned no embeddings")
	}

	return results[0], nil
}

// EmbedBatch sends multiple texts to the Jina embeddings API in a single HTTP
// request and returns the resulting vectors. The returned slice preserves the
// same ordering as the input texts.
//
// The Jina API is OpenAI-compatible, so the request and response formats
// follow the OpenAI /v1/embeddings specification:
//
//	POST /v1/embeddings
//	{
//	  "model": "jina-embeddings-v2-base-en",
//	  "input": ["text1", "text2", ...]
//	}
func (s *embeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("embedding_service.EmbedBatch: texts slice must not be empty")
	}

	start := time.Now()

	// Build the request payload.
	payload := embeddingRequest{
		Model: s.model,
		Input: texts,
	}

	buf := s.bufPool.Get().(*bytes.Buffer)
	defer s.bufPool.Put(buf)

	var body []byte
	var respStatusCode int
	maxRetries := 5
	backoff := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		buf.Reset()
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return nil, fmt.Errorf("embedding_service.EmbedBatch: failed to marshal request: %w", err)
		}

		// Build the HTTP request.
		endpoint := s.baseURL + "/embeddings"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, buf)
		if err != nil {
			return nil, fmt.Errorf("embedding_service.EmbedBatch: failed to create HTTP request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.apiKey)

		// Execute the request.
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("embedding_service.EmbedBatch: HTTP request failed: %w", err)
		}

		// Read the full response body so we can provide useful error messages.
		body, err = io.ReadAll(resp.Body)
		respStatusCode = resp.StatusCode
		resp.Body.Close()

		if err != nil {
			return nil, fmt.Errorf("embedding_service.EmbedBatch: failed to read response body: %w", err)
		}

		if respStatusCode == http.StatusTooManyRequests && attempt < maxRetries {
			log.Warn().
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("embedding_service: rate limit hit, retrying")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			continue
		}

		if respStatusCode < 200 || respStatusCode >= 300 {
			return nil, fmt.Errorf(
				"embedding_service.EmbedBatch: API returned status %d: %s",
				respStatusCode,
				truncateBytes(body, 512),
			)
		}

		break // Success, exit retry loop
	}

	// Parse the response.
	var embResp embeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("embedding_service.EmbedBatch: failed to decode response: %w", err)
	}

	if len(embResp.Data) != len(texts) {
		return nil, fmt.Errorf(
			"embedding_service.EmbedBatch: expected %d embeddings, got %d",
			len(texts), len(embResp.Data),
		)
	}

	// Sort results by index to ensure correct ordering (the API may return
	// them out of order, though in practice most providers preserve order).
	results := make([][]float32, len(texts))
	for _, item := range embResp.Data {
		if item.Index < 0 || item.Index >= len(texts) {
			return nil, fmt.Errorf(
				"embedding_service.EmbedBatch: response contains out-of-range index %d",
				item.Index,
			)
		}

		// Validate dimensions.
		if len(item.Embedding) != s.dimensions {
			return nil, fmt.Errorf(
				"embedding_service.EmbedBatch: embedding at index %d has %d dimensions, expected %d",
				item.Index, len(item.Embedding), s.dimensions,
			)
		}

		results[item.Index] = item.Embedding
	}

	// Verify no nil slots (would indicate duplicate or missing indices).
	for i, r := range results {
		if r == nil {
			return nil, fmt.Errorf(
				"embedding_service.EmbedBatch: missing embedding for index %d",
				i,
			)
		}
	}

	elapsed := time.Since(start)
	log.Debug().
		Int("texts", len(texts)).
		Dur("latency", elapsed).
		Msg("embedding_service: batch embedding completed")

	return results, nil
}

// ---------------------------------------------------------------------------
// OpenAI-compatible request/response types
// ---------------------------------------------------------------------------

// embeddingRequest is the JSON body sent to the /v1/embeddings endpoint.
type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embeddingResponse is the JSON body returned by the /v1/embeddings endpoint.
type embeddingResponse struct {
	Object string              `json:"object"`
	Data   []embeddingDataItem `json:"data"`
	Model  string              `json:"model"`
	Usage  *embeddingUsage     `json:"usage,omitempty"`
}

// embeddingDataItem represents a single embedding vector in the API response.
type embeddingDataItem struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

// embeddingUsage contains token usage information returned by some providers.
type embeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// truncateBytes truncates a byte slice to maxLen bytes and returns it as a
// string. If truncated, "..." is appended. Used for logging error response
// bodies without dumping potentially large payloads.
func truncateBytes(b []byte, maxLen int) string {
	if len(b) <= maxLen {
		return string(b)
	}
	return string(b[:maxLen]) + "..."
}
