package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ▸ Task 1 — publish the ArticleCreated Data Product
//
// HermesPublisher is the dispatcher your use cases have fed since Phase 05,
// grown up: instead of collecting events in memory and moving on, it maps
// each domain event to a Data Product payload, validates it against the
// JSON schema in schemas/, and wraps it in a CloudEvents envelope. Records
// are kept in memory for the exercise; in production the same publisher
// would push to the platform via the Hermes SDK (ADR-014).
//
// The pipeline (Dispatch, given) already works end to end for
// InventoryAdjusted: that mapping is the worked example. Your job is the
// ArticleCreated branch, in two places: mapEventToSchema (Task 1a) and
// eventMetadata (Task 1b). The specification is hermes_publisher_test.go.
type HermesPublisher struct {
	registry *SchemaRegistry
	mu       sync.Mutex
	es       []DomainEvent
	ces      []CloudEvent
}

// CloudEvent is the minimal event envelope used by the exercise, aligned
// with Platform ADR0004. The Data Product payload travels in Data and is
// validated separately against schemas/.
type CloudEvent struct {
	SpecVersion     string         `json:"specversion"`
	ID              string         `json:"id"`
	Source          string         `json:"source"`
	Type            string         `json:"type"`
	Subject         string         `json:"subject,omitempty"`
	Time            string         `json:"time"`
	DataContentType string         `json:"datacontenttype"`
	Data            map[string]any `json:"data"`
}

// Compile-time check: HermesPublisher satisfies the dispatcher.EventDispatcher
// contract (duck-typed to avoid an import cycle).
var _ interface {
	Dispatch(context.Context, []DomainEvent) error
} = (*HermesPublisher)(nil)

func NewHermesPublisher(r *SchemaRegistry) *HermesPublisher {
	return &HermesPublisher{registry: r}
}

// Dispatch is GIVEN: map → validate against the Data Product schema → wrap
// in a CloudEvents envelope → keep. Read it once, then work on the two
// functions below.
func (p *HermesPublisher) Dispatch(ctx context.Context, events []DomainEvent) error {
	if p.registry == nil {
		return errors.New("hermes: nil schema registry")
	}
	accepted := make([]CloudEvent, 0, len(events))
	for _, e := range events {
		schemaID, payload, err := mapEventToSchema(e)
		if err != nil {
			return err
		}
		if err := p.registry.Validate(schemaID, payload); err != nil {
			return fmt.Errorf("hermes: %w", err)
		}
		envelope, err := cloudEventFor(e, schemaID, payload)
		if err != nil {
			return err
		}
		if err := validateCloudEvent(envelope); err != nil {
			return fmt.Errorf("cloudevents: %w", err)
		}
		accepted = append(accepted, envelope)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.es = append(p.es, events...)
	p.ces = append(p.ces, accepted...)
	return nil
}

// Published returns a snapshot of the domain events accepted by the publisher.
func (p *HermesPublisher) Published() []DomainEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]DomainEvent(nil), p.es...)
}

// PublishedCloudEvents returns the accepted CloudEvents envelopes: what a
// consumer reads through /debug/hermes/records.
func (p *HermesPublisher) PublishedCloudEvents() []CloudEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]CloudEvent(nil), p.ces...)
}

// mapEventToSchema picks the Data Product (schema id) and builds its payload
// for a known event type. Unknown types return an error: nothing leaves the
// BC without a declared, versioned contract.
func mapEventToSchema(e DomainEvent) (string, map[string]any, error) {
	switch ev := e.(type) {
	case InventoryAdjusted:
		// GIVEN — the worked example: one Data Product mapping, complete.
		// The payload keys come from the schema, the values from the event,
		// timestamps serialized as RFC3339Nano strings.
		return "warehouse.inventory.adjusted.v1", map[string]any{
			"article_id":    ev.ArticleID,
			"location_code": ev.LocationCode,
			"delta":         ev.Delta,
			"new_quantity":  ev.NewQuantity,
			"reason":        ev.Reason,
			"at":            ev.At.Format(time.RFC3339Nano),
		}, nil
	case ArticleCreated:
		// TODO Phase 09 Task 1a — map ArticleCreated to the Data Product
		// "warehouse.article.v1". The contract is
		// schemas/article-entity-record-v1.json: look at its required list
		// and pattern-match the worked example above. Mind the field names:
		// the event says ArticleName, the contract says something else.
		_ = ev
		return "", nil, fmt.Errorf("TODO Phase 09 Task 1a: map ArticleCreated to warehouse.article.v1")
	default:
		return "", nil, fmt.Errorf("hermes: no schema mapping for %T", e)
	}
}

func cloudEventFor(e DomainEvent, schemaID string, payload map[string]any) (CloudEvent, error) {
	eventTime, subject, err := eventMetadata(e)
	if err != nil {
		return CloudEvent{}, err
	}
	return CloudEvent{
		SpecVersion:     "1.0",
		ID:              fmt.Sprintf("%s:%s:%d", schemaID, subject, eventTime.UnixNano()),
		Source:          "warehouse-bc",
		Type:            schemaID,
		Subject:         subject,
		Time:            eventTime.Format(time.RFC3339Nano),
		DataContentType: "application/json",
		Data:            payload,
	}, nil
}

// eventMetadata extracts the CloudEvents time and subject for a known event.
func eventMetadata(e DomainEvent) (time.Time, string, error) {
	switch ev := e.(type) {
	case InventoryAdjusted:
		// GIVEN — the worked example.
		return ev.At, "articles/" + ev.ArticleID + "/inventory/" + ev.LocationCode, nil
	case ArticleCreated:
		// TODO Phase 09 Task 1b — the envelope metadata for ArticleCreated:
		// the event's time, and the subject "articles/<article id>".
		_ = ev
		return time.Time{}, "", fmt.Errorf("TODO Phase 09 Task 1b: CloudEvents metadata for ArticleCreated")
	default:
		return time.Time{}, "", fmt.Errorf("hermes: no CloudEvents metadata for %T", e)
	}
}

func validateCloudEvent(e CloudEvent) error {
	required := map[string]string{
		"specversion":     e.SpecVersion,
		"id":              e.ID,
		"source":          e.Source,
		"type":            e.Type,
		"time":            e.Time,
		"datacontenttype": e.DataContentType,
	}
	for name, value := range required {
		if value == "" {
			return fmt.Errorf("missing required attribute %q", name)
		}
	}
	if e.SpecVersion != "1.0" {
		return fmt.Errorf("unsupported specversion %q", e.SpecVersion)
	}
	if e.DataContentType != "application/json" {
		return fmt.Errorf("unsupported datacontenttype %q", e.DataContentType)
	}
	if e.Data == nil {
		return errors.New("missing data payload")
	}
	return nil
}
