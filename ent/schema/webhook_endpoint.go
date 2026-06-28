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

// WebhookEndpoint is a workspace-scoped HTTP destination that receives domain
// events. The signing secret is stored encrypted (like integration creds); the
// "webhooks" bus subscriber fans matching events out to a river delivery job.
type WebhookEndpoint struct {
	ent.Schema
}

func (WebhookEndpoint) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "webhook_endpoints"},
	}
}

func (WebhookEndpoint) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.String("url").
			NotEmpty(),
		// HMAC signing secret, encrypted at rest via internal/secrets.Cipher.
		field.String("secret_encrypted").
			Sensitive(),
		// Event names this endpoint subscribes to; empty/nil means all events.
		field.Strings("event_types").
			Optional(),
		field.Bool("enabled").
			Default(true),
		field.Int64("workspace_id"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (WebhookEndpoint) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("webhook_endpoints").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (WebhookEndpoint) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id"),
		index.Fields("workspace_id", "enabled"),
	}
}
