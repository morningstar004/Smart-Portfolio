package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ZRishu/smart-portfolio/internal/modules/content/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectRepository handles all database operations for the projects table.
type ProjectRepository struct {
	pool *pgxpool.Pool
}

// NewProjectRepository creates a new ProjectRepository backed by the given
// connection pool.
func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

// FindAll returns every project ordered by creation date descending.
func (r *ProjectRepository) FindAll(ctx context.Context) ([]model.Project, error) {
	const query = `
		SELECT id, title, description, tech_stack, github_url, live_url, created_at
		FROM projects
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("project_repo.FindAll: query failed: %w", err)
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Description,
			&p.TechStack,
			&p.GithubURL,
			&p.LiveURL,
			&p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("project_repo.FindAll: scan failed: %w", err)
		}
		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("project_repo.FindAll: rows iteration error: %w", err)
	}

	return projects, nil
}

// FindByID returns a single project by its UUID. Returns nil and no error if
// the project does not exist.
func (r *ProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	const query = `
		SELECT id, title, description, tech_stack, github_url, live_url, created_at
		FROM projects
		WHERE id = $1
	`

	var p model.Project
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.Title,
		&p.Description,
		&p.TechStack,
		&p.GithubURL,
		&p.LiveURL,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("project_repo.FindByID: query failed: %w", err)
	}

	return &p, nil
}

// Create inserts a new project and returns the fully populated model with the
// database-generated id and created_at timestamp.
func (r *ProjectRepository) Create(ctx context.Context, p *model.Project) (*model.Project, error) {
	const query = `
		INSERT INTO projects (title, description, tech_stack, github_url, live_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, tech_stack, github_url, live_url, created_at
	`

	var created model.Project
	err := r.pool.QueryRow(ctx, query,
		p.Title,
		p.Description,
		p.TechStack,
		p.GithubURL,
		p.LiveURL,
	).Scan(
		&created.ID,
		&created.Title,
		&created.Description,
		&created.TechStack,
		&created.GithubURL,
		&created.LiveURL,
		&created.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("project_repo.Create: insert failed: %w", err)
	}

	return &created, nil
}

// Update modifies an existing project identified by its ID. Only the title,
// description, tech_stack, github_url, and live_url columns are updated.
// Returns the updated model or nil if the project was not found.
func (r *ProjectRepository) Update(ctx context.Context, p *model.Project) (*model.Project, error) {
	const query = `
		UPDATE projects
		SET title       = $1,
		    description = $2,
		    tech_stack  = $3,
		    github_url  = $4,
		    live_url    = $5
		WHERE id = $6
		RETURNING id, title, description, tech_stack, github_url, live_url, created_at
	`

	var updated model.Project
	err := r.pool.QueryRow(ctx, query,
		p.Title,
		p.Description,
		p.TechStack,
		p.GithubURL,
		p.LiveURL,
		p.ID,
	).Scan(
		&updated.ID,
		&updated.Title,
		&updated.Description,
		&updated.TechStack,
		&updated.GithubURL,
		&updated.LiveURL,
		&updated.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("project_repo.Update: query failed: %w", err)
	}

	return &updated, nil
}

// Delete removes a project by its UUID. Returns true if a row was deleted,
// false if the project was not found.
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	const query = `DELETE FROM projects WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return false, fmt.Errorf("project_repo.Delete: exec failed: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}
