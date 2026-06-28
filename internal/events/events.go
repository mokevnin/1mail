// Package events is the internal domain-event system: a domain event (e.g.
// contact.created) is published once, inside the same transaction that commits
// the state change (transactional outbox), and fanned out to any number of
// independent subscribers over watermill. See docs/design/domain-events.md.
//
// The bus carries a single generic Envelope on one topic; subscribers
// (persist → Event projection, later automations/webhooks/analytics) consume
// every event and dispatch on Envelope.Name. Producer code stays typed: a
// concrete DomainEvent struct is marshaled into the envelope payload.
package events

import (
	"encoding/json"
	"time"
)

// TopicDomainEvents is the single watermill topic all domain events flow
// through. watermill-sql maps it to the table "watermill_domain_events", which
// doubles as the transactional outbox.
const TopicDomainEvents = "domain_events"

// Event name vocabulary — "aggregate.verb" (past tense). These names are also
// the automation trigger_event values, so the taxonomy and the trigger
// vocabulary are one and the same.
const (
	NameContactCreated = "contact.created"
)

// Envelope is the generic bus message. It carries enough for the projection
// (Subject, Name, OccurredAt) without the subscriber knowing the concrete type;
// numeric ids and type-specific fields live in Payload.
type Envelope struct {
	ID          string          `json:"id"`          // ULID; idempotency key for consumers
	Name        string          `json:"name"`        // e.g. "contact.created"
	Version     int             `json:"version"`     // payload schema version
	WorkspaceID int64           `json:"workspaceId"` // every event is workspace-scoped
	Subject     string          `json:"subject"`     // identity string → events.subject_id (email/phone/visitor)
	OccurredAt  time.Time       `json:"occurredAt"`
	Payload     json.RawMessage `json:"payload"` // marshaled DomainEvent
}

// DomainEvent is a typed, producer-side domain event. Implementations carry the
// event's data as struct fields (marshaled into Envelope.Payload) and report the
// envelope metadata the bus needs.
type DomainEvent interface {
	// EventName returns the stable "aggregate.verb" name.
	EventName() string
	// EventVersion is the payload schema version (bumped on additive changes).
	EventVersion() int
	// Workspace is the owning workspace id.
	Workspace() int64
	// Subject is the identity string the projection keys on (email/phone/visitor).
	Subject() string
}

// ContactCreated is emitted when a contact is created in a workspace.
type ContactCreated struct {
	WorkspaceID int64  `json:"workspaceId"`
	ContactID   int64  `json:"contactId"`
	Email       string `json:"email"`
}

func (ContactCreated) EventName() string  { return NameContactCreated }
func (ContactCreated) EventVersion() int  { return 1 }
func (e ContactCreated) Workspace() int64 { return e.WorkspaceID }
func (e ContactCreated) Subject() string  { return e.Email }
