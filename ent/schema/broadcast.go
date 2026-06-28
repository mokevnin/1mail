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

// Broadcast is a workspace-scoped one-off email campaign. Its lifecycle is
// draft -> scheduled -> sending -> sent (or failed). The body is authored as
// HTML; body_text is derived for the plain-text part. Audience is the segment
// when segment_id is set, otherwise all active contacts in the workspace.
// Aggregate counters are denormalized for cheap reporting; per-recipient state
// lives in BroadcastRecipient.
type Broadcast struct {
	ent.Schema
}

func (Broadcast) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "broadcasts"},
	}
}

func (Broadcast) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("subject").
			Default(""),
		field.String("from_name").
			Optional().
			Nillable(),
		field.String("from_email").
			Optional().
			Nillable(),
		field.String("body_html").
			Default(""),
		field.String("body_text").
			Default(""),
		// Nil segment_id means "all active contacts in the workspace".
		field.Int64("segment_id").
			Optional().
			Nillable(),
		// Nil integration_id means "the workspace default email integration".
		field.Int64("integration_id").
			Optional().
			Nillable(),
		field.Enum("status").
			Values("draft", "scheduled", "sending", "sent", "failed").
			Default("draft"),
		field.Time("scheduled_at").
			Optional().
			Nillable(),
		field.Time("sent_at").
			Optional().
			Nillable(),
		field.Int("recipients_total").
			Default(0).
			NonNegative(),
		field.Int("sent_count").
			Default(0).
			NonNegative(),
		field.Int("opened_count").
			Default(0).
			NonNegative(),
		field.Int("clicked_count").
			Default(0).
			NonNegative(),
		field.Int("unsubscribed_count").
			Default(0).
			NonNegative(),
		field.Int("failed_count").
			Default(0).
			NonNegative(),
		field.Int64("workspace_id"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Broadcast) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("broadcasts").
			Field("workspace_id").
			Required().
			Unique(),
		edge.To("recipients", BroadcastRecipient.Type),
	}
}

func (Broadcast) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id"),
	}
}
