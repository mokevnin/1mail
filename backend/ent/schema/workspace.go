package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Workspace struct {
	ent.Schema
}

func (Workspace) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "workspaces"},
	}
}

func (Workspace) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty(),
		field.String("slug").
			NotEmpty().
			Unique(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Workspace) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("memberships", WorkspaceMembership.Type),
		edge.To("projects", Project.Type),
	}
}
