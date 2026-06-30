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

// Visitor is an anonymous device/browser identity — a visitor_id cookie, unique per
// workspace. Before Identify it may belong to no Contact (contact_id null); Identify
// binds it to a Contact, and one Contact owns many Visitors (the same person across
// devices).
type Visitor struct {
	ent.Schema
}

func (Visitor) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "visitors"},
	}
}

func (Visitor) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.String("visitor_id").
			NotEmpty(),
		field.Int64("workspace_id"),
		// Null until Identify resolves who this device is.
		field.Int64("contact_id").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Time("last_seen_at").
			Default(time.Now),
	}
}

func (Visitor) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("contact", Contact.Type).
			Ref("visitors").
			Field("contact_id").
			Unique(),
		edge.From("workspace", Workspace.Type).
			Ref("visitors").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (Visitor) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("visitor_id", "workspace_id").Unique().StorageKey("visitors_visitor_id_workspace_id"),
	}
}
