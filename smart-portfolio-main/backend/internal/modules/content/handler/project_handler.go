package handler

import (
	"net/http"

	"github.com/ZRishu/smart-portfolio/internal/httputil"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/dto"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/service"
	"github.com/go-chi/chi/v5"
)

// ProjectHandler handles HTTP requests for the /api/projects endpoints.
type ProjectHandler struct {
	service service.ProjectService
}

// NewProjectHandler creates a new ProjectHandler backed by the given service.
func NewProjectHandler(svc service.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: svc}
}

// Routes returns a chi.Router with all project routes mounted. This keeps
// route definitions co-located with their handler and allows the top-level
// server to mount the sub-router cleanly at any prefix.
//
// Mounted at /api/projects:
//
//	GET  /           → GetAll
//	GET  /{id}       → GetByID
//	POST /           → Create
//	PUT  /{id}       → Update
//	DELETE /{id}     → Delete
func (h *ProjectHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetAll)
	r.Get("/{id}", h.GetByID)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}

// GetAll handles GET /api/projects and returns every project. The service layer
// serves from an in-memory cache when available, so this endpoint is very fast
// for repeat calls.
func (h *ProjectHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	projects, err := h.service.GetAllProjects(r.Context())
	if err != nil {
		httputil.WriteInternalError(w, err, "ProjectHandler.GetAll")
		return
	}

	// Return an empty array instead of null when there are no projects.
	if projects == nil {
		projects = []dto.ProjectResponse{}
	}

	httputil.WriteJSON(w, http.StatusOK, projects)
}

// GetByID handles GET /api/projects/{id} and returns a single project by its
// UUID. Returns 400 if the ID is malformed and 404 if no project matches.
func (h *ProjectHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing project id in URL path")
		return
	}

	project, err := h.service.GetProjectByID(r.Context(), id)
	if httputil.HandleServiceError(w, err, "ProjectHandler.GetByID") {
		return
	}

	httputil.WriteJSON(w, http.StatusOK, project)
}

// Create handles POST /api/projects. It decodes and validates the request body,
// persists the new project, and returns the created resource with a 201 status.
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.ProjectRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	created, err := h.service.CreateProject(r.Context(), req)
	if httputil.HandleServiceError(w, err, "ProjectHandler.Create") {
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, created)
}

// Update handles PUT /api/projects/{id}. It decodes and validates the request
// body, updates the project matching the URL parameter, and returns the updated
// resource. Returns 404 if no project matches the given ID.
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing project id in URL path")
		return
	}

	var req dto.ProjectRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	updated, err := h.service.UpdateProject(r.Context(), id, req)
	if httputil.HandleServiceError(w, err, "ProjectHandler.Update") {
		return
	}

	httputil.WriteJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /api/projects/{id}. It removes the project matching the
// URL parameter and returns 204 No Content on success. Returns 404 if no project
// matches the given ID.
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing project id in URL path")
		return
	}

	err := h.service.DeleteProject(r.Context(), id)
	if httputil.HandleServiceError(w, err, "ProjectHandler.Delete") {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
