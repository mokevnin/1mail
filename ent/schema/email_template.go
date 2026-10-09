package schema

import (
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

func (EmailTemplate) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
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
		// Email body authored as MJML; compiled to email-safe HTML on send.
		field.String("body").
			Default(""),
		field.Int64("workspace_id"),
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
