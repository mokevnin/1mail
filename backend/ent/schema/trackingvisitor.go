package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type TrackingVisitor struct {
	ent.Schema
}

func (TrackingVisitor) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "tracking_visitors"},
	}
}

func (TrackingVisitor) Fields() []ent.Field {
	return []ent.Field{
		field.String("visitor_id").
			NotEmpty().
			Unique(),
		field.Int64("profile_id").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Time("last_seen_at").
			Default(time.Now),
	}
}

func (TrackingVisitor) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("profile", TrackingProfile.Type).
			Ref("visitors").
			Field("profile_id").
			Unique(),
	}
}
