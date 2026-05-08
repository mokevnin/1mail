package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
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
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}
