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

// Automation is a workspace-scoped, event-triggered workflow. A contact is
// enrolled when an Event whose action equals trigger_event is recorded for them;
// the run then walks the linear list of steps in definition (JSON). Branch/goal
// nodes and a visual builder come later.
type Automation struct {
	ent.Schema
}

func (Automation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "automations"},
	}
}

func (Automation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.Enum("status").
			Values("draft", "active").
			Default("draft"),
		// The Event.action that enrolls a contact (e.g. "contact.created",
		// "email.opened", a custom collect event).
		field.String("trigger_event").
			NotEmpty(),
		// JSON array of steps: [{"type":"email","subject":..,"body":<mjml>},
		// {"type":"wait","seconds":N}].
		field.String("definition").
			Default("[]"),
		field.Int64("workspace_id"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Automation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("automations").
			Field("workspace_id").
			Required().
			Unique(),
		edge.To("runs", AutomationRun.Type),
	}
}

func (Automation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id"),
		// Trigger lookup: active automations for a given event action.
		index.Fields("workspace_id", "status", "trigger_event"),
	}
}
