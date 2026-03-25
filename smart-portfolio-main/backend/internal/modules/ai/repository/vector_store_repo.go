package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// EmbeddingDocument represents a single chunk of resume content stored alongside
// its vector embedding in the resume_embeddings table. This is the core data
// structure for the RAG (Retrieval-Augmented Generation) pipeline.
type EmbeddingDocument struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Embedding []float32         `json:"-"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// VectorStoreRepository handles database operations for the resume_embeddings
// table. It provides methods for storing document chunks with their vector
// embeddings and performing similarity searches using pgvector's cosine
// distance operator (<=>).
type VectorStoreRepository struct {
	pool       *pgxpool.Pool
	dimensions int
}

// NewVectorStoreRepository creates a new VectorStoreRepository backed by the
// given connection pool. The dimensions parameter must match the embedding model
// output size (e.g. 768 for jina-embeddings-v2-base-en) and the VECTOR column
// definition in the resume_embeddings table.
func NewVectorStoreRepository(pool *pgxpool.Pool, dimensions int) *VectorStoreRepository {
	return &VectorStoreRepository{
		pool:       pool,
		dimensions: dimensions,
	}
}

// SimilaritySearch finds the topK document chunks whose embeddings are closest
// to the provided query embedding using cosine distance. Results are ordered by
// ascending distance (most similar first).
//
// The returned documents include the content text and metadata but do NOT
// include the raw embedding vectors to keep memory usage low on the application
// side.
func (r *VectorStoreRepository) SimilaritySearch(ctx context.Context, queryEmbedding []float32, topK int) ([]EmbeddingDocument, error) {
	if len(queryEmbedding) != r.dimensions {
		return nil, fmt.Errorf(
			"vector_store_repo.SimilaritySearch: embedding dimensions mismatch: got %d, expected %d",
			len(queryEmbedding), r.dimensions,
		)
	}

	vectorLiteral := float32SliceToVectorLiteral(queryEmbedding)

	const query = `
		SELECT id, content, metadata
		FROM resume_embeddings
		ORDER BY embedding <=> $1::vector ASC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, vectorLiteral, topK)
	if err != nil {
		return nil, fmt.Errorf("vector_store_repo.SimilaritySearch: query failed: %w", err)
	}
	defer rows.Close()

	documents := make([]EmbeddingDocument, 0, topK)
	for rows.Next() {
		var doc EmbeddingDocument
		var metadata *map[string]string

		if err := rows.Scan(&doc.ID, &doc.Content, &metadata); err != nil {
			return nil, fmt.Errorf("vector_store_repo.SimilaritySearch: scan failed: %w", err)
		}

		if metadata != nil {
			doc.Metadata = *metadata
		}

		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vector_store_repo.SimilaritySearch: rows iteration error: %w", err)
	}

	log.Debug().
		Int("results", len(documents)).
		Int("top_k", topK).
		Msg("vector_store_repo: similarity search completed")

	return documents, nil
}

// AddDocuments inserts one or more document chunks with their embeddings into
// the resume_embeddings table. Each document's embedding must have exactly
// r.dimensions elements.
//
// Inserts are performed in a single transaction using a batch approach so that
// either all documents are stored or none are (atomicity). This is important
// during PDF ingestion where we want the entire resume to be indexed or the
// operation to fail cleanly.
func (r *VectorStoreRepository) AddDocuments(ctx context.Context, docs []EmbeddingDocument) error {
	if len(docs) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("vector_store_repo.AddDocuments: failed to begin transaction: %w", err)
	}
	defer func() {
		// Rollback is a no-op if the transaction was already committed.
		_ = tx.Rollback(ctx)
	}()

	const insertSQL = `
		INSERT INTO resume_embeddings (content, embedding, metadata)
		VALUES ($1, $2::vector, $3::jsonb)
	`

	for i, doc := range docs {
		if len(doc.Embedding) != r.dimensions {
			return fmt.Errorf(
				"vector_store_repo.AddDocuments: document %d has %d dimensions, expected %d",
				i, len(doc.Embedding), r.dimensions,
			)
		}

		vectorLiteral := float32SliceToVectorLiteral(doc.Embedding)

		metadataJSON := "{}"
		if len(doc.Metadata) > 0 {
			metadataJSON = metadataMapToJSON(doc.Metadata)
		}

		if _, err := tx.Exec(ctx, insertSQL, doc.Content, vectorLiteral, metadataJSON); err != nil {
			return fmt.Errorf("vector_store_repo.AddDocuments: insert failed for document %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("vector_store_repo.AddDocuments: commit failed: %w", err)
	}

	log.Info().
		Int("count", len(docs)).
		Msg("vector_store_repo: documents added to vector store")

	return nil
}

// DeleteAll removes every row from the resume_embeddings table. This is a
// destructive operation intended for re-ingestion workflows where the entire
// resume is replaced.
func (r *VectorStoreRepository) DeleteAll(ctx context.Context) (int64, error) {
	const query = `DELETE FROM resume_embeddings`

	tag, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("vector_store_repo.DeleteAll: exec failed: %w", err)
	}

	count := tag.RowsAffected()
	log.Info().
		Int64("deleted", count).
		Msg("vector_store_repo: cleared all documents from vector store")

	return count, nil
}

// Count returns the total number of document chunks currently stored in the
// resume_embeddings table. Useful for health checks and monitoring.
func (r *VectorStoreRepository) Count(ctx context.Context) (int64, error) {
	const query = `SELECT COUNT(*) FROM resume_embeddings`

	var count int64
	if err := r.pool.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("vector_store_repo.Count: query failed: %w", err)
	}

	return count, nil
}

// metadataMapToJSON converts a simple string map into a JSON object string
// suitable for insertion into a JSONB column. Keys and values are escaped to
// prevent injection.
func metadataMapToJSON(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}

	var sb strings.Builder
	sb.WriteByte('{')

	first := true
	for k, v := range m {
		if !first {
			sb.WriteByte(',')
		}
		first = false

		sb.WriteByte('"')
		sb.WriteString(escapeJSONString(k))
		sb.WriteString(`":"`)
		sb.WriteString(escapeJSONString(v))
		sb.WriteByte('"')
	}

	sb.WriteByte('}')
	return sb.String()
}

// escapeJSONString escapes special characters in a string so it can be safely
// embedded inside a JSON string value. This handles the minimal set of
// characters that must be escaped per RFC 8259.
func escapeJSONString(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))

	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteRune(r)
		}
	}

	return sb.String()
}
