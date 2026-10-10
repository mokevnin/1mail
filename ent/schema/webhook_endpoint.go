package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
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
		Audited{Action: "webhook_endpoint", NameField: "url", NameIsURL: true},
	}
}

func (WebhookEndpoint) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}, WorkspaceMixin{Ref: "webhook_endpoints"}}
}

func (WebhookEndpoint) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		// The URL may carry a secret (a token in the path or query), so a diff records
		// only that it changed, and the name snapshot drops the query and userinfo.
		field.String("url").
			NotEmpty().
			Annotations(Sensitive{}),
		// HMAC signing secret, encrypted at rest via internal/secrets.Cipher.
		field.String("secret_encrypted").
			Sensitive().
			Annotations(Sensitive{}),
		// Event names this endpoint subscribes to; empty/nil means all events.
		field.Strings("event_types").
			Optional(),
		field.Bool("enabled").
			Default(true),
	}
}

func (WebhookEndpoint) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id"),
		index.Fields("workspace_id", "enabled"),
	}
}
