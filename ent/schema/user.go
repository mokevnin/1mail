package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type User struct {
	ent.Schema
}

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "users"},
	}
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("email").
			NotEmpty().
			Unique(),
		field.String("password_hash").
			Sensitive().
			Optional(),
		// When the user confirmed ownership of their email (nullable = unverified).
		// Verification is soft: login is not gated on it; the UI surfaces a banner.
		field.Time("email_verified_at").
			Optional().
			Nillable(),
		// The generation of the User's sessions (ADR 0020): every session token
		// carries the value current at issuance, and a token whose value differs is
		// rejected. Bumping it ends every session of the User at once.
		field.Int64("session_epoch").
			Default(0),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("memberships", Membership.Type),
		edge.To("sent_invitations", Invitation.Type),
	}
}
