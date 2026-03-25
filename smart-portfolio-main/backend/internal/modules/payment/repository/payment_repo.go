package repository

import (
	"context"
	"fmt"

	"github.com/ZRishu/smart-portfolio/internal/modules/payment/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// PaymentRepository handles all database operations for the sponsors and
// outbox_events tables. It provides transactional methods that ensure a sponsor
// row and its corresponding outbox event are always committed atomically
// (the transactional outbox pattern).
type PaymentRepository struct {
	pool *pgxpool.Pool
}

// NewPaymentRepository creates a new PaymentRepository backed by the given
// connection pool.
func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

// ProcessSponsorshipTx inserts a new sponsor row and a corresponding outbox
// event inside a single database transaction. If either insert fails the entire
// transaction is rolled back, guaranteeing that we never have a sponsor without
// its outbox event (or vice-versa).
//
// Parameters:
//   - razorpayEventID: the unique event ID from Razorpay's webhook payload,
//     used as the outbox event_id to achieve idempotency (duplicate webhooks
//     are rejected by the UNIQUE constraint on event_id).
//   - paymentID: the Razorpay payment ID (pay_xxxxx).
//   - name: sponsor display name extracted from the payment notes.
//   - email: sponsor email from the payment entity.
//   - amount: payment amount in the major currency unit (e.g. INR, not paise).
//   - currency: three-letter ISO currency code.
//
// Returns the generated sponsor UUID on success.
func (r *PaymentRepository) ProcessSponsorshipTx(
	ctx context.Context,
	razorpayEventID string,
	paymentID string,
	name string,
	email string,
	amount float64,
	currency string,
) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("payment_repo.ProcessSponsorshipTx: failed to begin transaction: %w", err)
	}
	defer func() {
		// Rollback is a no-op if the transaction was already committed.
		_ = tx.Rollback(ctx)
	}()

	// ── Insert sponsor ──────────────────────────────────────────────────
	const insertSponsorSQL = `
		INSERT INTO sponsors (sponsor_name, email, amount, currency, status, razorpay_payment_id)
		VALUES ($1, $2, $3, $4, 'SUCCESS', $5)
		RETURNING id
	`

	var sponsorID uuid.UUID
	err = tx.QueryRow(ctx, insertSponsorSQL,
		name,
		email,
		amount,
		currency,
		paymentID,
	).Scan(&sponsorID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("payment_repo.ProcessSponsorshipTx: sponsor insert failed: %w", err)
	}

	// ── Build the JSON payload for the outbox event ─────────────────────
	// We use fmt.Sprintf here for simplicity. The values are already
	// validated/sanitised by the webhook controller before reaching this
	// method. The payload mirrors what SponsorNotificationListener expects.
	payloadJSON := fmt.Sprintf(
		`{"sponsorName":%q,"amount":%f,"currency":%q,"email":%q}`,
		name, amount, currency, email,
	)

	// ── Insert outbox event ─────────────────────────────────────────────
	// The event_id column has a UNIQUE constraint. If the same Razorpay
	// webhook fires twice, the second insert will fail with a duplicate key
	// error, which the controller catches and treats as a safe no-op.
	outboxID := uuid.New()

	const insertOutboxSQL = `
		INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, is_processed, event_id)
		VALUES ($1, 'SPONSOR', $2, 'SPONSOR_CREATED', $3::jsonb, false, $4)
	`

	_, err = tx.Exec(ctx, insertOutboxSQL,
		outboxID,
		sponsorID.String(),
		payloadJSON,
		razorpayEventID,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("payment_repo.ProcessSponsorshipTx: outbox insert failed: %w", err)
	}

	// ── Commit ──────────────────────────────────────────────────────────
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("payment_repo.ProcessSponsorshipTx: commit failed: %w", err)
	}

	log.Info().
		Str("sponsor_id", sponsorID.String()).
		Str("payment_id", paymentID).
		Str("email", email).
		Float64("amount", amount).
		Str("currency", currency).
		Msg("payment_repo: sponsor + outbox event committed successfully")

	return sponsorID, nil
}

// FetchPendingOutboxEvents retrieves up to `limit` unprocessed outbox events
// ordered by creation time (oldest first). The caller (outbox poller) is
// responsible for publishing each event to the in-process event bus and then
// calling MarkOutboxEventProcessed to flag it as done.
func (r *PaymentRepository) FetchPendingOutboxEvents(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	const query = `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, is_processed, event_id, created_at
		FROM outbox_events
		WHERE is_processed = false
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("payment_repo.FetchPendingOutboxEvents: query failed: %w", err)
	}
	defer rows.Close()

	var events []model.OutboxEvent
	for rows.Next() {
		var e model.OutboxEvent
		if err := rows.Scan(
			&e.ID,
			&e.AggregateType,
			&e.AggregateID,
			&e.EventType,
			&e.Payload,
			&e.IsProcessed,
			&e.EventID,
			&e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("payment_repo.FetchPendingOutboxEvents: scan failed: %w", err)
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("payment_repo.FetchPendingOutboxEvents: rows iteration error: %w", err)
	}

	return events, nil
}

// MarkOutboxEventProcessed sets is_processed = true for the event with the
// given ID. Returns true if a row was updated, false if the event was not found
// (or was already processed).
func (r *PaymentRepository) MarkOutboxEventProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	const query = `
		UPDATE outbox_events
		SET is_processed = true
		WHERE id = $1 AND is_processed = false
	`

	tag, err := r.pool.Exec(ctx, query, eventID)
	if err != nil {
		return false, fmt.Errorf("payment_repo.MarkOutboxEventProcessed: exec failed: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

// FindSponsorByPaymentID looks up a sponsor by their Razorpay payment ID.
// Returns nil and no error if no sponsor is found.
func (r *PaymentRepository) FindSponsorByPaymentID(ctx context.Context, paymentID string) (*model.Sponsor, error) {
	const query = `
		SELECT id, sponsor_name, email, amount, currency, status, razorpay_payment_id, created_at
		FROM sponsors
		WHERE razorpay_payment_id = $1
	`

	var s model.Sponsor
	err := r.pool.QueryRow(ctx, query, paymentID).Scan(
		&s.ID,
		&s.SponsorName,
		&s.Email,
		&s.Amount,
		&s.Currency,
		&s.Status,
		&s.RazorpayPaymentID,
		&s.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("payment_repo.FindSponsorByPaymentID: query failed: %w", err)
	}

	return &s, nil
}

// FindAllSponsors returns every sponsor row ordered by creation date descending.
// Intended for admin dashboard use.
func (r *PaymentRepository) FindAllSponsors(ctx context.Context) ([]model.Sponsor, error) {
	const query = `
		SELECT id, sponsor_name, email, amount, currency, status, razorpay_payment_id, created_at
		FROM sponsors
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("payment_repo.FindAllSponsors: query failed: %w", err)
	}
	defer rows.Close()

	var sponsors []model.Sponsor
	for rows.Next() {
		var s model.Sponsor
		if err := rows.Scan(
			&s.ID,
			&s.SponsorName,
			&s.Email,
			&s.Amount,
			&s.Currency,
			&s.Status,
			&s.RazorpayPaymentID,
			&s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("payment_repo.FindAllSponsors: scan failed: %w", err)
		}
		sponsors = append(sponsors, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("payment_repo.FindAllSponsors: rows iteration error: %w", err)
	}

	return sponsors, nil
}

// FindRecentSponsors returns the most recent successful sponsors.
// Intended for public view.
func (r *PaymentRepository) FindRecentSponsors(ctx context.Context, limit int) ([]model.Sponsor, error) {
	const query = `
		SELECT id, sponsor_name, email, amount, currency, status, razorpay_payment_id, created_at
		FROM sponsors
		WHERE status = 'SUCCESS'
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("payment_repo.FindRecentSponsors: query failed: %w", err)
	}
	defer rows.Close()

	var sponsors []model.Sponsor
	for rows.Next() {
		var s model.Sponsor
		if err := rows.Scan(
			&s.ID,
			&s.SponsorName,
			&s.Email,
			&s.Amount,
			&s.Currency,
			&s.Status,
			&s.RazorpayPaymentID,
			&s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("payment_repo.FindRecentSponsors: scan failed: %w", err)
		}
		sponsors = append(sponsors, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("payment_repo.FindRecentSponsors: rows iteration error: %w", err)
	}

	return sponsors, nil
}

// CountSponsors returns the total number of sponsors and the sum of all
// successful sponsorship amounts. Useful for dashboard statistics.
func (r *PaymentRepository) CountSponsors(ctx context.Context) (count int64, totalAmount float64, err error) {
	const query = `
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(amount), 0) AS total_amount
		FROM sponsors
		WHERE status = 'SUCCESS'
	`

	err = r.pool.QueryRow(ctx, query).Scan(&count, &totalAmount)
	if err != nil {
		return 0, 0, fmt.Errorf("payment_repo.CountSponsors: query failed: %w", err)
	}

	return count, totalAmount, nil
}

// PendingOutboxCount returns the number of outbox events that have not yet
// been processed. Useful for health checks and monitoring.
func (r *PaymentRepository) PendingOutboxCount(ctx context.Context) (int64, error) {
	const query = `SELECT COUNT(*) FROM outbox_events WHERE is_processed = false`

	var count int64
	if err := r.pool.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("payment_repo.PendingOutboxCount: query failed: %w", err)
	}

	return count, nil
}
