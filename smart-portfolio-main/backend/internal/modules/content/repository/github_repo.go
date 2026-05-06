package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/modules/content/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GitHubProfileRepository struct {
	pool *pgxpool.Pool
}

func NewGitHubProfileRepository(pool *pgxpool.Pool) *GitHubProfileRepository {
	return &GitHubProfileRepository{pool: pool}
}

func (r *GitHubProfileRepository) GetByUsername(ctx context.Context, username string) (*model.GitHubProfile, error) {
	const query = `
		SELECT username, display_name, bio, profile_url, repositories_url, avatar_url,
		       last_synced_at, last_error, rate_limit_remaining, rate_limit_reset_at,
		       created_at, updated_at
		FROM github_profiles
		WHERE username = $1
	`

	var profile model.GitHubProfile
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&profile.Username,
		&profile.DisplayName,
		&profile.Bio,
		&profile.ProfileURL,
		&profile.RepositoriesURL,
		&profile.AvatarURL,
		&profile.LastSyncedAt,
		&profile.LastError,
		&profile.RateLimitRemaining,
		&profile.RateLimitResetAt,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("github_profile_repo.GetByUsername: %w", err)
	}

	return &profile, nil
}

func (r *GitHubProfileRepository) Upsert(ctx context.Context, profile *model.GitHubProfile) error {
	const query = `
		INSERT INTO github_profiles (
			username, display_name, bio, profile_url, repositories_url, avatar_url,
			last_synced_at, last_error, rate_limit_remaining, rate_limit_reset_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
		ON CONFLICT (username) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    bio = EXCLUDED.bio,
		    profile_url = EXCLUDED.profile_url,
		    repositories_url = EXCLUDED.repositories_url,
		    avatar_url = EXCLUDED.avatar_url,
		    last_synced_at = EXCLUDED.last_synced_at,
		    last_error = EXCLUDED.last_error,
		    rate_limit_remaining = EXCLUDED.rate_limit_remaining,
		    rate_limit_reset_at = EXCLUDED.rate_limit_reset_at,
		    updated_at = now()
	`

	if _, err := r.pool.Exec(ctx, query,
		profile.Username,
		profile.DisplayName,
		profile.Bio,
		profile.ProfileURL,
		profile.RepositoriesURL,
		profile.AvatarURL,
		profile.LastSyncedAt,
		profile.LastError,
		profile.RateLimitRemaining,
		profile.RateLimitResetAt,
	); err != nil {
		return fmt.Errorf("github_profile_repo.Upsert: %w", err)
	}

	return nil
}

type GitHubRepositoryRepository struct {
	pool *pgxpool.Pool
}

func NewGitHubRepositoryRepository(pool *pgxpool.Pool) *GitHubRepositoryRepository {
	return &GitHubRepositoryRepository{pool: pool}
}

func (r *GitHubRepositoryRepository) ListByUsername(ctx context.Context, username string, limit int) ([]model.GitHubRepository, error) {
	const query = `
		SELECT id, github_repo_id, username, owner_login, name, full_name, description,
		       readme_summary, tech_stack, github_url, homepage_url, primary_language,
		       stars, forks, watchers, is_pinned, is_archived, github_updated_at,
		       pushed_at, readme_sha, synced_at, created_at
		FROM github_repositories
		WHERE username = $1
		ORDER BY is_pinned DESC, stars DESC, forks DESC, pushed_at DESC NULLS LAST, github_updated_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, username, limit)
	if err != nil {
		return nil, fmt.Errorf("github_repo_repo.ListByUsername: %w", err)
	}
	defer rows.Close()

	var repos []model.GitHubRepository
	for rows.Next() {
		var repo model.GitHubRepository
		if err := rows.Scan(
			&repo.ID,
			&repo.GitHubRepoID,
			&repo.Username,
			&repo.OwnerLogin,
			&repo.Name,
			&repo.FullName,
			&repo.Description,
			&repo.ReadmeSummary,
			&repo.TechStack,
			&repo.GitHubURL,
			&repo.HomepageURL,
			&repo.PrimaryLanguage,
			&repo.Stars,
			&repo.Forks,
			&repo.Watchers,
			&repo.IsPinned,
			&repo.IsArchived,
			&repo.GitHubUpdatedAt,
			&repo.PushedAt,
			&repo.ReadmeSHA,
			&repo.SyncedAt,
			&repo.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("github_repo_repo.ListByUsername: scan failed: %w", err)
		}
		repos = append(repos, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("github_repo_repo.ListByUsername: rows failed: %w", err)
	}

	return repos, nil
}

func (r *GitHubRepositoryRepository) FindByGitHubIDs(ctx context.Context, ids []int64) (map[int64]model.GitHubRepository, error) {
	if len(ids) == 0 {
		return map[int64]model.GitHubRepository{}, nil
	}

	const query = `
		SELECT id, github_repo_id, username, owner_login, name, full_name, description,
		       readme_summary, tech_stack, github_url, homepage_url, primary_language,
		       stars, forks, watchers, is_pinned, is_archived, github_updated_at,
		       pushed_at, readme_sha, synced_at, created_at
		FROM github_repositories
		WHERE github_repo_id = ANY($1)
	`

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("github_repo_repo.FindByGitHubIDs: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]model.GitHubRepository, len(ids))
	for rows.Next() {
		var repo model.GitHubRepository
		if err := rows.Scan(
			&repo.ID,
			&repo.GitHubRepoID,
			&repo.Username,
			&repo.OwnerLogin,
			&repo.Name,
			&repo.FullName,
			&repo.Description,
			&repo.ReadmeSummary,
			&repo.TechStack,
			&repo.GitHubURL,
			&repo.HomepageURL,
			&repo.PrimaryLanguage,
			&repo.Stars,
			&repo.Forks,
			&repo.Watchers,
			&repo.IsPinned,
			&repo.IsArchived,
			&repo.GitHubUpdatedAt,
			&repo.PushedAt,
			&repo.ReadmeSHA,
			&repo.SyncedAt,
			&repo.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("github_repo_repo.FindByGitHubIDs: scan failed: %w", err)
		}
		result[repo.GitHubRepoID] = repo
	}

	return result, nil
}

func (r *GitHubRepositoryRepository) UpsertMany(ctx context.Context, repos []model.GitHubRepository) error {
	if len(repos) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("github_repo_repo.UpsertMany: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const query = `
		INSERT INTO github_repositories (
			github_repo_id, username, owner_login, name, full_name, description,
			readme_summary, tech_stack, github_url, homepage_url, primary_language,
			stars, forks, watchers, is_pinned, is_archived, github_updated_at,
			pushed_at, readme_sha, synced_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, now())
		ON CONFLICT (github_repo_id) DO UPDATE
		SET username = EXCLUDED.username,
		    owner_login = EXCLUDED.owner_login,
		    name = EXCLUDED.name,
		    full_name = EXCLUDED.full_name,
		    description = EXCLUDED.description,
		    readme_summary = EXCLUDED.readme_summary,
		    tech_stack = EXCLUDED.tech_stack,
		    github_url = EXCLUDED.github_url,
		    homepage_url = EXCLUDED.homepage_url,
		    primary_language = EXCLUDED.primary_language,
		    stars = EXCLUDED.stars,
		    forks = EXCLUDED.forks,
		    watchers = EXCLUDED.watchers,
		    is_pinned = EXCLUDED.is_pinned,
		    is_archived = EXCLUDED.is_archived,
		    github_updated_at = EXCLUDED.github_updated_at,
		    pushed_at = EXCLUDED.pushed_at,
		    readme_sha = EXCLUDED.readme_sha,
		    synced_at = now()
	`

	for _, repo := range repos {
		if _, err := tx.Exec(ctx, query,
			repo.GitHubRepoID,
			repo.Username,
			repo.OwnerLogin,
			repo.Name,
			repo.FullName,
			repo.Description,
			repo.ReadmeSummary,
			repo.TechStack,
			repo.GitHubURL,
			repo.HomepageURL,
			repo.PrimaryLanguage,
			repo.Stars,
			repo.Forks,
			repo.Watchers,
			repo.IsPinned,
			repo.IsArchived,
			repo.GitHubUpdatedAt,
			repo.PushedAt,
			repo.ReadmeSHA,
		); err != nil {
			return fmt.Errorf("github_repo_repo.UpsertMany: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("github_repo_repo.UpsertMany: commit: %w", err)
	}

	return nil
}

func (r *GitHubRepositoryRepository) DeleteMissingByUsername(ctx context.Context, username string, keepIDs []int64) error {
	if len(keepIDs) == 0 {
		const query = `DELETE FROM github_repositories WHERE username = $1`
		if _, err := r.pool.Exec(ctx, query, username); err != nil {
			return fmt.Errorf("github_repo_repo.DeleteMissingByUsername: %w", err)
		}
		return nil
	}

	const query = `
		DELETE FROM github_repositories
		WHERE username = $1
		  AND NOT (github_repo_id = ANY($2))
	`
	if _, err := r.pool.Exec(ctx, query, username, keepIDs); err != nil {
		return fmt.Errorf("github_repo_repo.DeleteMissingByUsername: %w", err)
	}

	return nil
}

func (r *GitHubRepositoryRepository) CountByUsername(ctx context.Context, username string) (int64, error) {
	const query = `SELECT COUNT(*) FROM github_repositories WHERE username = $1`
	var count int64
	if err := r.pool.QueryRow(ctx, query, username).Scan(&count); err != nil {
		return 0, fmt.Errorf("github_repo_repo.CountByUsername: %w", err)
	}
	return count, nil
}

func PtrTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
