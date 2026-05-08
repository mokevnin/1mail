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

type WorkspaceMembership struct {
	ent.Schema
}

func (WorkspaceMembership) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "workspace_memberships"},
	}
}

func (WorkspaceMembership) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Optional().
			Nillable(),
		field.Int64("workspace_id").
			Optional().
			Nillable(),
		field.Enum("role").
			Values("owner", "admin", "member").
			Default("member"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (WorkspaceMembership) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("workspace_memberships").
			Field("user_id").
			Unique(),
		edge.From("workspace", Workspace.Type).
			Ref("memberships").
			Field("workspace_id").
			Unique(),
	}
}

func (WorkspaceMembership) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "workspace_id").Unique(),
	}
}
