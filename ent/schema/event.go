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

type Event struct {
	ent.Schema
}

func (Event) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "events"},
	}
}

func (Event) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		// source_id is the originating domain-event id (ULID) for rows written by
		// the events bus persist subscriber. Unique so at-least-once redelivery
		// dedupes via upsert. Nillable: rows written outside the bus (fixtures)
		// leave it null, and Postgres treats nulls as distinct.
		field.String("source_id").
			Optional().
			Nillable().
			Unique(),
		field.String("subject_id").
			NotEmpty(),
		field.String("email").
			Optional().
			Nillable(),
		field.String("phone").
			Optional().
			Nillable(),
		field.String("action").
			NotEmpty(),
		field.JSON("properties", map[string]interface{}{}).
			Optional(),
		field.Time("occurred_at").
			Optional().
			Nillable(),
		field.Bool("prospect").
			Optional().
			Nillable(),
		field.Int64("workspace_id"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (Event) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("events").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (Event) Indexes() []ent.Index {
	return []ent.Index{
		// Backs event-based segment conditions: the correlated EXISTS filters by
		// workspace + email + action (the contact↔event join + the event type).
		index.Fields("workspace_id", "email", "action"),
	}
}
