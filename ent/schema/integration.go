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

// Integration is a workspace-scoped connection to an external sending provider.
// It is channel-agnostic on purpose: "email" today (smtp/ses), "sms" and others
// plug in the same way later by adding enum values + a catalog descriptor, with
// no schema reshape. Provider-specific credentials live encrypted in
// config_encrypted (the cleartext JSON shape is owned by the messaging catalog).
type Integration struct {
	ent.Schema
}

func (Integration) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "integrations"},
	}
}

func (Integration) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.String("name").
			NotEmpty(),
		// Channel reserves "sms" now so the column/type is ready before any SMS
		// provider exists.
		field.Enum("channel").
			Values("email", "sms").
			Default("email"),
		field.Enum("provider").
			Values("smtp", "ses"),
		// Encrypted JSON blob produced by internal/secrets.Cipher.
		field.String("config_encrypted").
			Sensitive(),
		field.Bool("enabled").
			Default(true),
		field.Bool("is_default").
			Default(false),
		field.Int64("workspace_id"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Integration) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("integrations").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (Integration) Indexes() []ent.Index {
	return []ent.Index{
		// At most one default provider per (workspace, channel). Partial index so
		// only is_default rows are constrained.
		index.Fields("workspace_id", "channel").
			Annotations(entsql.IndexWhere("is_default")).
			Unique(),
	}
}
