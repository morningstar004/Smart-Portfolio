package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ZRishu/smart-portfolio/internal/httputil"
	"github.com/ZRishu/smart-portfolio/internal/modules/ai/dto"
	"github.com/ZRishu/smart-portfolio/internal/modules/ai/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// AIHandler handles HTTP requests for the /api/chat and /api/ingest endpoints.
// It bridges the HTTP layer with the RAG service (for chat) and the ingestion
// service (for PDF upload and text ingestion).
type AIHandler struct {
	ragService       service.RAGService
	ingestionService service.IngestionService
}

// NewAIHandler creates a new AIHandler backed by the given RAG and ingestion
// services. Both services must be non-nil.
func NewAIHandler(ragSvc service.RAGService, ingestionSvc service.IngestionService) *AIHandler {
	return &AIHandler{
		ragService:       ragSvc,
		ingestionService: ingestionSvc,
	}
}

// ChatRoutes returns a chi.Router with all chat-related routes mounted.
//
// Mounted at /api/chat:
//
//	POST /         → AskQuestion (non-streaming JSON response)
//	POST /stream   → StreamQuestion (SSE streaming response)
func (h *AIHandler) ChatRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.AskQuestion)
	r.Post("/stream", h.StreamQuestion)

	return r
}

// IngestRoutes returns a chi.Router with all ingestion-related routes mounted.
//
// Mounted at /api/ingest:
//
//	POST /         → IngestPDF (multipart/form-data file upload)
//	POST /text     → IngestText (raw text ingestion via JSON body)
//	DELETE /       → ClearVectorStore (remove all embeddings)
func (h *AIHandler) IngestRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.IngestPDF)
	r.Post("/text", h.IngestText)
	r.Delete("/", h.ClearVectorStore)

	return r
}

// ---------------------------------------------------------------------------
// Chat handlers
// ---------------------------------------------------------------------------

// AskQuestion handles POST /api/chat. It accepts a JSON body with a "question"
// field, runs the full RAG pipeline (embed → cache check → vector search → LLM),
// and returns the complete answer as a standard JSON response.
func (h *AIHandler) AskQuestion(w http.ResponseWriter, r *http.Request) {
	var req dto.ChatRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	resp, err := h.ragService.AskQuestion(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "validation failed") {
			httputil.WriteValidationError(w, err)
			return
		}
		httputil.WriteInternalError(w, err, "AIHandler.AskQuestion")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// StreamQuestion handles POST /api/chat/stream. It accepts a JSON body with a
// "question" field and streams the answer back to the client using Server-Sent
// Events (SSE).
//
// The response uses Content-Type: text/event-stream. Each token/chunk from the
// LLM is sent as an SSE "data:" frame. When the stream is complete, a final
// "event: done" frame is sent so the client knows to close the connection.
func (h *AIHandler) StreamQuestion(w http.ResponseWriter, r *http.Request) {
	var req dto.ChatRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	// StreamQuestion writes directly to the ResponseWriter using SSE format.
	// Any errors after this point are logged but cannot be sent as JSON because
	// we've already started writing the SSE stream headers.
	if err := h.ragService.StreamQuestion(r.Context(), w, req); err != nil {
		// If the context was cancelled (client disconnected), this is expected.
		if r.Context().Err() != nil {
			log.Debug().Msg("AIHandler.StreamQuestion: client disconnected during stream")
			return
		}

		log.Error().Err(err).Msg("AIHandler.StreamQuestion: streaming failed")
	}
}

// ---------------------------------------------------------------------------
// Ingestion handlers
// ---------------------------------------------------------------------------

// IngestPDF handles POST /api/ingest. It expects a multipart/form-data request
// with a single file field named "file" containing a .pdf document.
func (h *AIHandler) IngestPDF(w http.ResponseWriter, r *http.Request) {
	// Limit the request body to 20 MB to prevent abuse.
	const maxUploadSize = 20 << 20 // 20 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "failed to parse multipart form — file may exceed 20 MB limit")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "missing 'file' field in multipart form: "+err.Error())
		return
	}
	defer file.Close()

	// Validate file name and extension.
	fileName := header.Filename
	if fileName == "" {
		httputil.WriteError(w, http.StatusBadRequest, "uploaded file has no filename")
		return
	}

	if !strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
		httputil.WriteError(w, http.StatusBadRequest, "only .pdf files are accepted — got: "+fileName)
		return
	}

	// Validate the file is not empty.
	if header.Size == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "uploaded PDF file is empty")
		return
	}

	log.Info().
		Str("file", fileName).
		Int64("size_bytes", header.Size).
		Msg("AIHandler.IngestPDF: received PDF upload")

	resp, err := h.ingestionService.IngestPDF(r.Context(), file, fileName)
	if err != nil {
		if strings.Contains(err.Error(), "no extractable text") || strings.Contains(err.Error(), "no pages") {
			httputil.WriteError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		httputil.WriteInternalError(w, err, "AIHandler.IngestPDF")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// IngestText handles POST /api/ingest/text. It accepts a JSON body with raw
// resume text that should be chunked, embedded, and stored in pgvector.
func (h *AIHandler) IngestText(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text       string `json:"text"`
		SourceName string `json:"source_name"`
	}

	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "text field is required and must not be empty")
		return
	}

	sourceName := strings.TrimSpace(req.SourceName)
	if sourceName == "" {
		sourceName = "manual-text-input"
	}

	// Reject text larger than 5 MB to prevent abuse.
	const maxTextSize = 5 << 20
	if len(req.Text) > maxTextSize {
		httputil.WriteError(w, http.StatusBadRequest, "text exceeds maximum allowed size of 5 MB")
		return
	}

	log.Info().
		Str("source", sourceName).
		Int("text_length", len(req.Text)).
		Msg("AIHandler.IngestText: received text for ingestion")

	resp, err := h.ingestionService.IngestText(r.Context(), req.Text, sourceName)
	if err != nil {
		if strings.Contains(err.Error(), "empty") || strings.Contains(err.Error(), "zero chunks") {
			httputil.WriteError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		httputil.WriteInternalError(w, err, "AIHandler.IngestText")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// ClearVectorStore handles DELETE /api/ingest. It removes all document
// embeddings from the resume_embeddings table, effectively wiping the RAG
// knowledge base. This is intended to be called before re-ingesting a new
// version of the resume.
func (h *AIHandler) ClearVectorStore(w http.ResponseWriter, r *http.Request) {
	// Consume and discard the body to prevent connection issues.
	_, _ = io.Copy(io.Discard, r.Body)

	deleted, err := h.ingestionService.ClearAll(r.Context())
	if err != nil {
		httputil.WriteInternalError(w, err, "AIHandler.ClearVectorStore")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Cleared %d documents from vector store", deleted),
		"deleted": deleted,
	})
}
