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
			NotEmpty(),
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
		field.Int64("workspace_project_id").
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

func (Contact) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("contacts").
			Field("workspace_project_id").
			Unique(),
	}
}

func (Contact) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email", "workspace_project_id").Unique(),
	}
}
