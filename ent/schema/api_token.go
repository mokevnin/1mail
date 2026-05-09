package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type ApiToken struct {
	ent.Schema
}

func (ApiToken) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "api_tokens"},
	}
}

func (ApiToken) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("prefix").
			NotEmpty().
			Unique().
			Immutable(),
		field.String("secret_hash").
			NotEmpty().
			Sensitive(),
		field.JSON("scopes", []string{}).
			Default([]string{}),
		field.Time("expires_at").
			Optional().
			Nillable(),
		field.Time("revoked_at").
			Optional().
			Nillable(),
		field.Time("last_used_at").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Int64("workspace_id").
			Optional().
			Nillable(),
	}
}

func (ApiToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("api_tokens").
			Field("workspace_id").
			Unique(),
	}
}
