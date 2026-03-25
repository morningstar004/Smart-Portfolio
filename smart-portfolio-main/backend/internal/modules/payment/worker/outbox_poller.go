package worker

import (
	"context"
	"sync"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/modules/payment/repository"
	"github.com/ZRishu/smart-portfolio/internal/platform/eventbus"
	"github.com/rs/zerolog/log"
)

// OutboxPoller is a background worker that periodically scans the outbox_events
// table for unprocessed events, publishes them onto the in-process event bus,
// and marks them as processed. This is the "relay" component of the
// transactional outbox pattern.
//
// Design notes:
//   - The poller runs in its own goroutine started by Start().
//   - It uses a time.Ticker so the interval is measured from the END of each
//     poll cycle, not the start (equivalent to Java's fixedDelay semantics).
//   - Graceful shutdown is handled via context cancellation. Call Stop() or
//     cancel the context passed to Start() to terminate the worker. The worker
//     will finish its current poll cycle before exiting.
//   - Each event is published and marked individually so that a failure on one
//     event does not block the rest of the batch.
type OutboxPoller struct {
	repo      *repository.PaymentRepository
	bus       *eventbus.Bus
	interval  time.Duration
	batchSize int

	stopOnce sync.Once
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewOutboxPoller creates a new OutboxPoller configured with the given poll
// interval and batch size. The poller is not started until Start() is called.
//
// Parameters:
//   - repo: the payment repository used to fetch and update outbox events.
//   - bus: the in-process event bus where events are published.
//   - interval: how long to wait after each poll cycle before polling again.
//   - batchSize: the maximum number of events to fetch per poll cycle.
func NewOutboxPoller(
	repo *repository.PaymentRepository,
	bus *eventbus.Bus,
	interval time.Duration,
	batchSize int,
) *OutboxPoller {
	if batchSize <= 0 {
		batchSize = 50
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}

	return &OutboxPoller{
		repo:      repo,
		bus:       bus,
		interval:  interval,
		batchSize: batchSize,
		done:      make(chan struct{}),
	}
}

// Start launches the polling loop in a background goroutine. It blocks until
// the provided context is cancelled or Stop() is called. The method itself
// returns immediately — the actual work runs in the spawned goroutine.
//
// It is safe to call Start only once. Calling it multiple times will spawn
// multiple goroutines, which is not recommended.
func (p *OutboxPoller) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	p.cancel = cancel

	go p.run(ctx)

	log.Info().
		Dur("interval", p.interval).
		Int("batch_size", p.batchSize).
		Msg("outbox_poller: started background worker")
}

// Stop signals the polling loop to terminate and blocks until it has fully
// stopped. It is safe to call Stop multiple times — subsequent calls are
// no-ops.
func (p *OutboxPoller) Stop() {
	p.stopOnce.Do(func() {
		log.Info().Msg("outbox_poller: stop requested — waiting for current cycle to finish")
		if p.cancel != nil {
			p.cancel()
		}
		<-p.done
		log.Info().Msg("outbox_poller: stopped")
	})
}

// run is the main polling loop. It executes one poll cycle, then sleeps for
// the configured interval before repeating. It exits when the context is
// cancelled.
func (p *OutboxPoller) run(ctx context.Context) {
	defer close(p.done)

	// Execute one poll immediately on startup so we don't wait an entire
	// interval before processing events that accumulated while the server
	// was down.
	p.pollOnce(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("outbox_poller: context cancelled — exiting poll loop")
			return
		case <-ticker.C:
			p.pollOnce(ctx)
		}
	}
}

// pollOnce fetches a batch of pending outbox events from the database,
// publishes each one to the event bus, and marks it as processed. Events
// that fail to publish are left unprocessed so they will be retried on the
// next cycle.
func (p *OutboxPoller) pollOnce(ctx context.Context) {
	// Use a short timeout for the database query itself so a slow query
	// doesn't block the entire poll cycle indefinitely.
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	events, err := p.repo.FetchPendingOutboxEvents(queryCtx, p.batchSize)
	if err != nil {
		// If the context was cancelled (shutdown), this is expected.
		if ctx.Err() != nil {
			return
		}
		log.Error().Err(err).Msg("outbox_poller: failed to fetch pending events")
		return
	}

	if len(events) == 0 {
		return
	}

	log.Info().
		Int("count", len(events)).
		Msg("outbox_poller: found pending events — processing")

	for _, evt := range events {
		// Check for context cancellation between events so we can exit
		// promptly during shutdown.
		select {
		case <-ctx.Done():
			log.Info().Msg("outbox_poller: context cancelled mid-batch — stopping event processing")
			return
		default:
		}

		// Publish the event to the in-process event bus. The bus dispatches
		// handlers asynchronously in their own goroutines, so this call
		// returns immediately.
		p.bus.Publish(eventbus.Event{
			Type:    evt.EventType,
			Payload: evt.Payload,
		})

		// Mark the event as processed so it won't be picked up again.
		markCtx, markCancel := context.WithTimeout(ctx, 5*time.Second)
		updated, err := p.repo.MarkOutboxEventProcessed(markCtx, evt.ID)
		markCancel()

		if err != nil {
			log.Error().
				Err(err).
				Str("event_id", evt.ID.String()).
				Str("event_type", evt.EventType).
				Msg("outbox_poller: failed to mark event as processed — it will be retried")
			continue
		}

		if updated {
			log.Debug().
				Str("event_id", evt.ID.String()).
				Str("event_type", evt.EventType).
				Msg("outbox_poller: event published and marked as processed")
		} else {
			// This can happen if another instance already processed the
			// event between our SELECT and UPDATE (unlikely in a single-
			// instance deployment, but safe to handle).
			log.Debug().
				Str("event_id", evt.ID.String()).
				Msg("outbox_poller: event was already processed by another worker")
		}
	}

	log.Info().
		Int("count", len(events)).
		Msg("outbox_poller: batch processing complete")
}
