package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/mokevnin/1mail/ent"
)

// Enroller enrolls a contact into automations matching an event action. The
// automations subscriber delegates to it, keeping durable execution in river
// (the implementation enqueues an EvaluateTrigger job).
type Enroller interface {
	OnEvent(ctx context.Context, workspaceID, contactID int64, action string) error
}

// RegisterSubscribers wires the domain-event consumers onto the shared watermill
// router. Each consumer gets its OWN subscriber with a distinct consumer group,
// so every subscriber receives every event (fan-out) rather than competing for
// messages. Add new subscribers here without touching producers.
func RegisterSubscribers(router *message.Router, db *sql.DB, client *ent.Client, enroller Enroller) error {
	persistSub, err := newSubscriber(db, "persist")
	if err != nil {
		return fmt.Errorf("persist subscriber: %w", err)
	}
	router.AddConsumerHandler("persist_event", TopicDomainEvents, persistSub, persistConsumer(client))

	automationsSub, err := newSubscriber(db, "automations")
	if err != nil {
		return fmt.Errorf("automations subscriber: %w", err)
	}
	router.AddConsumerHandler("enroll_automations", TopicDomainEvents, automationsSub, automationsConsumer(enroller))
	return nil
}

// automationsConsumer enrolls the event's contact into matching automations. It
// skips events with no contact (ContactID == 0, e.g. anonymous collect events).
func automationsConsumer(enroller Enroller) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var env Envelope
		if err := json.Unmarshal(msg.Payload, &env); err != nil {
			return fmt.Errorf("unmarshal envelope: %w", err)
		}
		if env.ContactID == 0 {
			return nil
		}
		return enroller.OnEvent(msg.Context(), env.WorkspaceID, env.ContactID, env.Name)
	}
}

// newSubscriber builds a watermill-sql subscriber for the outbox topic under a
// dedicated consumer group (its own offset cursor). InitializeSchema is false —
// InitSchema creates the message + offsets tables at boot.
func newSubscriber(db *sql.DB, consumerGroup string) (message.Subscriber, error) {
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
// The mapping is generic: subject_id/action/workspace come from the envelope and
// the whole payload is stored as queryable properties. Type-aware fields (email)
// are lifted from the payload when present.
func Persist(ctx context.Context, client *ent.Client, env Envelope) error {
	props := map[string]any{}
	if len(env.Payload) > 0 {
		if err := json.Unmarshal(env.Payload, &props); err != nil {
			return fmt.Errorf("unmarshal payload: %w", err)
		}
	}

	create := client.Event.Create().
		SetWorkspaceID(env.WorkspaceID).
		SetSubjectID(env.Subject).
		SetAction(env.Name).
		SetProperties(props).
		SetOccurredAt(env.OccurredAt)
	if email, ok := props["email"].(string); ok && email != "" {
		create.SetEmail(email)
	}
	if _, err := create.Save(ctx); err != nil {
		return fmt.Errorf("persist event %q: %w", env.Name, err)
	}
	return nil
}
