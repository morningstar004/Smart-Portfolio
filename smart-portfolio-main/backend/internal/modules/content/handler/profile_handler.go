package handler

import (
	"net/http"

	"github.com/morningstar004/smart-portfolio/internal/httputil"
	"github.com/morningstar004/smart-portfolio/internal/modules/content/service"
	"github.com/go-chi/chi/v5"
)

type ProfileHandler struct {
	service service.ProfileService
}

func NewProfileHandler(svc service.ProfileService) *ProfileHandler {
	return &ProfileHandler{service: svc}
}

func (h *ProfileHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetProfile)
	r.Get("/education", h.GetEducation)
	r.Get("/experience", h.GetExperience)
	r.Get("/certifications", h.GetCertifications)
	r.Get("/achievements", h.GetAchievements)
	r.Get("/skills", h.GetSkills)

	return r
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.GetProfile(r.Context())
	if err != nil {
		httputil.WriteInternalError(w, err, "ProfileHandler.GetProfile")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, profile)
}

func (h *ProfileHandler) GetEducation(w http.ResponseWriter, r *http.Request) {
	edu, err := h.service.GetEducation(r.Context())
	if err != nil {
		httputil.WriteInternalError(w, err, "ProfileHandler.GetEducation")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, edu)
}

func (h *ProfileHandler) GetExperience(w http.ResponseWriter, r *http.Request) {
	exp, err := h.service.GetExperience(r.Context())
	if err != nil {
		httputil.WriteInternalError(w, err, "ProfileHandler.GetExperience")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, exp)
}

func (h *ProfileHandler) GetCertifications(w http.ResponseWriter, r *http.Request) {
	certs, err := h.service.GetCertifications(r.Context())
	if err != nil {
		httputil.WriteInternalError(w, err, "ProfileHandler.GetCertifications")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, certs)
}

func (h *ProfileHandler) GetAchievements(w http.ResponseWriter, r *http.Request) {
	ach, err := h.service.GetAchievements(r.Context())
	if err != nil {
		httputil.WriteInternalError(w, err, "ProfileHandler.GetAchievements")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, ach)
}

func (h *ProfileHandler) GetSkills(w http.ResponseWriter, r *http.Request) {
	skills, err := h.service.GetSkills(r.Context())
	if err != nil {
		httputil.WriteInternalError(w, err, "ProfileHandler.GetSkills")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, skills)
}
