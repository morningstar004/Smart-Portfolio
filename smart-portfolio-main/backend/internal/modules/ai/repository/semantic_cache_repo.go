package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// SemanticCacheRepository handles database operations for the ai_semantic_cache
// table. It leverages pgvector's cosine distance operator (<=>) to find
// semantically similar prompts that have already been answered, avoiding
// redundant LLM calls.
type SemanticCacheRepository struct {
	pool *pgxpool.Pool
}

// NewSemanticCacheRepository creates a new SemanticCacheRepository backed by the
// given connection pool.
func NewSemanticCacheRepository(pool *pgxpool.Pool) *SemanticCacheRepository {
	return &SemanticCacheRepository{pool: pool}
}

// distanceThreshold is the maximum cosine distance between two prompt embeddings
// for the cache to consider them "the same question". Cosine distance ranges
// from 0 (identical) to 2 (opposite). A threshold of 0.05 is very strict,
// meaning only nearly identical prompts will produce a cache hit.
const distanceThreshold = 0.05

// FindCachedResponse searches the semantic cache for a previously answered prompt
// whose embedding is within distanceThreshold of the provided embedding. If a
// match is found, the cached response string is returned. If no match exists,
// an empty string and false are returned (no error).
//
// The embedding slice must match the configured vector dimensions (768 for
// jina-embeddings-v2-base-en).
func (r *SemanticCacheRepository) FindCachedResponse(ctx context.Context, embedding []float32) (string, bool, error) {
	vectorLiteral := float32SliceToVectorLiteral(embedding)

	const query = `
		SELECT cached_response
		FROM ai_semantic_cache
		WHERE prompt_embedding <=> $1::vector < $2
		ORDER BY prompt_embedding <=> $1::vector ASC
		LIMIT 1
	`

	var cachedResponse string
	err := r.pool.QueryRow(ctx, query, vectorLiteral, distanceThreshold).Scan(&cachedResponse)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("semantic_cache_repo.FindCachedResponse: query failed: %w", err)
	}

	log.Debug().Msg("semantic_cache_repo: cache HIT — returning cached response")
	return cachedResponse, true, nil
}

// SaveToCache stores a new prompt + embedding + response triple in the semantic
// cache so future semantically-similar questions can be answered instantly
// without hitting the LLM.
//
// This method is safe to call from a goroutine — the underlying pgxpool handles
// concurrency.
func (r *SemanticCacheRepository) SaveToCache(ctx context.Context, promptText string, embedding []float32, response string) error {
	vectorLiteral := float32SliceToVectorLiteral(embedding)

	const query = `
		INSERT INTO ai_semantic_cache (prompt_text, prompt_embedding, cached_response)
		VALUES ($1, $2::vector, $3)
	`

	_, err := r.pool.Exec(ctx, query, promptText, vectorLiteral, response)
	if err != nil {
		return fmt.Errorf("semantic_cache_repo.SaveToCache: insert failed: %w", err)
	}

	log.Info().
		Str("prompt", truncate(promptText, 80)).
		Msg("semantic_cache_repo: saved new response to semantic cache")

	return nil
}

// PurgeOlderThan removes cached entries created before the specified interval.
// The interval is a PostgreSQL interval string, e.g. "7 days", "24 hours".
// Returns the number of rows deleted.
func (r *SemanticCacheRepository) PurgeOlderThan(ctx context.Context, interval string) (int64, error) {
	const query = `
		DELETE FROM ai_semantic_cache
		WHERE created_at < now() - $1::interval
	`

	tag, err := r.pool.Exec(ctx, query, interval)
	if err != nil {
		return 0, fmt.Errorf("semantic_cache_repo.PurgeOlderThan: delete failed: %w", err)
	}

	count := tag.RowsAffected()
	if count > 0 {
		log.Info().
			Int64("deleted", count).
			Str("older_than", interval).
			Msg("semantic_cache_repo: purged stale cache entries")
	}

	return count, nil
}

// Count returns the total number of entries currently in the semantic cache.
// Useful for monitoring and health checks.
func (r *SemanticCacheRepository) Count(ctx context.Context) (int64, error) {
	const query = `SELECT COUNT(*) FROM ai_semantic_cache`

	var count int64
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("semantic_cache_repo.Count: query failed: %w", err)
	}

	return count, nil
}

// float32SliceToVectorLiteral converts a Go []float32 slice into a PostgreSQL
// vector literal string of the form "[0.123,0.456,...]". This format is what
// pgvector expects when casting with ::vector.
func float32SliceToVectorLiteral(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.Grow(len(v)*12 + 2) // rough pre-allocation: ~12 chars per float + brackets

	sb.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("%g", f))
	}
	sb.WriteByte(']')

	return sb.String()
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
// Used for log messages to avoid dumping entire prompts into the log output.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
