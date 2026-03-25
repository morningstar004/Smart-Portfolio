package handler

import (
	"net/http"

	"github.com/ZRishu/smart-portfolio/internal/httputil"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/dto"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/model"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/service"
	"github.com/go-chi/chi/v5"
)

// ContactHandler handles HTTP requests for the /api/contact endpoints.
type ContactHandler struct {
	service  service.ContactMessageService
	adminKey string
}

// NewContactHandler creates a new ContactHandler backed by the given service.
func NewContactHandler(svc service.ContactMessageService, adminKey string) *ContactHandler {
	return &ContactHandler{service: svc, adminKey: adminKey}
}

// Routes returns a chi.Router with all contact message routes mounted. This
// keeps route definitions co-located with their handler and allows the
// top-level server to mount the sub-router cleanly at any prefix.
//
// Mounted at /api/contact:
//
//	POST /              → Submit (public — contact form submission)
//	GET  /              → GetAll (admin — list all messages)
//	GET  /unread        → GetUnread (admin — list unread messages)
//	PATCH /{id}/read    → MarkAsRead (admin — mark a message as read)
//	DELETE /{id}        → Delete (admin — remove a message)
func (h *ContactHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Submit)
	r.Get("/", h.GetAll)
	r.Get("/unread", h.GetUnread)
	r.Patch("/{id}/read", h.MarkAsRead)
	r.Delete("/{id}", h.Delete)

	return r
}

// Submit handles POST /api/contact. It decodes and validates the request body,
// persists the new contact message, fires an async Discord notification, and
// returns a lightweight confirmation response with 201 status.
func (h *ContactHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req dto.ContactMessageRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	resp, err := h.service.SubmitMessage(r.Context(), req)
	if httputil.HandleServiceError(w, err, "ContactHandler.Submit") {
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, resp)
}

// GetAll handles GET /api/contact. It returns every contact message ordered by
// submission date descending. Intended for admin dashboard use.
func (h *ContactHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeAdmin(w, r) {
		return
	}

	messages, err := h.service.GetAllMessages(r.Context())
	if httputil.HandleServiceError(w, err, "ContactHandler.GetAll") {
		return
	}

	// Return an empty array instead of null when there are no messages.
	if messages == nil {
		messages = []model.ContactMessage{}
	}

	httputil.WriteJSON(w, http.StatusOK, messages)
}

// GetUnread handles GET /api/contact/unread. It returns only unread contact
// messages ordered by submission date descending. Useful for quickly seeing
// new messages that need attention.
func (h *ContactHandler) GetUnread(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeAdmin(w, r) {
		return
	}

	messages, err := h.service.GetUnreadMessages(r.Context())
	if httputil.HandleServiceError(w, err, "ContactHandler.GetUnread") {
		return
	}

	if messages == nil {
		messages = []model.ContactMessage{}
	}

	httputil.WriteJSON(w, http.StatusOK, messages)
}

// MarkAsRead handles PATCH /api/contact/{id}/read. It sets the is_read flag to
// TRUE for the message matching the URL parameter. Returns 400 if the ID is
// malformed and 404 if no message matches or if it was already read.
func (h *ContactHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeAdmin(w, r) {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing message id in URL path")
		return
	}

	err := h.service.MarkAsRead(r.Context(), id)
	if httputil.HandleServiceError(w, err, "ContactHandler.MarkAsRead") {
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "contact message marked as read",
	})
}

// Delete handles DELETE /api/contact/{id}. It removes the contact message
// matching the URL parameter and returns 204 No Content on success. Returns
// 400 if the ID is malformed and 404 if no message matches the given ID.
func (h *ContactHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeAdmin(w, r) {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing message id in URL path")
		return
	}

	err := h.service.DeleteMessage(r.Context(), id)
	if httputil.HandleServiceError(w, err, "ContactHandler.Delete") {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ContactHandler) authorizeAdmin(w http.ResponseWriter, r *http.Request) bool {
	if h.adminKey == "" {
		return true
	}

	key := r.Header.Get("X-Admin-Key")
	if key == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "missing admin credentials")
		return false
	}

	if key != h.adminKey {
		httputil.WriteError(w, http.StatusUnauthorized, "invalid admin credentials")
		return false
	}

	return true
}
