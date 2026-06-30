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
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		// Alias keys: subject_id (the customer's own user id), email, phone. Each is
		// unique per workspace and ANY may be absent — an anonymous Contact has none.
		// Identity is multi-key; the stable anchor is the internal id, not the email.
		field.String("subject_id").
			Optional().
			Nillable(),
		field.String("email").
			Optional().
			Nillable(),
		field.String("phone").
			Optional().
			Nillable(),
		field.String("first_name").
			Optional().
			Nillable(),
		field.String("last_name").
			Optional().
			Nillable(),
		field.String("time_zone").
			Optional().
			Nillable(),
		// Typed Custom field values keyed by the field's machine key. The typed
		// definitions live in CustomField; values are widened to any to carry
		// number/bool/datetime, not only strings.
		field.JSON("custom_fields", map[string]any{}).
			Optional(),
		field.Int64("workspace_id"),
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
		// The anonymous devices seen as this person. A Visitor may belong to no
		// Contact before Identify; once bound, one Contact owns many Visitors.
		edge.To("visitors", Visitor.Type),
		edge.From("workspace", Workspace.Type).
			Ref("contacts").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (Contact) Indexes() []ent.Index {
	return []ent.Index{
		// Each alias key is unique per workspace; all three are nullable and Postgres
		// treats nulls as distinct, so any may be absent without colliding.
		index.Fields("email", "workspace_id").Unique().StorageKey("contacts_email_workspace_id"),
		index.Fields("subject_id", "workspace_id").Unique().StorageKey("contacts_subject_id_workspace_id"),
		index.Fields("phone", "workspace_id").Unique().StorageKey("contacts_phone_workspace_id"),
	}
}
