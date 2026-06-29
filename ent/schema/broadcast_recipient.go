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

// BroadcastRecipient is the per-recipient delivery log for a Broadcast. It is
// the basis for delivery tracking (opens/clicks) and idempotent sending: the
// unique (broadcast_id, contact_id) index guarantees one row per recipient.
type BroadcastRecipient struct {
	ent.Schema
}

func (BroadcastRecipient) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "broadcast_recipients"},
	}
}

func (BroadcastRecipient) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.Int64("broadcast_id"),
		field.Int64("contact_id"),
		field.Int64("workspace_id"),
		field.Enum("status").
			Values("pending", "sent", "failed").
			Default("pending"),
		field.String("error").
			Optional().
			Nillable(),
		field.Time("sent_at").
			Optional().
			Nillable(),
		field.Time("opened_at").
			Optional().
			Nillable(),
		field.Time("clicked_at").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (BroadcastRecipient) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("broadcast", Broadcast.Type).
			Ref("recipients").
			Field("broadcast_id").
			Required().
			Unique(),
		edge.From("workspace", Workspace.Type).
			Ref("broadcast_recipients").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (BroadcastRecipient) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("broadcast_id", "contact_id").
			Unique().
			StorageKey("broadcast_recipients_broadcast_id_contact_id"),
		index.Fields("broadcast_id"),
		// Supports the workspace analytics dashboard, which scans recipients by
		// workspace + delivery time for engagement aggregates and time series.
		index.Fields("workspace_id", "sent_at"),
	}
}
