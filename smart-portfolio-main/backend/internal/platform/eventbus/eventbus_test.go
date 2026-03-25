package eventbus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Subscribe
// ---------------------------------------------------------------------------

func TestSubscribe_SingleHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	bus.Subscribe("TEST_EVENT", func(ctx context.Context, event Event) error {
		return nil
	})

	if bus.HandlerCount() != 1 {
		t.Errorf("expected 1 handler, got %d", bus.HandlerCount())
	}

	if bus.HandlerCountForType("TEST_EVENT") != 1 {
		t.Errorf("expected 1 handler for TEST_EVENT, got %d", bus.HandlerCountForType("TEST_EVENT"))
	}
}

func TestSubscribe_MultipleHandlersSameType(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	noop := func(ctx context.Context, event Event) error { return nil }
	bus.Subscribe("ORDER_CREATED", noop)
	bus.Subscribe("ORDER_CREATED", noop)
	bus.Subscribe("ORDER_CREATED", noop)

	if bus.HandlerCount() != 3 {
		t.Errorf("expected 3 total handlers, got %d", bus.HandlerCount())
	}

	if bus.HandlerCountForType("ORDER_CREATED") != 3 {
		t.Errorf("expected 3 handlers for ORDER_CREATED, got %d", bus.HandlerCountForType("ORDER_CREATED"))
	}
}

func TestSubscribe_MultipleEventTypes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	noop := func(ctx context.Context, event Event) error { return nil }
	bus.Subscribe("EVENT_A", noop)
	bus.Subscribe("EVENT_B", noop)
	bus.Subscribe("EVENT_C", noop)

	if bus.HandlerCount() != 3 {
		t.Errorf("expected 3 total handlers, got %d", bus.HandlerCount())
	}

	if bus.HandlerCountForType("EVENT_A") != 1 {
		t.Errorf("expected 1 handler for EVENT_A, got %d", bus.HandlerCountForType("EVENT_A"))
	}
	if bus.HandlerCountForType("EVENT_B") != 1 {
		t.Errorf("expected 1 handler for EVENT_B, got %d", bus.HandlerCountForType("EVENT_B"))
	}
	if bus.HandlerCountForType("EVENT_C") != 1 {
		t.Errorf("expected 1 handler for EVENT_C, got %d", bus.HandlerCountForType("EVENT_C"))
	}
}

// ---------------------------------------------------------------------------
// HandlerCount
// ---------------------------------------------------------------------------

func TestHandlerCount_Empty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	if bus.HandlerCount() != 0 {
		t.Errorf("expected 0 handlers on new bus, got %d", bus.HandlerCount())
	}
}

func TestHandlerCountForType_Unregistered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	if bus.HandlerCountForType("NONEXISTENT") != 0 {
		t.Errorf("expected 0 handlers for unregistered type, got %d", bus.HandlerCountForType("NONEXISTENT"))
	}
}

// ---------------------------------------------------------------------------
// Publish
// ---------------------------------------------------------------------------

func TestPublish_DeliversToHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	var received atomic.Value

	bus.Subscribe("SPONSOR_CREATED", func(ctx context.Context, event Event) error {
		received.Store(event)
		return nil
	})

	bus.Publish(Event{
		Type:    "SPONSOR_CREATED",
		Payload: `{"name":"Alice","amount":500}`,
	})

	// Wait for the handler goroutine to complete
	bus.Shutdown()

	evt, ok := received.Load().(Event)
	if !ok {
		t.Fatal("expected handler to receive the event")
	}
	if evt.Type != "SPONSOR_CREATED" {
		t.Errorf("expected event type SPONSOR_CREATED, got %q", evt.Type)
	}
	if evt.Payload != `{"name":"Alice","amount":500}` {
		t.Errorf("unexpected payload: %q", evt.Payload)
	}
}

func TestPublish_DeliversToAllHandlers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	var counter atomic.Int32

	for i := 0; i < 5; i++ {
		bus.Subscribe("MULTI_EVENT", func(ctx context.Context, event Event) error {
			counter.Add(1)
			return nil
		})
	}

	bus.Publish(Event{Type: "MULTI_EVENT", Payload: "{}"})

	bus.Shutdown()

	if counter.Load() != 5 {
		t.Errorf("expected all 5 handlers to be called, got %d", counter.Load())
	}
}

func TestPublish_OnlyDeliversToMatchingType(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	var counterA, counterB atomic.Int32

	bus.Subscribe("TYPE_A", func(ctx context.Context, event Event) error {
		counterA.Add(1)
		return nil
	})
	bus.Subscribe("TYPE_B", func(ctx context.Context, event Event) error {
		counterB.Add(1)
		return nil
	})

	bus.Publish(Event{Type: "TYPE_A", Payload: "{}"})

	bus.Shutdown()

	if counterA.Load() != 1 {
		t.Errorf("expected TYPE_A handler to be called once, got %d", counterA.Load())
	}
	if counterB.Load() != 0 {
		t.Errorf("expected TYPE_B handler to NOT be called, got %d", counterB.Load())
	}
}

func TestPublish_NoHandlersRegistered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	// Should not panic when publishing with no handlers
	bus.Publish(Event{Type: "ORPHAN_EVENT", Payload: "{}"})

	bus.Shutdown()
}

func TestPublish_MultipleEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	var counter atomic.Int32

	bus.Subscribe("COUNTER", func(ctx context.Context, event Event) error {
		counter.Add(1)
		return nil
	})

	for i := 0; i < 100; i++ {
		bus.Publish(Event{Type: "COUNTER", Payload: "{}"})
	}

	bus.Shutdown()

	if counter.Load() != 100 {
		t.Errorf("expected 100 handler calls, got %d", counter.Load())
	}
}

func TestPublish_HandlerReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	var counter atomic.Int32

	// Handler that always errors — should not prevent other handlers from running
	bus.Subscribe("ERR_EVENT", func(ctx context.Context, event Event) error {
		counter.Add(1)
		return errors.New("handler failed")
	})

	bus.Subscribe("ERR_EVENT", func(ctx context.Context, event Event) error {
		counter.Add(1)
		return nil
	})

	bus.Publish(Event{Type: "ERR_EVENT", Payload: "{}"})

	bus.Shutdown()

	if counter.Load() != 2 {
		t.Errorf("expected both handlers to be called despite error, got %d", counter.Load())
	}
}

func TestPublish_HandlerPanics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	var safeCalled atomic.Bool

	// Handler that panics — should be recovered and not crash the bus
	bus.Subscribe("PANIC_EVENT", func(ctx context.Context, event Event) error {
		panic("handler exploded")
	})

	// Second handler should still run
	bus.Subscribe("PANIC_EVENT", func(ctx context.Context, event Event) error {
		safeCalled.Store(true)
		return nil
	})

	// Should NOT propagate the panic
	bus.Publish(Event{Type: "PANIC_EVENT", Payload: "{}"})

	bus.Shutdown()

	if !safeCalled.Load() {
		t.Error("expected the safe handler to still be called despite the other handler panicking")
	}
}

func TestPublish_AfterShutdown_IsNoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	bus := New(ctx)

	var counter atomic.Int32

	bus.Subscribe("POST_SHUTDOWN", func(ctx context.Context, event Event) error {
		counter.Add(1)
		return nil
	})

	// Cancel the context (simulating shutdown)
	cancel()

	// Give the bus a moment to detect the cancellation
	time.Sleep(10 * time.Millisecond)

	// Publish after shutdown — should be silently rejected
	bus.Publish(Event{Type: "POST_SHUTDOWN", Payload: "{}"})

	// Give it a moment to ensure no goroutines run
	time.Sleep(50 * time.Millisecond)

	if counter.Load() != 0 {
		t.Errorf("expected 0 calls after shutdown, got %d", counter.Load())
	}
}

// ---------------------------------------------------------------------------
// Shutdown
// ---------------------------------------------------------------------------

func TestShutdown_WaitsForInFlightHandlers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	var completed atomic.Bool

	bus.Subscribe("SLOW_EVENT", func(ctx context.Context, event Event) error {
		time.Sleep(100 * time.Millisecond)
		completed.Store(true)
		return nil
	})

	bus.Publish(Event{Type: "SLOW_EVENT", Payload: "{}"})

	// Shutdown should block until the slow handler finishes
	bus.Shutdown()

	if !completed.Load() {
		t.Error("expected Shutdown to wait for in-flight handler to complete")
	}
}

func TestShutdown_MultipleCallsAreIdempotent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	bus.Subscribe("IDEMPOTENT", func(ctx context.Context, event Event) error {
		return nil
	})

	bus.Publish(Event{Type: "IDEMPOTENT", Payload: "{}"})

	// Multiple shutdown calls should not panic or deadlock
	done := make(chan struct{})
	go func() {
		bus.Shutdown()
		bus.Shutdown()
		bus.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown deadlocked on multiple calls")
	}
}

func TestShutdown_NoHandlers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	// Should not block or panic with no handlers
	done := make(chan struct{})
	go func() {
		bus.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown deadlocked with no handlers")
	}
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestConcurrent_SubscribeAndPublish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	var counter atomic.Int32
	var wg sync.WaitGroup

	// Subscribe handlers concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Subscribe("CONCURRENT", func(ctx context.Context, event Event) error {
				counter.Add(1)
				return nil
			})
		}()
	}
	wg.Wait()

	if bus.HandlerCountForType("CONCURRENT") != 10 {
		t.Fatalf("expected 10 handlers, got %d", bus.HandlerCountForType("CONCURRENT"))
	}

	// Publish events concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(Event{Type: "CONCURRENT", Payload: "{}"})
		}()
	}
	wg.Wait()

	bus.Shutdown()

	// Each of the 50 events should trigger all 10 handlers = 500 calls
	expected := int32(500)
	if counter.Load() != expected {
		t.Errorf("expected %d handler calls, got %d", expected, counter.Load())
	}
}

func TestConcurrent_PublishManyEventTypes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	var counterA, counterB, counterC atomic.Int32

	bus.Subscribe("A", func(ctx context.Context, event Event) error {
		counterA.Add(1)
		return nil
	})
	bus.Subscribe("B", func(ctx context.Context, event Event) error {
		counterB.Add(1)
		return nil
	})
	bus.Subscribe("C", func(ctx context.Context, event Event) error {
		counterC.Add(1)
		return nil
	})

	var wg sync.WaitGroup
	types := []string{"A", "B", "C"}

	for i := 0; i < 300; i++ {
		wg.Add(1)
		eventType := types[i%3]
		go func(et string) {
			defer wg.Done()
			bus.Publish(Event{Type: et, Payload: "{}"})
		}(eventType)
	}
	wg.Wait()
	bus.Shutdown()

	if counterA.Load() != 100 {
		t.Errorf("expected 100 calls for A, got %d", counterA.Load())
	}
	if counterB.Load() != 100 {
		t.Errorf("expected 100 calls for B, got %d", counterB.Load())
	}
	if counterC.Load() != 100 {
		t.Errorf("expected 100 calls for C, got %d", counterC.Load())
	}
}

// ---------------------------------------------------------------------------
// Context propagation
// ---------------------------------------------------------------------------

func TestPublish_HandlerReceivesBusContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)

	var receivedCtx atomic.Value

	bus.Subscribe("CTX_EVENT", func(ctx context.Context, event Event) error {
		receivedCtx.Store(ctx)
		return nil
	})

	bus.Publish(Event{Type: "CTX_EVENT", Payload: "{}"})

	bus.Shutdown()

	handlerCtx, ok := receivedCtx.Load().(context.Context)
	if !ok {
		t.Fatal("expected handler to receive a context")
	}

	// The handler's context should be a child of the bus context
	// After cancel(), the handler context should also be done
	cancel()

	select {
	case <-handlerCtx.Done():
		// OK — context propagated correctly
	case <-time.After(time.Second):
		t.Error("expected handler context to be cancelled after bus context is cancelled")
	}
}

// ---------------------------------------------------------------------------
// Event struct
// ---------------------------------------------------------------------------

func TestEvent_Fields(t *testing.T) {
	evt := Event{
		Type:    "PAYMENT_CAPTURED",
		Payload: `{"amount":500,"currency":"INR"}`,
	}

	if evt.Type != "PAYMENT_CAPTURED" {
		t.Errorf("expected Type 'PAYMENT_CAPTURED', got %q", evt.Type)
	}
	if evt.Payload != `{"amount":500,"currency":"INR"}` {
		t.Errorf("unexpected Payload: %q", evt.Payload)
	}
}

func TestEvent_EmptyPayload(t *testing.T) {
	evt := Event{
		Type:    "EMPTY",
		Payload: "",
	}

	if evt.Type != "EMPTY" {
		t.Errorf("expected Type 'EMPTY', got %q", evt.Type)
	}
	if evt.Payload != "" {
		t.Errorf("expected empty payload, got %q", evt.Payload)
	}
}

// ---------------------------------------------------------------------------
// New
// ---------------------------------------------------------------------------

func TestNew_ReturnsNonNilBus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)
	if bus == nil {
		t.Fatal("expected New to return a non-nil Bus")
	}
}

func TestNew_InitializesEmptyHandlers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)
	if bus.HandlerCount() != 0 {
		t.Errorf("expected 0 handlers on new bus, got %d", bus.HandlerCount())
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkPublish_SingleHandler(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)
	bus.Subscribe("BENCH", func(ctx context.Context, event Event) error {
		return nil
	})

	evt := Event{Type: "BENCH", Payload: `{"test":true}`}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(evt)
	}
	bus.Shutdown()
}

func BenchmarkPublish_TenHandlers(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)
	for i := 0; i < 10; i++ {
		bus.Subscribe("BENCH10", func(ctx context.Context, event Event) error {
			return nil
		})
	}

	evt := Event{Type: "BENCH10", Payload: `{"test":true}`}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(evt)
	}
	bus.Shutdown()
}

func BenchmarkPublish_NoHandlers(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := New(ctx)
	evt := Event{Type: "NOBODY", Payload: `{}`}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(evt)
	}
	bus.Shutdown()
}
