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

type TrackingProfile struct {
	ent.Schema
}

func (TrackingProfile) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "tracking_profiles"},
	}
}

func (TrackingProfile) Fields() []ent.Field {
	return []ent.Field{
		field.String("subject_id").
			NotEmpty(),
		field.String("email").
			Optional().
			Nillable(),
		field.String("phone").
			Optional().
			Nillable(),
		field.JSON("traits", map[string]interface{}{}).
			Default(map[string]interface{}{}),
		field.Int64("workspace_project_id").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (TrackingProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("visitors", TrackingVisitor.Type),
		edge.From("project", Project.Type).
			Ref("tracking_profiles").
			Field("workspace_project_id").
			Unique(),
	}
}

func (TrackingProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("subject_id", "workspace_project_id").Unique(),
		index.Fields("email", "workspace_project_id").Unique(),
		index.Fields("phone", "workspace_project_id").Unique(),
	}
}
