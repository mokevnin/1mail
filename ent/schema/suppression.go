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

// Suppression is a workspace-scoped "do not send" registry entry. It is the
// central deliverability list: an address is suppressed when it unsubscribes,
// hard-bounces, files a spam complaint, or is added manually. The send path
// skips any recipient whose (lower-cased) email is present here, independent of
// the contact's status — so addresses with no contact (e.g. a bounce) can be
// suppressed too. Email is stored normalized (lower-cased) and unique per
// workspace, so ingestion can upsert idempotently.
type Suppression struct {
	ent.Schema
}

func (Suppression) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "suppressions"},
	}
}

func (Suppression) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		// Normalized (lower-cased) email address.
		field.String("email").
			NotEmpty(),
		// Why the address is suppressed.
		field.Enum("reason").
			Values("unsubscribed", "bounce", "complaint", "manual").
			Default("manual"),
		// The contact this address belongs to, when known (a bounce/complaint may
		// arrive for an address with no contact). Nullable.
		field.Int64("contact_id").
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

func (Suppression) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("suppressions").
			Field("workspace_id").
			Required().
			Unique(),
	}
}

func (Suppression) Indexes() []ent.Index {
	return []ent.Index{
		// One suppression per address per workspace; lets ingestion upsert.
		index.Fields("workspace_id", "email").
			Unique().
			StorageKey("suppressions_workspace_id_email"),
	}
}
