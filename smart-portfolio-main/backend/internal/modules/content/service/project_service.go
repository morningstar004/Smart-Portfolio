package service

import (
	"context"
	"fmt"

	"github.com/ZRishu/smart-portfolio/internal/httputil"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/dto"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/model"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/repository"
	"github.com/ZRishu/smart-portfolio/internal/platform/cache"
	"github.com/rs/zerolog/log"
)

const projectsCacheKey = "projects:all"

// ProjectService defines the interface for project business logic.
type ProjectService interface {
	GetAllProjects(ctx context.Context) ([]dto.ProjectResponse, error)
	GetProjectByID(ctx context.Context, id string) (*dto.ProjectResponse, error)
	CreateProject(ctx context.Context, req dto.ProjectRequest) (*dto.ProjectResponse, error)
	UpdateProject(ctx context.Context, id string, req dto.ProjectRequest) (*dto.ProjectResponse, error)
	DeleteProject(ctx context.Context, id string) error
}

// projectService is the concrete implementation of ProjectService.
type projectService struct {
	repo  *repository.ProjectRepository
	cache *cache.Cache
}

// NewProjectService creates a new ProjectService with the given repository and cache.
func NewProjectService(repo *repository.ProjectRepository, c *cache.Cache) ProjectService {
	return &projectService{
		repo:  repo,
		cache: c,
	}
}

// GetAllProjects returns all projects, serving from cache when available.
// On a cache miss, it queries the database and populates the cache for
// subsequent requests.
func (s *projectService) GetAllProjects(ctx context.Context) ([]dto.ProjectResponse, error) {
	// Check cache first.
	if cached, found := s.cache.Get(projectsCacheKey); found {
		if projects, ok := cached.([]dto.ProjectResponse); ok {
			log.Debug().Msg("project_service: cache HIT for all projects")
			return projects, nil
		}
	}

	log.Debug().Msg("project_service: cache MISS — querying database")

	projects, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("project_service.GetAllProjects: %w", err)
	}

	responses := make([]dto.ProjectResponse, 0, len(projects))
	for _, p := range projects {
		responses = append(responses, projectModelToResponse(p))
	}

	// Populate cache.
	s.cache.Set(projectsCacheKey, responses)
	log.Debug().Int("count", len(responses)).Msg("project_service: cached all projects")

	return responses, nil
}

// GetProjectByID returns a single project by ID.
// Returns ErrValidation if the ID is malformed or ErrNotFound if the project
// does not exist.
func (s *projectService) GetProjectByID(ctx context.Context, id string) (*dto.ProjectResponse, error) {
	uid, err := httputil.ParseUUID(id)
	if err != nil {
		return nil, err // already an *httputil.ErrValidation
	}

	project, err := s.repo.FindByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("project_service.GetProjectByID: %w", err)
	}
	if project == nil {
		return nil, httputil.NewErrNotFound("project", id)
	}

	resp := projectModelToResponse(*project)
	return &resp, nil
}

// CreateProject validates the request, persists a new project, invalidates the
// project cache, and returns the created project response.
func (s *projectService) CreateProject(ctx context.Context, req dto.ProjectRequest) (*dto.ProjectResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, &httputil.ErrValidation{Message: err.Error()}
	}

	entity := &model.Project{
		Title:       req.Title,
		Description: req.Description,
		TechStack:   req.TechStack,
		GithubURL:   req.GithubURL,
		LiveURL:     req.LiveURL,
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("project_service.CreateProject: %w", err)
	}

	// Invalidate the projects list cache so the next read picks up the new entry.
	s.cache.Delete(projectsCacheKey)
	log.Info().Str("id", created.ID.String()).Str("title", created.Title).Msg("project_service: project created")

	resp := projectModelToResponse(*created)
	return &resp, nil
}

// UpdateProject validates the request, updates an existing project by ID,
// invalidates the cache, and returns the updated project response.
// Returns ErrValidation if the ID or request is invalid, or ErrNotFound if
// no project matches the given ID.
func (s *projectService) UpdateProject(ctx context.Context, id string, req dto.ProjectRequest) (*dto.ProjectResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, &httputil.ErrValidation{Message: err.Error()}
	}

	uid, err := httputil.ParseUUID(id)
	if err != nil {
		return nil, err // already an *httputil.ErrValidation
	}

	entity := &model.Project{
		ID:          uid,
		Title:       req.Title,
		Description: req.Description,
		TechStack:   req.TechStack,
		GithubURL:   req.GithubURL,
		LiveURL:     req.LiveURL,
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("project_service.UpdateProject: %w", err)
	}
	if updated == nil {
		return nil, httputil.NewErrNotFound("project", id)
	}

	s.cache.Delete(projectsCacheKey)
	log.Info().Str("id", updated.ID.String()).Msg("project_service: project updated")

	resp := projectModelToResponse(*updated)
	return &resp, nil
}

// DeleteProject removes a project by ID and invalidates the cache.
// Returns ErrValidation if the ID is malformed or ErrNotFound if no project
// matches the given ID.
func (s *projectService) DeleteProject(ctx context.Context, id string) error {
	uid, err := httputil.ParseUUID(id)
	if err != nil {
		return err // already an *httputil.ErrValidation
	}

	deleted, err := s.repo.Delete(ctx, uid)
	if err != nil {
		return fmt.Errorf("project_service.DeleteProject: %w", err)
	}
	if !deleted {
		return httputil.NewErrNotFound("project", id)
	}

	s.cache.Delete(projectsCacheKey)
	log.Info().Str("id", id).Msg("project_service: project deleted")

	return nil
}

// projectModelToResponse converts a model.Project into a dto.ProjectResponse.
func projectModelToResponse(p model.Project) dto.ProjectResponse {
	return dto.ProjectResponse{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		TechStack:   p.TechStack,
		GithubURL:   p.GithubURL,
		LiveURL:     p.LiveURL,
		CreatedAt:   p.CreatedAt,
	}
}
