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

// SendingDomain is a workspace-scoped, 1mail-authenticated sending identity
// (ADR 0010). 1mail signs outbound mail itself: it generates a per-domain DKIM
// keypair, the user publishes one <selector>._domainkey TXT, and the same
// verified domain works across every transport (smtp/ses/…) with no
// re-verification when the Integration changes. Distinct from a Sending source
// (the unsubscribe consent scope) despite the similar name.
//
// verified is a *live* property, not a one-time flag: a periodic re-check
// re-validates the DKIM DNS and flips it if the record disappears (see
// internal/jobs). The private key is stored Tink-encrypted; the public key is
// the TXT value and is safe to expose.
type SendingDomain struct {
	ent.Schema
}

func (SendingDomain) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sending_domains"},
	}
}

func (SendingDomain) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		// The authenticated domain, e.g. "mail.acme.com".
		field.String("domain").
			NotEmpty(),
		// DKIM selector; the DNS record is "<dkim_selector>._domainkey.<domain>".
		field.String("dkim_selector").
			NotEmpty(),
		// Tink-encrypted PKCS#8 PEM private key (internal/secrets.Cipher).
		field.String("dkim_private_key_encrypted").
			Sensitive(),
		// The DKIM TXT value ("v=DKIM1; k=rsa; p=<base64 DER>"); safe to expose.
		field.String("dkim_public_key").
			NotEmpty(),
		// Live DKIM-verified state, re-checked by the periodic job.
		field.Bool("verified").
			Default(false),
		// When the DKIM DNS was most recently checked (nil = never checked yet).
		field.Time("last_checked_at").
			Optional().
			Nillable(),
		// When the domain most recently became verified (nil = never verified).
		field.Time("verified_at").
			Optional().
			Nillable(),
		field.Int64("workspace_id"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (SendingDomain) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("sending_domains").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (SendingDomain) Indexes() []ent.Index {
	return []ent.Index{
		// One row per (workspace, domain).
		index.Fields("workspace_id", "domain").
			Unique(),
		// Lets the periodic re-check job find the least-recently-checked rows.
		index.Fields("last_checked_at"),
	}
}
