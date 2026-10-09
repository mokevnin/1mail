package schema

import (
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

func (Membership) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}, WorkspaceMixin{Ref: "memberships"}}
}

func (Membership) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.Int64("user_id"),
		// The User's permission level in this Workspace. owner + admin manage
		// members and invites; member cannot. Only owner may transfer ownership.
		field.Enum("role").
			Values("owner", "admin", "member"),
	}
}

func (Membership) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("memberships").
			Field("user_id").
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
