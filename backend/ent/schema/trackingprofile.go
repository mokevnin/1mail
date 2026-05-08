package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type TrackingProfile struct {
	ent.Schema
}

func (TrackingProfile) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "tracking_profiles"},
	}
}

func (TrackingProfile) Fields() []ent.Field {
	return []ent.Field{
		field.String("subject_id").
			NotEmpty().
			Unique(),
		field.String("email").
			Optional().
			Nillable().
			Unique(),
		field.String("phone").
			Optional().
			Nillable().
			Unique(),
		field.JSON("traits", map[string]interface{}{}).
			Default(map[string]interface{}{}),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (TrackingProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("visitors", TrackingVisitor.Type),
	}
}
