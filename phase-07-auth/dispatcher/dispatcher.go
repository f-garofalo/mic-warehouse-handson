package dispatcher

import (
	"context"
	"sync"

	"warehouse.local/core/events"
)

// EventDispatcher fans domain events out to subscribers (logs, Hermes, etc.).
// Per ADR-011: aggregates record events; the use case calls Dispatch after Save;
// the dispatcher implementation owns transport and ordering guarantees.
type EventDispatcher interface {
	Dispatch(ctx context.Context, es []events.DomainEvent) error
}

// InMemoryDispatcher captures dispatched events in a goroutine-safe slice.
// FailOnDispatch lets tests simulate transport errors.
type InMemoryDispatcher struct {
	mu             sync.Mutex
	es             []events.DomainEvent
	FailOnDispatch error
}

func NewInMemoryDispatcher() *InMemoryDispatcher {
	return &InMemoryDispatcher{}
}

var _ EventDispatcher = (*InMemoryDispatcher)(nil)

func (d *InMemoryDispatcher) Dispatch(ctx context.Context, es []events.DomainEvent) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.FailOnDispatch != nil {
		return d.FailOnDispatch
	}
	d.es = append(d.es, es...)
	return nil
}

// Events returns a snapshot of dispatched events for assertion in tests.
func (d *InMemoryDispatcher) Events() []events.DomainEvent {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]events.DomainEvent(nil), d.es...)
}

// Reset clears the recorded events. Useful between subtest cases.
func (d *InMemoryDispatcher) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.es = nil
}
