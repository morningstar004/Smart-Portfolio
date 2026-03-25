package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/config"
	"github.com/ZRishu/smart-portfolio/internal/modules/ai/dto"
	"github.com/ZRishu/smart-portfolio/internal/modules/ai/repository"
	"github.com/rs/zerolog/log"
)

// RAGService implements the Hybrid RAG (Retrieval-Augmented Generation) pipeline.
// For every incoming question it:
//
//  1. Embeds the question via the EmbeddingService.
//  2. Checks the semantic cache for a previously computed answer (cosine distance < threshold).
//  3. On cache miss, performs a pgvector similarity search against resume embeddings.
//  4. Builds a system prompt with the retrieved context and streams the answer
//     from the Groq LLM (OpenAI-compatible chat completions endpoint).
//  5. After the stream completes, saves the full response to the semantic cache
//     in a background goroutine so the next similar question is served instantly.
type RAGService interface {
	// AskQuestion performs the full RAG pipeline and returns a complete (non-streaming)
	// answer. Useful for simple API consumers that don't need SSE.
	AskQuestion(ctx context.Context, req dto.ChatRequest) (*dto.ChatResponse, error)

	// StreamQuestion performs the RAG pipeline and writes Server-Sent Events to
	// the provided http.ResponseWriter. The writer must support flushing. The
	// function blocks until the entire stream is consumed or the context is cancelled.
	StreamQuestion(ctx context.Context, w http.ResponseWriter, req dto.ChatRequest) error
}

// ragService is the concrete implementation of RAGService.
type ragService struct {
	embeddingSvc EmbeddingService
	cacheRepo    *repository.SemanticCacheRepository
	vectorRepo   *repository.VectorStoreRepository
	aiCfg        config.AIConfig
	httpClient   *http.Client
	bufPool      sync.Pool
	cacheWg      sync.WaitGroup
}

// NewRAGService creates a new RAGService wired to the embedding service, semantic
// cache repository, vector store repository, and Groq AI configuration.
func NewRAGService(
	embeddingSvc EmbeddingService,
	cacheRepo *repository.SemanticCacheRepository,
	vectorRepo *repository.VectorStoreRepository,
	aiCfg config.AIConfig,
) RAGService {
	svc := &ragService{
		embeddingSvc: embeddingSvc,
		cacheRepo:    cacheRepo,
		vectorRepo:   vectorRepo,
		aiCfg:        aiCfg,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // LLM streaming can be slow; generous timeout.
		},
		bufPool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}

	log.Info().
		Str("model", aiCfg.Model).
		Str("base_url", aiCfg.BaseURL).
		Float32("temperature", aiCfg.Temperature).
		Msg("rag_service: initialized")

	return svc
}

// ShutdownCacheWorkers waits for any background goroutines that are saving
// responses to the semantic cache. Call this during graceful shutdown to avoid
// losing cache entries.
func (s *ragService) ShutdownCacheWorkers() {
	s.cacheWg.Wait()
}

// ---------------------------------------------------------------------------
// AskQuestion (non-streaming)
// ---------------------------------------------------------------------------

func (s *ragService) AskQuestion(ctx context.Context, req dto.ChatRequest) (*dto.ChatResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("rag_service.AskQuestion: %w", err)
	}

	question := strings.TrimSpace(req.Question)
	log.Info().Str("question", truncateStr(question, 100)).Msg("rag_service: received question (non-streaming)")

	// Step 1: Embed the question.
	embedding, err := s.embeddingSvc.Embed(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("rag_service.AskQuestion: embedding failed: %w", err)
	}

	// Step 2: Check semantic cache.
	cached, found, err := s.cacheRepo.FindCachedResponse(ctx, embedding)
	if err != nil {
		log.Warn().Err(err).Msg("rag_service: semantic cache lookup failed — continuing without cache")
	}
	if found {
		log.Info().Msg("rag_service: semantic cache HIT — returning cached response")
		return &dto.ChatResponse{
			Answer: cached,
			Cached: true,
		}, nil
	}

	log.Info().Msg("rag_service: semantic cache MISS — querying vector store and LLM")

	// Step 3: RAG retrieval — find the top-K most relevant resume chunks.
	contextText, err := s.retrieveContext(ctx, embedding, 3)
	if err != nil {
		return nil, fmt.Errorf("rag_service.AskQuestion: %w", err)
	}

	// Step 4: Call the LLM (non-streaming).
	answer, err := s.callLLM(ctx, question, contextText, false)
	if err != nil {
		return nil, fmt.Errorf("rag_service.AskQuestion: %w", err)
	}

	// Step 5: Save to semantic cache in a background goroutine.
	s.saveToCacheAsync(question, embedding, answer)

	return &dto.ChatResponse{
		Answer: answer,
		Cached: false,
	}, nil
}

// ---------------------------------------------------------------------------
// StreamQuestion (SSE streaming)
// ---------------------------------------------------------------------------

func (s *ragService) StreamQuestion(ctx context.Context, w http.ResponseWriter, req dto.ChatRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("rag_service.StreamQuestion: %w", err)
	}

	question := strings.TrimSpace(req.Question)
	log.Info().Str("question", truncateStr(question, 100)).Msg("rag_service: received question (streaming)")

	// Step 1: Embed the question.
	embedding, err := s.embeddingSvc.Embed(ctx, question)
	if err != nil {
		return fmt.Errorf("rag_service.StreamQuestion: embedding failed: %w", err)
	}

	// Step 2: Check semantic cache.
	cached, found, err := s.cacheRepo.FindCachedResponse(ctx, embedding)
	if err != nil {
		log.Warn().Err(err).Msg("rag_service: semantic cache lookup failed — continuing without cache")
	}

	// Set SSE headers before writing any bytes.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering if behind a reverse proxy.

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("rag_service.StreamQuestion: ResponseWriter does not support flushing")
	}

	if found {
		log.Info().Msg("rag_service: semantic cache HIT — streaming cached response via SSE")
		writeSSEData(w, flusher, cached)
		writeSSEEvent(w, flusher, "done", "")
		return nil
	}

	log.Info().Msg("rag_service: semantic cache MISS — streaming from LLM via SSE")

	// Step 3: RAG retrieval.
	contextText, err := s.retrieveContext(ctx, embedding, 3)
	if err != nil {
		return fmt.Errorf("rag_service.StreamQuestion: %w", err)
	}

	// Step 4: Stream from the LLM.
	fullResponse, err := s.streamFromLLM(ctx, w, flusher, question, contextText)
	if err != nil {
		return fmt.Errorf("rag_service.StreamQuestion: %w", err)
	}

	// Step 5: Send the terminal SSE event.
	writeSSEEvent(w, flusher, "done", "")

	// Step 6: Save full response to semantic cache in a background goroutine.
	if fullResponse != "" {
		s.saveToCacheAsync(question, embedding, fullResponse)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// retrieveContext performs a pgvector similarity search and concatenates the
// content of the top-K most relevant resume chunks into a single context string.
func (s *ragService) retrieveContext(ctx context.Context, queryEmbedding []float32, topK int) (string, error) {
	docs, err := s.vectorRepo.SimilaritySearch(ctx, queryEmbedding, topK)
	if err != nil {
		return "", fmt.Errorf("retrieveContext: similarity search failed: %w", err)
	}

	if len(docs) == 0 {
		log.Warn().Msg("rag_service: no relevant documents found in vector store")
		return "", nil
	}

	var sb strings.Builder
	for i, doc := range docs {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(doc.Content)
	}

	log.Debug().
		Int("chunks", len(docs)).
		Int("context_length", sb.Len()).
		Msg("rag_service: retrieved context from vector store")

	return sb.String(), nil
}

// systemPrompt builds the system prompt that constrains the LLM to answer only
// from the provided resume context. This is the core of the RAG pattern —
// grounding the LLM's responses in factual, retrieved information.
func systemPrompt(contextText string) string {
	return fmt.Sprintf(`You are the personal AI assistant for ZR, a backend developer specializing in Java.
Your ONLY purpose is to answer questions about ZR using the provided context about his skills, projects, and experience.
If the answer is not in the context, politely state that you don't have that specific information and encourage them to use the contact form.
Do not answer general knowledge questions or questions unrelated to ZR.

Be concise, professional, and helpful. Use markdown formatting when appropriate.

CONTEXT:
%s`, contextText)
}

// callLLM sends a non-streaming chat completion request to the Groq API and
// returns the full assistant message content.
func (s *ragService) callLLM(ctx context.Context, question, contextText string, stream bool) (string, error) {
	payload := chatCompletionRequest{
		Model:       s.aiCfg.Model,
		Temperature: s.aiCfg.Temperature,
		Stream:      stream,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt(contextText)},
			{Role: "user", Content: question},
		},
	}

	buf := s.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer s.bufPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return "", fmt.Errorf("callLLM: failed to marshal request: %w", err)
	}

	endpoint := s.aiCfg.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, buf)
	if err != nil {
		return "", fmt.Errorf("callLLM: failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.aiCfg.APIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("callLLM: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("callLLM: failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("callLLM: API returned status %d: %s", resp.StatusCode, truncateBytes(body, 512))
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("callLLM: failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("callLLM: API returned no choices")
	}

	answer := chatResp.Choices[0].Message.Content
	log.Debug().
		Int("prompt_tokens", chatResp.Usage.PromptTokens).
		Int("completion_tokens", chatResp.Usage.CompletionTokens).
		Msg("rag_service: LLM response received (non-streaming)")

	return answer, nil
}

// streamFromLLM sends a streaming chat completion request to the Groq API,
// reads the SSE stream from the API, and forwards each content delta to the
// client as an SSE data event. It accumulates and returns the full response text.
func (s *ragService) streamFromLLM(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, question, contextText string) (string, error) {
	payload := chatCompletionRequest{
		Model:       s.aiCfg.Model,
		Temperature: s.aiCfg.Temperature,
		Stream:      true,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt(contextText)},
			{Role: "user", Content: question},
		},
	}

	buf := s.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer s.bufPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return "", fmt.Errorf("streamFromLLM: failed to marshal request: %w", err)
	}

	endpoint := s.aiCfg.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, buf)
	if err != nil {
		return "", fmt.Errorf("streamFromLLM: failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.aiCfg.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("streamFromLLM: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("streamFromLLM: API returned status %d: %s", resp.StatusCode, truncateBytes(body, 512))
	}

	// Read the SSE stream line-by-line.
	scanner := bufio.NewScanner(resp.Body)
	// Increase scanner buffer for potentially large delta payloads.
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	var fullResponse strings.Builder

	for scanner.Scan() {
		// Check for context cancellation (client disconnect).
		select {
		case <-ctx.Done():
			log.Warn().Msg("rag_service: client disconnected during stream")
			return fullResponse.String(), ctx.Err()
		default:
		}

		line := scanner.Text()

		// SSE lines prefixed with "data: " carry the payload.
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// The stream terminates with "data: [DONE]".
		if data == "[DONE]" {
			break
		}

		var chunk chatCompletionStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Warn().
				Err(err).
				Str("raw", truncateStr(data, 200)).
				Msg("rag_service: failed to parse stream chunk — skipping")
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}

		fullResponse.WriteString(delta)

		// Forward the delta to the client as an SSE data event.
		writeSSEData(w, flusher, delta)
	}

	if err := scanner.Err(); err != nil {
		return fullResponse.String(), fmt.Errorf("streamFromLLM: scanner error: %w", err)
	}

	log.Info().
		Int("response_length", fullResponse.Len()).
		Msg("rag_service: LLM stream completed")

	return fullResponse.String(), nil
}

// saveToCacheAsync saves a question + embedding + response triple to the semantic
// cache in a background goroutine. This ensures the HTTP response is never
// delayed by the cache write.
func (s *ragService) saveToCacheAsync(question string, embedding []float32, response string) {
	s.cacheWg.Add(1)
	go func() {
		defer s.cacheWg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("rag_service: panic during async cache save — recovered")
			}
		}()

		// Use a background context with a reasonable timeout since the original
		// HTTP request context may already be cancelled.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.cacheRepo.SaveToCache(ctx, question, embedding, response); err != nil {
			log.Error().Err(err).Msg("rag_service: failed to save response to semantic cache")
		} else {
			log.Info().Str("prompt", truncateStr(question, 80)).Msg("rag_service: saved response to semantic cache")
		}
	}()
}

// ---------------------------------------------------------------------------
// SSE helpers
// ---------------------------------------------------------------------------

// writeSSEData writes a single SSE "data:" frame to the writer and flushes
// immediately so the client receives it without delay. Newlines in the data
// are escaped to "\\n" to prevent breaking the SSE format while ensuring
// the client can reconstruct the original formatting.
func writeSSEData(w http.ResponseWriter, flusher http.Flusher, data string) {
	// SSE spec: each line of data must be prefixed with "data: " and the
	// message is terminated by a blank line.
	escaped := strings.ReplaceAll(data, "\n", "\\n")
	fmt.Fprintf(w, "data: %s\n\n", escaped)
	flusher.Flush()
}

// writeSSEEvent writes a named SSE event with optional data and flushes.
func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event, data string) {
	fmt.Fprintf(w, "event: %s\n", event)
	if data != "" {
		fmt.Fprintf(w, "data: %s\n", data)
	}
	fmt.Fprint(w, "\n")
	flusher.Flush()
}

// ---------------------------------------------------------------------------
// OpenAI-compatible chat completion types
// ---------------------------------------------------------------------------

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float32       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type chatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []chatCompletionChoice `json:"choices"`
	Usage   chatCompletionUsage    `json:"usage"`
}

type chatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Stream chunk types for SSE responses from the LLM.
type chatCompletionStreamChunk struct {
	ID      string                       `json:"id"`
	Object  string                       `json:"object"`
	Created int64                        `json:"created"`
	Model   string                       `json:"model"`
	Choices []chatCompletionStreamChoice `json:"choices"`
}

type chatCompletionStreamChoice struct {
	Index        int              `json:"index"`
	Delta        chatMessageDelta `json:"delta"`
	FinishReason *string          `json:"finish_reason"`
}

type chatMessageDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
