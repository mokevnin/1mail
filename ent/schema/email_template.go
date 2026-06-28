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

// EmailTemplate is a workspace-scoped, reusable email body. Broadcasts copy from
// a template (no FK) so editing or deleting a template never affects a sent or
// in-flight broadcast.
type EmailTemplate struct {
	ent.Schema
}

func (EmailTemplate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "email_templates"},
	}
}

func (EmailTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("subject").
			Default(""),
		field.String("body_html").
			Default(""),
		field.Enum("body_format").
			Values("html", "mjml").
			Default("html"),
		field.Int64("workspace_id"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (EmailTemplate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("email_templates").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (EmailTemplate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id"),
	}
}
