package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GitHubEmbeddingDocument struct {
	EntityKey    string
	Username     string
	EntityType   string
	GitHubRepoID *int64
	Content      string
	Embedding    []float32
	Metadata     map[string]string
}

type GitHubEmbeddingRepository struct {
	pool       *pgxpool.Pool
	dimensions int
}

func NewGitHubEmbeddingRepository(pool *pgxpool.Pool, dimensions int) *GitHubEmbeddingRepository {
	return &GitHubEmbeddingRepository{
		pool:       pool,
		dimensions: dimensions,
	}
}

func (r *GitHubEmbeddingRepository) UpsertMany(ctx context.Context, docs []GitHubEmbeddingDocument) error {
	if len(docs) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("github_embedding_repo.UpsertMany: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const query = `
		INSERT INTO github_embeddings (entity_key, username, entity_type, github_repo_id, content, embedding, metadata, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::vector, $7::jsonb, now())
		ON CONFLICT (entity_key) DO UPDATE
		SET username = EXCLUDED.username,
		    entity_type = EXCLUDED.entity_type,
		    github_repo_id = EXCLUDED.github_repo_id,
		    content = EXCLUDED.content,
		    embedding = EXCLUDED.embedding,
		    metadata = EXCLUDED.metadata,
		    updated_at = now()
	`

	for i, doc := range docs {
		if len(doc.Embedding) != r.dimensions {
			return fmt.Errorf("github_embedding_repo.UpsertMany: document %d has %d dimensions, expected %d", i, len(doc.Embedding), r.dimensions)
		}

		metadataJSON := "{}"
		if len(doc.Metadata) > 0 {
			metadataJSON = metadataMapToJSON(doc.Metadata)
		}

		if _, err := tx.Exec(ctx, query,
			doc.EntityKey,
			doc.Username,
			doc.EntityType,
			doc.GitHubRepoID,
			doc.Content,
			float32SliceToVectorLiteral(doc.Embedding),
			metadataJSON,
		); err != nil {
			return fmt.Errorf("github_embedding_repo.UpsertMany: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("github_embedding_repo.UpsertMany: commit: %w", err)
	}

	return nil
}

func (r *GitHubEmbeddingRepository) DeleteMissingByUsername(ctx context.Context, username string, keepKeys []string) error {
	if len(keepKeys) == 0 {
		const query = `DELETE FROM github_embeddings WHERE username = $1`
		if _, err := r.pool.Exec(ctx, query, username); err != nil {
			return fmt.Errorf("github_embedding_repo.DeleteMissingByUsername: %w", err)
		}
		return nil
	}

	const query = `
		DELETE FROM github_embeddings
		WHERE username = $1
		  AND NOT (entity_key = ANY($2))
	`
	if _, err := r.pool.Exec(ctx, query, username, keepKeys); err != nil {
		return fmt.Errorf("github_embedding_repo.DeleteMissingByUsername: %w", err)
	}

	return nil
}
