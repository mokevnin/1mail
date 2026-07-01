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

// Membership is the join that grants a User access to a Workspace with a Role.
// Workspaces are reached through Memberships, not owned by a single User.
type Membership struct {
	ent.Schema
}

func (Membership) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "memberships"},
	}
}

func (Membership) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.Int64("user_id"),
		field.Int64("workspace_id"),
		// The User's permission level in this Workspace. owner + admin manage
		// members and invites; member cannot. Only owner may transfer ownership.
		field.Enum("role").
			Values("owner", "admin", "member"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Membership) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("memberships").
			Field("user_id").
			Required().
			Unique(),
		edge.From("workspace", Workspace.Type).
			Ref("memberships").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (Membership) Indexes() []ent.Index {
	return []ent.Index{
		// A User has at most one Membership per Workspace.
		index.Fields("user_id", "workspace_id").
			Unique(),
	}
}
