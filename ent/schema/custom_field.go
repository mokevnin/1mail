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

// CustomField is the workspace-scoped, typed, named definition of a Contact
// attribute beyond the core fields (email / phone / subject_id / name). It is the
// one attribute concept — there is no schemaless "trait". An unknown key arriving
// from Identify is auto-created here with an inferred type (declared-by-use), but is
// a first-class, renameable definition from the first sight. Per-contact values live
// in Contact.custom_fields keyed by this field's `key`.
type CustomField struct {
	ent.Schema
}

func (CustomField) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "custom_fields"},
	}
}

func (CustomField) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		// Machine key — how values are addressed in Contact.custom_fields and segment
		// rules. Stable; the display name is renameable independently.
		field.String("key").
			NotEmpty(),
		field.String("name").
			NotEmpty(),
		// Inferred on first sight; later values of a conflicting type widen to string
		// (coercion is handled at ingest, not constrained here).
		field.Enum("type").
			Values("string", "number", "bool", "datetime").
			Default("string"),
		field.Int64("workspace_id"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (CustomField) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("custom_fields").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (CustomField) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("key", "workspace_id").Unique().StorageKey("custom_fields_key_workspace_id"),
	}
}
