package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type Contact struct {
	ent.Schema
}

func (Contact) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "contacts"},
	}
}

func (Contact) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").
			NotEmpty().
			Unique(),
		field.String("first_name").
			Optional().
			Nillable(),
		field.String("last_name").
			Optional().
			Nillable(),
		field.Enum("status").
			Values("active", "unsubscribed").
			Default("active"),
		field.String("time_zone").
			Optional().
			Nillable(),
		field.JSON("custom_fields", map[string]string{}).
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}
