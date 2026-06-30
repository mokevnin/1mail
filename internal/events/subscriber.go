package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/ent/suppression"
)

// Enroller enrolls a contact into automations matching an event action. The
// automations subscriber delegates to it, keeping durable execution in river
// (the implementation enqueues an EvaluateTrigger job).
type Enroller interface {
	OnEvent(ctx context.Context, workspaceID, contactID int64, action string) error
}

// WebhookDispatcher fans an event out to the workspace's matching webhook
// endpoints. The webhooks subscriber delegates to it, keeping durable delivery
// (per-endpoint jobs, retries) in river.
type WebhookDispatcher interface {
	Dispatch(ctx context.Context, workspaceID int64, eventName, deliveryID string, body []byte) error
}

// NewRouter builds the watermill router that hosts the domain-event subscribers.
// The events package owns its full consume-side runtime (router + per-group
// subscribers); the producer side is Bus. Run the returned router in a goroutine
// and Close it on shutdown.
func NewRouter() (*message.Router, error) {
	return message.NewRouter(message.RouterConfig{}, watermill.NewStdLogger(false, false))
}

// RegisterSubscribers wires the domain-event consumers onto the shared watermill
// router. Each consumer gets its OWN subscriber with a distinct consumer group,
// so every subscriber receives every event (fan-out) rather than competing for
// messages. Add new subscribers here without touching producers.
func RegisterSubscribers(router *message.Router, db *sql.DB, client *ent.Client, enroller Enroller, dispatcher WebhookDispatcher) error {
	persistSub, err := NewSubscriber(db, "persist")
	if err != nil {
		return fmt.Errorf("persist subscriber: %w", err)
	}
	router.AddConsumerHandler("persist_event", TopicDomainEvents, persistSub, persistConsumer(client))

	automationsSub, err := NewSubscriber(db, "automations")
	if err != nil {
		return fmt.Errorf("automations subscriber: %w", err)
	}
	router.AddConsumerHandler("enroll_automations", TopicDomainEvents, automationsSub, automationsConsumer(enroller))

	webhooksSub, err := NewSubscriber(db, "webhooks")
	if err != nil {
		return fmt.Errorf("webhooks subscriber: %w", err)
	}
	router.AddConsumerHandler("dispatch_webhooks", TopicDomainEvents, webhooksSub, webhooksConsumer(dispatcher))

	suppressionSub, err := NewSubscriber(db, "suppression")
	if err != nil {
		return fmt.Errorf("suppression subscriber: %w", err)
	}
	router.AddConsumerHandler("update_suppression", TopicDomainEvents, suppressionSub, suppressionConsumer(client))
	return nil
}

// suppressionReason maps a projected event to the suppression reason it implies,
// or false for events that don't suppress (opens, clicks, transient bounces). It
// takes the whole projection because a bounce only suppresses when permanent,
// which lives in the projected properties.
func suppressionReason(p Projection) (suppression.Reason, bool) {
	switch p.Action {
	case NameEmailUnsubscribed:
		return suppression.ReasonUnsubscribed, true
	case NameEmailComplained:
		return suppression.ReasonComplaint, true
	case NameEmailBounced:
		// Only hard (permanent) bounces suppress; transient bounces are temporary.
		if kind, _ := p.Properties["bounceKind"].(string); kind == BounceKindPermanent {
			return suppression.ReasonBounce, true
		}
		return "", false
	default:
		return "", false
	}
}

// suppressionConsumer adds the event's address to the workspace suppression list
// when the action implies it (today: unsubscribe).
func suppressionConsumer(client *ent.Client) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var env Envelope
		if err := json.Unmarshal(msg.Payload, &env); err != nil {
			return fmt.Errorf("unmarshal envelope: %w", err)
		}
		return Suppress(msg.Context(), client, env)
	}
}

// Suppress adds the event's address to the workspace suppression list when the
// action implies a do-not-send (today: unsubscribe; bounce/complaint later).
// No-op for other actions or a missing email. Idempotent — a redelivered or
// repeated event keeps the original entry rather than erroring or overwriting
// its reason. Exported so the logic can be tested without running the router.
func Suppress(ctx context.Context, client *ent.Client, env Envelope) error {
	ev, err := Decode(env)
	if err != nil {
		return err
	}
	p := ev.Project()
	reason, ok := suppressionReason(p)
	if !ok {
		return nil
	}
	email := strings.ToLower(strings.TrimSpace(p.Email))
	if email == "" {
		return nil
	}
	create := client.Suppression.Create().
		SetWorkspaceID(env.WorkspaceID).
		SetEmail(email).
		SetReason(reason)
	if p.ContactID != 0 {
		create.SetContactID(p.ContactID)
	}
	if err := create.
		OnConflictColumns(suppression.FieldWorkspaceID, suppression.FieldEmail).
		Ignore().
		Exec(ctx); err != nil {
		return fmt.Errorf("suppress %q: %w", email, err)
	}
	return nil
}

// webhookPayload is the public JSON delivered to endpoints (and signed).
type webhookPayload struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	OccurredAt  time.Time       `json:"occurredAt"`
	WorkspaceID int64           `json:"workspaceId"`
	Subject     string          `json:"subject"`
	ContactID   int64           `json:"contactId,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
}

func webhooksConsumer(dispatcher WebhookDispatcher) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var env Envelope
		if err := json.Unmarshal(msg.Payload, &env); err != nil {
			return fmt.Errorf("unmarshal envelope: %w", err)
		}
		ev, err := Decode(env)
		if err != nil {
			return err
		}
		p := ev.Project()
		body, err := json.Marshal(webhookPayload{
			ID:          env.ID,
			Type:        p.Action,
			OccurredAt:  env.OccurredAt,
			WorkspaceID: env.WorkspaceID,
			Subject:     p.Subject,
			ContactID:   p.ContactID,
			Data:        env.Data,
		})
		if err != nil {
			return fmt.Errorf("marshal webhook payload: %w", err)
		}
		// Filter/route on the semantic action (e.g. "page_view"), not the bus type.
		return dispatcher.Dispatch(msg.Context(), env.WorkspaceID, p.Action, env.ID, body)
	}
}

// automationsConsumer enrolls the event's contact into matching automations. It
// skips events with no contact (ContactID == 0, e.g. anonymous collect events).
func automationsConsumer(enroller Enroller) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var env Envelope
		if err := json.Unmarshal(msg.Payload, &env); err != nil {
			return fmt.Errorf("unmarshal envelope: %w", err)
		}
		ev, err := Decode(env)
		if err != nil {
			return err
		}
		p := ev.Project()
		if p.ContactID == 0 {
			return nil // no contact to enroll (e.g. an anonymous collected event)
		}
		// Enroll on the semantic action so a trigger can be "contact.created" or a
		// customer action like "page_view".
		return enroller.OnEvent(msg.Context(), env.WorkspaceID, p.ContactID, p.Action)
	}
}

// NewSubscriber builds a watermill-sql subscriber for the outbox topic under a
// dedicated consumer group (its own offset cursor). InitializeSchema is false —
// InitSchema creates the message + offsets tables at boot.
func NewSubscriber(db *sql.DB, consumerGroup string) (message.Subscriber, error) {
	return watermillsql.NewSubscriber(db, watermillsql.SubscriberConfig{
		SchemaAdapter:    outboxSchema,
		OffsetsAdapter:   outboxOffsets,
		InitializeSchema: false,
		ConsumerGroup:    consumerGroup,
	}, watermill.NewStdLogger(false, false))
}

func persistConsumer(client *ent.Client) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var env Envelope
		if err := json.Unmarshal(msg.Payload, &env); err != nil {
			return fmt.Errorf("unmarshal envelope: %w", err)
		}
		return Persist(msg.Context(), client, env)
	}
}

// Persist writes the durable Event-projection row for a domain event. It is the
// one place the engagement log is written, replacing scattered Event.Create
// calls. Exported so the projection logic can be unit-tested directly without
// running the router.
//
// It decodes the typed event and writes the row from its Project(), so any event
// type projects faithfully without persist knowing the concrete fields.
func Persist(ctx context.Context, client *ent.Client, env Envelope) error {
	ev, err := Decode(env)
	if err != nil {
		return err
	}
	p := ev.Project()

	create := client.Event.Create().
		SetWorkspaceID(env.WorkspaceID).
		SetSubjectID(p.Subject).
		SetAction(p.Action).
		SetProperties(p.Properties).
		SetOccurredAt(env.OccurredAt)
	// Dedup on the DedupKey when the event carries a natural upstream id, else the
	// envelope ULID. Empty ⇒ leave source_id NULL (distinct under the unique index)
	// rather than "", so a missing id never collides and false-dedupes a distinct event.
	sourceID := env.ID
	if env.DedupKey != "" {
		sourceID = env.DedupKey
	}
	if sourceID != "" {
		create.SetSourceID(sourceID)
	}
	if p.Email != "" {
		create.SetEmail(p.Email)
	}
	if p.Phone != "" {
		create.SetPhone(p.Phone)
	}
	if p.Prospect != nil {
		create.SetProspect(*p.Prospect)
	}
	// Idempotent: at-least-once delivery can redeliver an envelope after a crash;
	// dedupe on the source_id (the envelope ULID) so the projection row is written
	// at most once.
	if err := create.OnConflictColumns(event.FieldSourceID).Ignore().Exec(ctx); err != nil {
		return fmt.Errorf("persist event %q: %w", env.Name, err)
	}
	return nil
}
