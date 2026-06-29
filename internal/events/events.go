// Package events is the internal domain-event system: a domain event is published
// once, inside the same transaction that commits the state change (transactional
// outbox), and fanned out to independent subscribers over watermill. See
// docs/design/domain-events.md.
//
// Two categories of events, never conflated:
//   - OUR events are a closed, typed union (ContactCreated, EmailEngagement, …).
//     Each Go type owns its projection via Project().
//   - A CUSTOMER's event (page_view, added_to_cart) is the user's domain — opaque
//     to us, stored as-is. Ingesting one is itself OUR typed event, CollectedEvent,
//     whose body carries the user's action + properties verbatim.
//
// On the wire the envelope carries the typed event as Data (json); subscribers
// decode it back into the concrete type via the name registry and call Project().
package events

import (
	"encoding/json"
	"fmt"
	"time"
)

// TopicDomainEvents is the single watermill topic all domain events flow through.
// watermill-sql maps it to "watermill_domain_events", which is the outbox table.
const TopicDomainEvents = "domain_events"

// Bus event names — the stable type discriminators (registry keys). These are
// OURS and finite. A customer's action name is NOT one of these; it lives inside
// CollectedEvent.Action.
const (
	NameContactCreated    = "contact.created"
	NameEmailOpened       = "email.opened"
	NameEmailClicked      = "email.clicked"
	NameEmailUnsubscribed = "email.unsubscribed"
	NameCollected         = "event.collected"
)

// Envelope is the generic bus message. It carries transport metadata plus the
// typed event as Data; subscribers decode Data via the registry.
type Envelope struct {
	ID          string          `json:"id"`          // ULID; idempotency key for consumers
	Name        string          `json:"name"`        // bus event type (registry key)
	Version     int             `json:"version"`     // schema version of Data
	WorkspaceID int64           `json:"workspaceId"` // every event is workspace-scoped
	OccurredAt  time.Time       `json:"occurredAt"`
	Data        json.RawMessage `json:"data"` // the typed event, marshaled
}

// Projection is the durable Event-row an event maps to. Each event type computes
// its own via Project(), so the projection logic lives with the type rather than
// being flattened by producers.
type Projection struct {
	Subject    string         // → events.subject_id (email/phone/visitor identity)
	Action     string         // → events.action; also the automation trigger key
	Email      string         // "" ⇒ unset
	Phone      string         // "" ⇒ unset
	Prospect   *bool          // nil ⇒ unset
	Properties map[string]any // → events.properties (opaque for customer events)
	OccurredAt time.Time      // zero ⇒ publish time
	ContactID  int64          // automations enroll target; 0 ⇒ none
}

// DomainEvent is a typed, producer-side domain event. Implementations are
// registered in registry and must use pointer receivers so Decode can unmarshal.
type DomainEvent interface {
	EventName() string   // bus type / registry key
	EventVersion() int   // schema version
	Workspace() int64    // owning workspace
	Project() Projection // how this event maps to the Event row + trigger
}

// registry maps a bus event name to a constructor for its concrete type, so a
// serialized envelope can be decoded back into the right Go type. All variants
// are ours and finite — there is no catch-all.
var registry = map[string]func() DomainEvent{
	NameContactCreated:    func() DomainEvent { return &ContactCreated{} },
	NameEmailOpened:       func() DomainEvent { return &EmailEngagement{} },
	NameEmailClicked:      func() DomainEvent { return &EmailEngagement{} },
	NameEmailUnsubscribed: func() DomainEvent { return &EmailEngagement{} },
	NameCollected:         func() DomainEvent { return &CollectedEvent{} },
}

// Decode reconstructs the typed event from an envelope.
func Decode(env Envelope) (DomainEvent, error) {
	newEvent, ok := registry[env.Name]
	if !ok {
		return nil, fmt.Errorf("events: unknown event %q", env.Name)
	}
	ev := newEvent()
	if err := json.Unmarshal(env.Data, ev); err != nil {
		return nil, fmt.Errorf("events: decode %q: %w", env.Name, err)
	}
	return ev, nil
}

// ContactCreated is emitted when a contact is created in a workspace.
type ContactCreated struct {
	WorkspaceID int64  `json:"workspaceId"`
	ContactID   int64  `json:"contactId"`
	Email       string `json:"email"`
}

func (*ContactCreated) EventName() string  { return NameContactCreated }
func (*ContactCreated) EventVersion() int  { return 1 }
func (e *ContactCreated) Workspace() int64 { return e.WorkspaceID }
func (e *ContactCreated) Project() Projection {
	return Projection{Subject: e.Email, Action: NameContactCreated, Email: e.Email, ContactID: e.ContactID}
}

// EmailEngagement is emitted when a recipient opens, clicks, or unsubscribes from
// a broadcast email. Action is the specific name; URL is set for clicks only.
type EmailEngagement struct {
	Action      string `json:"action"` // email.opened|email.clicked|email.unsubscribed
	WorkspaceID int64  `json:"workspaceId"`
	ContactID   int64  `json:"contactId"`
	Email       string `json:"email"`
	BroadcastID int64  `json:"broadcastId"`
	URL         string `json:"url,omitempty"`
}

func (e *EmailEngagement) EventName() string { return e.Action }
func (*EmailEngagement) EventVersion() int   { return 1 }
func (e *EmailEngagement) Workspace() int64  { return e.WorkspaceID }
func (e *EmailEngagement) Project() Projection {
	props := map[string]any{"broadcastId": e.BroadcastID}
	if e.URL != "" {
		props["url"] = e.URL
	}
	return Projection{Subject: e.Email, Action: e.Action, Email: e.Email, Properties: props, ContactID: e.ContactID}
}

// CollectedEvent wraps a customer's own event ingested via the collect or external
// API. Its body is the user's domain, stored as-is: Action and Properties are
// opaque to us. It carries no contact, so it never enrolls automations by contact
// — but its Action is the automation trigger key (a user can trigger on "page_view").
type CollectedEvent struct {
	WorkspaceID int64          `json:"workspaceId"`
	SubjectID   string         `json:"subjectId"`
	Action      string         `json:"action"`
	Email       string         `json:"email,omitempty"`
	Phone       string         `json:"phone,omitempty"`
	Prospect    *bool          `json:"prospect,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
	OccurredAt  time.Time      `json:"occurredAt,omitempty"`
}

func (*CollectedEvent) EventName() string  { return NameCollected }
func (*CollectedEvent) EventVersion() int  { return 1 }
func (e *CollectedEvent) Workspace() int64 { return e.WorkspaceID }
func (e *CollectedEvent) Project() Projection {
	return Projection{
		Subject:    e.SubjectID,
		Action:     e.Action,
		Email:      e.Email,
		Phone:      e.Phone,
		Prospect:   e.Prospect,
		Properties: e.Properties,
		OccurredAt: e.OccurredAt,
	}
}
