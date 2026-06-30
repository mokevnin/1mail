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

// Suppression is the workspace's authoritative, (channel, destination)-keyed
// do-not-send registry for hard, global facts: hard bounce, spam complaint, and
// manual bans. It is the send path's global hard floor, checked on every send,
// and a compliance ratchet (bounce/complaint do not auto-clear). Keyed by
// (channel, destination) — not by contact — so it covers destinations with no
// contact (e.g. a bounce) and survives contact deletion/re-import. The
// destination is stored normalized (lower-cased) and unique per (workspace,
// channel), so ingestion can upsert idempotently. Distinct from Unsubscribe:
// Suppression is global and hard; an unsubscribe is per-sending-source and
// toggleable (see the Unsubscribe entity).
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
		// The channel this suppression applies to (email today; sms reserved).
		field.Enum("channel").
			Values("email").
			Default("email"),
		// Normalized (lower-cased) channel-specific destination address.
		field.String("destination").
			NotEmpty(),
		// Why the destination is suppressed. Unsubscribe is NOT a suppression
		// reason — it lives in the separate, toggleable Unsubscribe entity.
		field.Enum("reason").
			Values("bounce", "complaint", "manual").
			Default("manual"),
		// The contact this destination belongs to, when known (a bounce/complaint
		// may arrive for a destination with no contact). Display only. Nullable.
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
		// One suppression per (channel, destination) per workspace; lets ingestion upsert.
		index.Fields("workspace_id", "channel", "destination").
			Unique().
			StorageKey("suppressions_workspace_id_channel_destination"),
	}
}
