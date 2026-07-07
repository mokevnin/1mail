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
		// Secret per-workspace key that routes inbound provider webhooks
		// (/hooks/{ingest_key}/{provider}, e.g. SES bounce/complaint via SNS).
		// Distinct from the browser-exposed collect_key.
		field.String("ingest_key").
			NotEmpty().
			Unique().
			Sensitive(),
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
		edge.To("custom_fields", CustomField.Type),
		edge.To("segments", Segment.Type),
		edge.To("events", Event.Type),
		edge.To("visitors", Visitor.Type),
		edge.To("api_tokens", ApiToken.Type),
		edge.To("integrations", Integration.Type),
		edge.To("sending_domains", SendingDomain.Type),
		edge.To("broadcasts", Broadcast.Type),
		edge.To("broadcast_recipients", BroadcastRecipient.Type),
		edge.To("email_templates", EmailTemplate.Type),
		edge.To("automations", Automation.Type),
		edge.To("automation_runs", AutomationRun.Type),
		edge.To("webhook_endpoints", WebhookEndpoint.Type),
		edge.To("suppressions", Suppression.Type),
		edge.To("unsubscribes", Unsubscribe.Type),
		edge.To("transactional_emails", TransactionalEmail.Type),
		edge.To("memberships", Membership.Type),
		edge.To("invitations", Invitation.Type),
	}
}
