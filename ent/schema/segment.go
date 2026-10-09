package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Segment struct {
	ent.Schema
}

func (Segment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "segments"},
	}
}

func (Segment) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (Segment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("definition").
			Optional().
			Nillable(),
		field.Int64("workspace_id"),
	}
}

func (Segment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("segments").
			Field("workspace_id").
			Required().
			Unique(),
	}
}
