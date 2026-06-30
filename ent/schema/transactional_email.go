package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TransactionalEmail is the per-send record for the transactional surface (ADR
// 0005): one row per API-triggered single-recipient send. It is the transactional
// counterpart of BroadcastRecipient — a durable, queryable send trace — and the
// synchronous claim that makes a client Idempotency-Key safe: the row is inserted
// before the provider call, so a concurrent or retried request with the same key
// loses the unique-index race and replays the winner's outcome instead of sending
// a second email (the Event-log DedupKey only dedupes the event row, never the
// send). The referenced template is stored as a plain id snapshot, never an FK:
// the record must outlive a deleted template, and (per ADR 0003/0005) carries no
// copy of the rendered content (PII + volume) — only the provenance and outcome.
type TransactionalEmail struct {
	ent.Schema
}

func (TransactionalEmail) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "transactional_emails"},
	}
}

func (TransactionalEmail) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		// The channel this send went out on (email today; sms reserved).
		field.Enum("channel").
			Values("email").
			Default("email"),
		// Normalized (lower-cased) channel-specific destination address.
		field.String("destination").
			NotEmpty(),
		// The Template rendered at send time, bound by reference (ADR 0005). A plain
		// id snapshot, not an FK: the record outlives a deleted template.
		field.Int64("template_id"),
		// The contact this destination resolved to at send time, when one exists
		// (transactional mail may go to an address with no contact). Display only,
		// nullable; the send fact also attaches to it in the Event log.
		field.Int64("contact_id").
			Optional().
			Nillable(),
		// Outcome: accepted by the provider (sent), blocked by Suppression
		// (suppressed — no send happened), or the provider call failed (failed).
		field.Enum("status").
			Values("pending", "sent", "suppressed", "failed").
			Default("pending"),
		// Provider error when status=failed. Nullable.
		field.String("error").
			Optional().
			Nillable(),
		// Client-supplied Idempotency-Key (Stripe-style). Nullable: a keyless send is
		// always accepted. The unique index below includes it, and Postgres treats
		// NULLs as distinct, so keyless rows never collide while a repeated key does.
		field.String("idempotency_key").
			Optional().
			Nillable(),
		field.Int64("workspace_id"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (TransactionalEmail) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("transactional_emails").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (TransactionalEmail) Indexes() []ent.Index {
	return []ent.Index{
		// Idempotency claim: at most one row per (workspace, key). A keyless send
		// (idempotency_key NULL) is exempt — NULLs are distinct in Postgres.
		index.Fields("workspace_id", "idempotency_key").
			Unique().
			StorageKey("transactional_emails_workspace_id_idempotency_key"),
		// Recency listing for the send-history UI.
		index.Fields("workspace_id", "created_at"),
	}
}
