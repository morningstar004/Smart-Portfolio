package service

import (
	"context"
	"fmt"

	"github.com/ZRishu/smart-portfolio/internal/httputil"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/dto"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/model"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/repository"
	notifservice "github.com/ZRishu/smart-portfolio/internal/modules/notification/service"
	"github.com/rs/zerolog/log"
)

// ContactMessageService defines the interface for contact message business logic.
type ContactMessageService interface {
	// SubmitMessage persists a new contact message and fires an async Discord
	// notification. Returns a lightweight response DTO on success.
	SubmitMessage(ctx context.Context, req dto.ContactMessageRequest) (*dto.ContactMessageResponse, error)

	// GetAllMessages returns every contact message ordered by submission date
	// descending. Intended for admin use.
	GetAllMessages(ctx context.Context) ([]model.ContactMessage, error)

	// GetUnreadMessages returns only unread messages ordered by submission date
	// descending.
	GetUnreadMessages(ctx context.Context) ([]model.ContactMessage, error)

	// MarkAsRead sets a message's is_read flag to TRUE. Returns an error
	// wrapping ErrNotFound if no message matches the given ID.
	MarkAsRead(ctx context.Context, id string) error

	// DeleteMessage removes a contact message by ID.
	DeleteMessage(ctx context.Context, id string) error
}

// contactMessageService is the concrete implementation of ContactMessageService.
type contactMessageService struct {
	repo     *repository.ContactRepository
	notifier notifservice.NotificationService
}

// NewContactMessageService creates a new ContactMessageService backed by the
// given repository and notification service.
func NewContactMessageService(
	repo *repository.ContactRepository,
	notifier notifservice.NotificationService,
) ContactMessageService {
	return &contactMessageService{
		repo:     repo,
		notifier: notifier,
	}
}

// SubmitMessage validates the incoming request, persists the contact message to
// PostgreSQL, and dispatches a Discord notification asynchronously (the Discord
// call runs in a goroutine inside the notification service, so this method
// returns as soon as the database write succeeds).
func (s *contactMessageService) SubmitMessage(ctx context.Context, req dto.ContactMessageRequest) (*dto.ContactMessageResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, &httputil.ErrValidation{Message: err.Error()}
	}

	entity := &model.ContactMessage{
		SenderName:  req.SenderName,
		SenderEmail: req.SenderEmail,
		MessageBody: req.MessageBody,
		IsRead:      false,
	}

	saved, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("contact_service.SubmitMessage: %w", err)
	}

	log.Info().
		Str("id", saved.ID.String()).
		Str("sender_email", saved.SenderEmail).
		Msg("contact_service: message submitted successfully")

	// Fire-and-forget Discord notification in a background goroutine. The
	// notification service manages its own goroutine lifecycle and WaitGroup
	// so this never blocks the HTTP response.
	s.notifier.SendContactNotification(ctx, saved.SenderName, saved.SenderEmail, saved.MessageBody)

	resp := &dto.ContactMessageResponse{
		ID:          saved.ID,
		SenderName:  saved.SenderName,
		SubmittedAt: saved.SubmittedAt,
	}

	return resp, nil
}

// GetAllMessages retrieves every contact message from the database.
func (s *contactMessageService) GetAllMessages(ctx context.Context) ([]model.ContactMessage, error) {
	messages, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("contact_service.GetAllMessages: %w", err)
	}

	log.Debug().Int("count", len(messages)).Msg("contact_service: fetched all messages")
	return messages, nil
}

// GetUnreadMessages retrieves only unread contact messages from the database.
func (s *contactMessageService) GetUnreadMessages(ctx context.Context) ([]model.ContactMessage, error) {
	messages, err := s.repo.FindUnread(ctx)
	if err != nil {
		return nil, fmt.Errorf("contact_service.GetUnreadMessages: %w", err)
	}

	log.Debug().Int("count", len(messages)).Msg("contact_service: fetched unread messages")
	return messages, nil
}

// MarkAsRead sets the is_read flag to TRUE for the message matching the given
// ID string. Returns ErrValidation if the ID is malformed or ErrNotFound if
// the message does not exist or was already read.
func (s *contactMessageService) MarkAsRead(ctx context.Context, id string) error {
	uid, err := httputil.ParseUUID(id)
	if err != nil {
		return err // already an *httputil.ErrValidation
	}

	updated, err := s.repo.MarkAsRead(ctx, uid)
	if err != nil {
		return fmt.Errorf("contact_service.MarkAsRead: %w", err)
	}
	if !updated {
		return httputil.NewErrNotFound("contact message", id)
	}

	log.Info().Str("id", id).Msg("contact_service: message marked as read")
	return nil
}

// DeleteMessage removes a contact message by its ID string. Returns
// ErrValidation if the ID is malformed or ErrNotFound if the message does
// not exist.
func (s *contactMessageService) DeleteMessage(ctx context.Context, id string) error {
	uid, err := httputil.ParseUUID(id)
	if err != nil {
		return err // already an *httputil.ErrValidation
	}

	deleted, err := s.repo.Delete(ctx, uid)
	if err != nil {
		return fmt.Errorf("contact_service.DeleteMessage: %w", err)
	}
	if !deleted {
		return httputil.NewErrNotFound("contact message", id)
	}

	log.Info().Str("id", id).Msg("contact_service: message deleted")
	return nil
}
