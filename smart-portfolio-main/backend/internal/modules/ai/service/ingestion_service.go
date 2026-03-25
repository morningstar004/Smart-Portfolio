package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/ZRishu/smart-portfolio/internal/modules/ai/dto"
	"github.com/ZRishu/smart-portfolio/internal/modules/ai/repository"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// Public interface
// ---------------------------------------------------------------------------

// IngestionService handles the end-to-end pipeline for converting an uploaded
// PDF resume into vector embeddings stored in pgvector. The pipeline is:
//
//  1. Extract raw text from the PDF.
//  2. Split the text into overlapping chunks suitable for embedding.
//  3. Generate embeddings for every chunk (in parallel batches).
//  4. Store the chunks + embeddings in the resume_embeddings table (atomically).
//
// The service is safe for concurrent use.
type IngestionService interface {
	// IngestPDF reads a PDF from the provided io.Reader, processes it through
	// the full ingestion pipeline, and returns a summary of what was stored.
	// The fileName parameter is used only for logging.
	IngestPDF(ctx context.Context, reader io.Reader, fileName string) (*dto.IngestResponse, error)

	// IngestText takes raw text (e.g. pasted resume content), chunks it,
	// embeds it, and stores it. Useful when the source is not a PDF.
	IngestText(ctx context.Context, text string, sourceName string) (*dto.IngestResponse, error)

	// ClearAll removes every document from the vector store so a fresh
	// re-ingestion can be performed. Returns the number of deleted rows.
	ClearAll(ctx context.Context) (int64, error)
}

// ---------------------------------------------------------------------------
// Configuration knobs
// ---------------------------------------------------------------------------

const (
	// chunkSize is the target number of characters per chunk. Chunks may be
	// slightly larger because we never split in the middle of a word.
	chunkSize = 800

	// chunkOverlap is the number of trailing characters from the previous
	// chunk that are prepended to the next chunk. Overlap ensures that
	// sentences spanning a chunk boundary are still captured.
	chunkOverlap = 200

	// embeddingBatchSize is the maximum number of chunks sent to the
	// embedding API in a single HTTP request. Jina's API supports batched
	// input, and larger batches reduce round-trip overhead.
	embeddingBatchSize = 32

	// maxEmbedConcurrency caps the number of concurrent embedding API calls.
	// This prevents overwhelming the API with too many parallel requests
	// when processing a very large document.
	maxEmbedConcurrency = 4
)

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

type ingestionService struct {
	embeddingSvc EmbeddingService
	vectorRepo   *repository.VectorStoreRepository
}

// NewIngestionService creates a new IngestionService wired to the given
// embedding service (for generating vectors) and vector store repository
// (for persisting them to pgvector).
func NewIngestionService(
	embeddingSvc EmbeddingService,
	vectorRepo *repository.VectorStoreRepository,
) IngestionService {
	log.Info().Msg("ingestion_service: initialized")
	return &ingestionService{
		embeddingSvc: embeddingSvc,
		vectorRepo:   vectorRepo,
	}
}

// ---------------------------------------------------------------------------
// IngestPDF
// ---------------------------------------------------------------------------

func (s *ingestionService) IngestPDF(ctx context.Context, reader io.Reader, fileName string) (*dto.IngestResponse, error) {
	start := time.Now()
	log.Info().Str("file", fileName).Msg("ingestion_service: starting PDF ingestion")

	// Step 1: Extract text from the PDF.
	text, pageCount, err := extractTextFromPDF(reader)
	if err != nil {
		return nil, fmt.Errorf("ingestion_service.IngestPDF: text extraction failed: %w", err)
	}

	text = normaliseWhitespace(text)

	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("ingestion_service.IngestPDF: the PDF appears to contain no extractable text — scanned/image PDFs are not supported")
	}

	log.Info().
		Str("file", fileName).
		Int("pages", pageCount).
		Int("text_length", len(text)).
		Msg("ingestion_service: PDF text extracted")

	// Steps 2-4 are shared with IngestText.
	chunks, err := s.processAndStore(ctx, text, fileName, pageCount)
	if err != nil {
		return nil, err
	}

	elapsed := time.Since(start)
	log.Info().
		Str("file", fileName).
		Int("pages", pageCount).
		Int("chunks", chunks).
		Dur("elapsed", elapsed).
		Msg("ingestion_service: PDF ingestion complete")

	return &dto.IngestResponse{
		Message: fmt.Sprintf("PDF '%s' successfully parsed, chunked, embedded, and saved to pgvector.", fileName),
		Pages:   pageCount,
		Chunks:  chunks,
	}, nil
}

// ---------------------------------------------------------------------------
// IngestText
// ---------------------------------------------------------------------------

func (s *ingestionService) IngestText(ctx context.Context, text string, sourceName string) (*dto.IngestResponse, error) {
	start := time.Now()
	log.Info().Str("source", sourceName).Msg("ingestion_service: starting text ingestion")

	text = normaliseWhitespace(text)

	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("ingestion_service.IngestText: provided text is empty")
	}

	chunks, err := s.processAndStore(ctx, text, sourceName, 0)
	if err != nil {
		return nil, err
	}

	elapsed := time.Since(start)
	log.Info().
		Str("source", sourceName).
		Int("chunks", chunks).
		Dur("elapsed", elapsed).
		Msg("ingestion_service: text ingestion complete")

	return &dto.IngestResponse{
		Message: fmt.Sprintf("Text from '%s' successfully chunked, embedded, and saved to pgvector.", sourceName),
		Pages:   0,
		Chunks:  chunks,
	}, nil
}

// ---------------------------------------------------------------------------
// ClearAll
// ---------------------------------------------------------------------------

func (s *ingestionService) ClearAll(ctx context.Context) (int64, error) {
	deleted, err := s.vectorRepo.DeleteAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("ingestion_service.ClearAll: %w", err)
	}
	log.Info().Int64("deleted", deleted).Msg("ingestion_service: vector store cleared")
	return deleted, nil
}

// ---------------------------------------------------------------------------
// processAndStore: shared pipeline for steps 2-4
// ---------------------------------------------------------------------------

func (s *ingestionService) processAndStore(ctx context.Context, text, sourceName string, pageCount int) (int, error) {
	// Step 2: Split the text into overlapping chunks.
	chunks := chunkText(text, chunkSize, chunkOverlap)
	if len(chunks) == 0 {
		return 0, fmt.Errorf("ingestion_service: text produced zero chunks after splitting")
	}

	log.Info().
		Str("source", sourceName).
		Int("chunks", len(chunks)).
		Int("chunk_size", chunkSize).
		Int("overlap", chunkOverlap).
		Msg("ingestion_service: text split into chunks")

	// Step 3: Generate embeddings for every chunk using concurrent batches.
	embeddings, err := s.embedChunksConcurrently(ctx, chunks)
	if err != nil {
		return 0, fmt.Errorf("ingestion_service: embedding generation failed: %w", err)
	}

	// Step 4: Build EmbeddingDocument slice and store atomically.
	docs := make([]repository.EmbeddingDocument, len(chunks))
	for i, chunk := range chunks {
		metadata := map[string]string{
			"source":       sourceName,
			"chunk_index":  fmt.Sprintf("%d", i),
			"total_chunks": fmt.Sprintf("%d", len(chunks)),
		}
		if pageCount > 0 {
			metadata["total_pages"] = fmt.Sprintf("%d", pageCount)
		}

		docs[i] = repository.EmbeddingDocument{
			Content:   chunk,
			Embedding: embeddings[i],
			Metadata:  metadata,
		}
	}

	if err := s.vectorRepo.AddDocuments(ctx, docs); err != nil {
		return 0, fmt.Errorf("ingestion_service: failed to store documents: %w", err)
	}

	return len(docs), nil
}

// ---------------------------------------------------------------------------
// embedChunksConcurrently batches chunks and embeds them in parallel
// ---------------------------------------------------------------------------

func (s *ingestionService) embedChunksConcurrently(ctx context.Context, chunks []string) ([][]float32, error) {
	total := len(chunks)
	results := make([][]float32, total)

	// Split chunks into batches of embeddingBatchSize.
	type batchWork struct {
		startIdx int
		texts    []string
	}

	var batches []batchWork
	for i := 0; i < total; i += embeddingBatchSize {
		end := i + embeddingBatchSize
		if end > total {
			end = total
		}
		batches = append(batches, batchWork{
			startIdx: i,
			texts:    chunks[i:end],
		})
	}

	log.Debug().
		Int("total_chunks", total).
		Int("batch_count", len(batches)).
		Int("batch_size", embeddingBatchSize).
		Int("concurrency", maxEmbedConcurrency).
		Msg("ingestion_service: starting concurrent embedding")

	// Process batches with bounded concurrency using a semaphore channel.
	sem := make(chan struct{}, maxEmbedConcurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error

	for _, batch := range batches {
		// Check for context cancellation before starting more work.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Check if a previous batch already failed.
		mu.Lock()
		if firstErr != nil {
			mu.Unlock()
			break
		}
		mu.Unlock()

		wg.Add(1)
		sem <- struct{}{} // acquire semaphore slot

		go func(b batchWork) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore slot

			// Recover from panics inside the goroutine.
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("ingestion_service: panic during embedding batch at index %d: %v", b.startIdx, r)
					}
					mu.Unlock()
					log.Error().
						Int("start_index", b.startIdx).
						Interface("panic", r).
						Msg("ingestion_service: recovered from panic in embedding goroutine")
				}
			}()

			batchEmbeddings, err := s.embeddingSvc.EmbedBatch(ctx, b.texts)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("embedding batch starting at index %d failed: %w", b.startIdx, err)
				}
				mu.Unlock()
				return
			}

			if len(batchEmbeddings) != len(b.texts) {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf(
						"embedding batch at index %d: expected %d embeddings, got %d",
						b.startIdx, len(b.texts), len(batchEmbeddings),
					)
				}
				mu.Unlock()
				return
			}

			// Write results into the pre-allocated slice. Each goroutine
			// writes to its own non-overlapping index range, so no lock
			// is needed for the results slice itself.
			for j, emb := range batchEmbeddings {
				results[b.startIdx+j] = emb
			}
		}(batch)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	// Verify that every slot was filled (guards against logic errors).
	for i, r := range results {
		if r == nil {
			return nil, fmt.Errorf("ingestion_service: embedding result at index %d is nil — this should never happen", i)
		}
	}

	log.Info().
		Int("total_chunks", total).
		Msg("ingestion_service: all embeddings generated successfully")

	return results, nil
}

// ---------------------------------------------------------------------------
// PDF text extraction
// ---------------------------------------------------------------------------

// extractTextFromPDF reads a PDF from the io.Reader and returns the
// concatenated text content of all pages along with the total page count.
//
// We read the entire PDF into memory first because the underlying PDF library
// requires random access (io.ReaderAt). For typical resumes (< 5 MB) this is
// perfectly acceptable.
func extractTextFromPDF(reader io.Reader) (string, int, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", 0, fmt.Errorf("extractTextFromPDF: failed to read PDF bytes: %w", err)
	}

	if len(data) == 0 {
		return "", 0, fmt.Errorf("extractTextFromPDF: PDF file is empty")
	}

	// Use the ledongthuc/pdf library which provides a pure-Go PDF text
	// extraction without any CGO dependencies.
	pdfReader, err := openPDFFromBytes(data)
	if err != nil {
		return "", 0, fmt.Errorf("extractTextFromPDF: failed to open PDF: %w", err)
	}

	pageCount := pdfReader.NumPage()
	if pageCount == 0 {
		return "", 0, fmt.Errorf("extractTextFromPDF: PDF contains no pages")
	}

	var sb strings.Builder
	sb.Grow(len(data)) // rough pre-allocation

	for i := 1; i <= pageCount; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		content, err := page.GetPlainText(nil)
		if err != nil {
			log.Warn().
				Int("page", i).
				Err(err).
				Msg("ingestion_service: failed to extract text from page — skipping")
			continue
		}

		if sb.Len() > 0 && content != "" {
			sb.WriteString("\n\n")
		}
		sb.WriteString(content)
	}

	return sb.String(), pageCount, nil
}

func openPDFFromBytes(data []byte) (*pdfBytesReader, error) {
	r := &bytesReaderAt{data: data}
	reader, err := pdfNewReader(r, int64(len(data)))
	if err != nil {
		return nil, err
	}
	return &pdfBytesReader{Reader: reader}, nil
}

// bytesReaderAt implements io.ReaderAt over a byte slice.
type bytesReaderAt struct {
	data []byte
}

func (b *bytesReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n := copy(p, b.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// ---------------------------------------------------------------------------
// Text chunking
// ---------------------------------------------------------------------------

// chunkText splits text into chunks of approximately targetSize characters
// with overlap characters of overlap between consecutive chunks. Splits are
// made at word boundaries to avoid cutting words in half.
//
// If the text is shorter than targetSize, a single chunk is returned.
func chunkText(text string, targetSize, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	if targetSize <= 0 {
		targetSize = 800
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= targetSize {
		overlap = targetSize / 4
	}

	// If the entire text fits in one chunk, return it directly.
	if len(text) <= targetSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + targetSize

		// Clamp to text length.
		if end >= len(text) {
			chunk := strings.TrimSpace(text[start:])
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
			break
		}

		// Walk backwards from `end` to find the nearest word boundary
		// (space, newline, etc.) so we don't cut a word in half.
		splitAt := end
		for splitAt > start && !unicode.IsSpace(rune(text[splitAt])) {
			splitAt--
		}

		// If we couldn't find a word boundary (extremely long word), just
		// hard-split at the target size to avoid an infinite loop.
		if splitAt == start {
			splitAt = end
		}

		chunk := strings.TrimSpace(text[start:splitAt])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		// Advance start by (chunkSize - overlap) so the next chunk overlaps
		// with the tail of the current one.
		advance := splitAt - start - overlap
		if advance <= 0 {
			advance = 1 // always advance by at least 1 to avoid infinite loop
		}
		start += advance
	}

	return chunks
}

// normaliseWhitespace replaces runs of whitespace (tabs, multiple spaces,
// form-feeds, etc.) with single spaces and trims leading/trailing whitespace.
// Newlines are preserved as single newlines to maintain paragraph structure.
func normaliseWhitespace(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))

	prevWasSpace := false
	prevWasNewline := false

	for _, r := range s {
		switch {
		case r == '\n' || r == '\r':
			if !prevWasNewline {
				sb.WriteByte('\n')
				prevWasNewline = true
			}
			prevWasSpace = false
		case unicode.IsSpace(r):
			if !prevWasSpace && !prevWasNewline {
				sb.WriteByte(' ')
				prevWasSpace = true
			}
		default:
			sb.WriteRune(r)
			prevWasSpace = false
			prevWasNewline = false
		}
	}

	return strings.TrimSpace(sb.String())
}
