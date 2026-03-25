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

// ContactRepository handles all database operations for the contact_messages table.
type ContactRepository struct {
	pool *pgxpool.Pool
}

// NewContactRepository creates a new ContactRepository backed by the given
// connection pool.
func NewContactRepository(pool *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{pool: pool}
}

// Create inserts a new contact message and returns the fully populated model
// with the database-generated id and submitted_at timestamp.
func (r *ContactRepository) Create(ctx context.Context, msg *model.ContactMessage) (*model.ContactMessage, error) {
	const query = `
		INSERT INTO contact_messages (sender_name, sender_email, message_body)
		VALUES ($1, $2, $3)
		RETURNING id, sender_name, sender_email, message_body, is_read, submitted_at
	`

	var created model.ContactMessage
	err := r.pool.QueryRow(ctx, query,
		msg.SenderName,
		msg.SenderEmail,
		msg.MessageBody,
	).Scan(
		&created.ID,
		&created.SenderName,
		&created.SenderEmail,
		&created.MessageBody,
		&created.IsRead,
		&created.SubmittedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("contact_repo.Create: insert failed: %w", err)
	}

	return &created, nil
}

// FindAll returns every contact message ordered by submission date descending.
func (r *ContactRepository) FindAll(ctx context.Context) ([]model.ContactMessage, error) {
	const query = `
		SELECT id, sender_name, sender_email, message_body, is_read, submitted_at
		FROM contact_messages
		ORDER BY submitted_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("contact_repo.FindAll: query failed: %w", err)
	}
	defer rows.Close()

	var messages []model.ContactMessage
	for rows.Next() {
		var m model.ContactMessage
		if err := rows.Scan(
			&m.ID,
			&m.SenderName,
			&m.SenderEmail,
			&m.MessageBody,
			&m.IsRead,
			&m.SubmittedAt,
		); err != nil {
			return nil, fmt.Errorf("contact_repo.FindAll: scan failed: %w", err)
		}
		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("contact_repo.FindAll: rows iteration error: %w", err)
	}

	return messages, nil
}

// FindUnread returns all unread contact messages ordered by submission date
// descending. This is the primary query used by the admin dashboard to see
// new messages that need attention.
func (r *ContactRepository) FindUnread(ctx context.Context) ([]model.ContactMessage, error) {
	const query = `
		SELECT id, sender_name, sender_email, message_body, is_read, submitted_at
		FROM contact_messages
		WHERE is_read = FALSE
		ORDER BY submitted_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("contact_repo.FindUnread: query failed: %w", err)
	}
	defer rows.Close()

	var messages []model.ContactMessage
	for rows.Next() {
		var m model.ContactMessage
		if err := rows.Scan(
			&m.ID,
			&m.SenderName,
			&m.SenderEmail,
			&m.MessageBody,
			&m.IsRead,
			&m.SubmittedAt,
		); err != nil {
			return nil, fmt.Errorf("contact_repo.FindUnread: scan failed: %w", err)
		}
		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("contact_repo.FindUnread: rows iteration error: %w", err)
	}

	return messages, nil
}

// FindByID returns a single contact message by its UUID. Returns nil and no
// error if the message does not exist.
func (r *ContactRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.ContactMessage, error) {
	const query = `
		SELECT id, sender_name, sender_email, message_body, is_read, submitted_at
		FROM contact_messages
		WHERE id = $1
	`

	var m model.ContactMessage
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.SenderName,
		&m.SenderEmail,
		&m.MessageBody,
		&m.IsRead,
		&m.SubmittedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("contact_repo.FindByID: query failed: %w", err)
	}

	return &m, nil
}

// MarkAsRead sets is_read = TRUE for the message with the given UUID.
// Returns true if the row was updated, false if the message was not found.
func (r *ContactRepository) MarkAsRead(ctx context.Context, id uuid.UUID) (bool, error) {
	const query = `
		UPDATE contact_messages
		SET is_read = TRUE
		WHERE id = $1 AND is_read = FALSE
	`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return false, fmt.Errorf("contact_repo.MarkAsRead: exec failed: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

// Delete removes a contact message by its UUID. Returns true if a row was
// deleted, false if the message was not found.
func (r *ContactRepository) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	const query = `DELETE FROM contact_messages WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return false, fmt.Errorf("contact_repo.Delete: exec failed: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

// Count returns the total number of contact messages and the number of unread
// messages. This is useful for dashboard summary statistics.
func (r *ContactRepository) Count(ctx context.Context) (total int64, unread int64, err error) {
	const query = `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE is_read = FALSE) AS unread
		FROM contact_messages
	`

	err = r.pool.QueryRow(ctx, query).Scan(&total, &unread)
	if err != nil {
		return 0, 0, fmt.Errorf("contact_repo.Count: query failed: %w", err)
	}

	return total, unread, nil
}
