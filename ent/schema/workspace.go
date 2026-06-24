package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Workspace struct {
	ent.Schema
}

func (Workspace) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "workspaces"},
	}
}

func (Workspace) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("slug").
			NotEmpty().
			Unique(),
		field.String("collect_key").
			NotEmpty().
			Unique().
			Sensitive(),
		field.Int64("user_id").
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

func (Workspace) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("contacts", Contact.Type),
		edge.To("segments", Segment.Type),
		edge.To("events", Event.Type),
		edge.To("tracking_profiles", TrackingProfile.Type),
		edge.To("tracking_visitors", TrackingVisitor.Type),
		edge.To("api_tokens", ApiToken.Type),
		edge.From("user", User.Type).
			Ref("workspaces").
			Field("user_id").
			Unique(),
	}
}
