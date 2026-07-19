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
		// Confirmed opt-in policy (ADR 0013). When true, marketing sends require a
		// Confirmation for the destination (double opt-in); when false (the default),
		// the subtractive single-opt-in model applies unchanged. Forward-looking:
		// enabling it backfills grandfathered confirmations rather than muting the
		// existing list. Schema-only for now — exposed on the /site API in a later slice.
		field.Bool("require_confirmed_opt_in").
			Default(false),
		// Physical postal address printed in the marketing email footer, as
		// CAN-SPAM 15 U.S.C. §7704(a)(5) requires of every commercial message.
		// Freeform so it carries the legal sender name + address on one field
		// (the "one powerful primitive" convention). Optional and non-gating for
		// now: a workspace can still send without it — whether an empty address
		// should hard-block marketing sends (like the verified-domain gate) is a
		// separate send-path ADR. Transactional mail (ADR 0005) is exempt and
		// never renders this footer.
		field.String("postal_address").
			Optional().
			Default(""),
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
		edge.To("confirmations", Confirmation.Type),
		edge.To("transactional_emails", TransactionalEmail.Type),
		edge.To("memberships", Membership.Type),
		edge.To("invitations", Invitation.Type),
	}
}
