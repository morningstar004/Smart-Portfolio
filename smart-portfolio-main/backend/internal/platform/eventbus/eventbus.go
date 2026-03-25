package eventbus

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
)

// Event represents a domain event published on the bus.
type Event struct {
	// Type is a string identifier for the event, e.g. "SPONSOR_CREATED".
	Type string
	// Payload is the raw JSON payload associated with the event.
	Payload string
}

// HandlerFunc is a function that processes an event. Handlers receive a context
// so they can respect cancellation/timeouts from the application lifecycle.
type HandlerFunc func(ctx context.Context, event Event) error

// Bus is an in-process, goroutine-based event bus. It supports multiple
// subscribers per event type and dispatches events asynchronously.
//
// Design notes:
//   - Handlers are invoked in separate goroutines for concurrency.
//   - The bus tracks active goroutines via a sync.WaitGroup so Shutdown can
//     drain in-flight handlers before the process exits.
//   - A buffered channel is used per-publish to avoid blocking the publisher.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]HandlerFunc
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// New creates a new event bus bound to the provided parent context.
// When the parent context is cancelled (e.g. during graceful shutdown), all
// in-flight handler contexts will also be cancelled.
func New(parent context.Context) *Bus {
	ctx, cancel := context.WithCancel(parent)

	log.Info().Msg("eventbus: initialized in-process event bus")

	return &Bus{
		handlers: make(map[string][]HandlerFunc),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Subscribe registers a handler for the given event type. Multiple handlers
// can be registered for the same event type; all of them will be called when
// an event of that type is published.
//
// Subscribe is safe to call from multiple goroutines, but in practice you
// should register all handlers during application startup before publishing
// any events.
func (b *Bus) Subscribe(eventType string, handler HandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	log.Info().
		Str("event_type", eventType).
		Int("total_handlers", len(b.handlers[eventType])).
		Msg("eventbus: handler subscribed")
}

// Publish dispatches an event to all registered handlers for that event type.
// Each handler is invoked in its own goroutine so the publisher is never blocked.
//
// If the bus has been shut down (context cancelled), Publish is a no-op and
// returns immediately.
func (b *Bus) Publish(event Event) {
	// Check if the bus is still alive.
	select {
	case <-b.ctx.Done():
		log.Warn().
			Str("event_type", event.Type).
			Msg("eventbus: publish rejected — bus is shut down")
		return
	default:
	}

	b.mu.RLock()
	handlers, ok := b.handlers[event.Type]
	b.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		log.Debug().
			Str("event_type", event.Type).
			Msg("eventbus: no handlers registered for event type")
		return
	}

	log.Debug().
		Str("event_type", event.Type).
		Int("handler_count", len(handlers)).
		Msg("eventbus: dispatching event")

	for i, handler := range handlers {
		b.wg.Add(1)

		// Capture loop variables for the goroutine closure.
		handlerIdx := i
		h := handler
		evt := event

		go func() {
			defer b.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Error().
						Str("event_type", evt.Type).
						Int("handler_index", handlerIdx).
						Interface("panic", r).
						Msg("eventbus: handler panicked — recovered")
				}
			}()

			if err := h(b.ctx, evt); err != nil {
				log.Error().
					Err(err).
					Str("event_type", evt.Type).
					Int("handler_index", handlerIdx).
					Msg("eventbus: handler returned error")
			}
		}()
	}
}

// Shutdown cancels the bus context and waits for all in-flight handlers to
// finish. Call this during graceful application shutdown to ensure no event
// processing is lost.
func (b *Bus) Shutdown() {
	log.Info().Msg("eventbus: shutting down — waiting for in-flight handlers")
	b.cancel()
	b.wg.Wait()
	log.Info().Msg("eventbus: all handlers drained — shutdown complete")
}

// HandlerCount returns the total number of registered handlers across all
// event types. Useful for health checks and diagnostics.
func (b *Bus) HandlerCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	total := 0
	for _, hs := range b.handlers {
		total += len(hs)
	}
	return total
}

// HandlerCountForType returns the number of handlers registered for a
// specific event type.
func (b *Bus) HandlerCountForType(eventType string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.handlers[eventType])
}
