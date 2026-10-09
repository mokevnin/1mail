package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// OAuthClient is an MCP client that registered itself through dynamic client
// registration (RFC 7591, ADR 0016). All clients are public (PKCE, no secret).
type OAuthClient struct {
	ent.Schema
}

func (OAuthClient) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oauth_clients"},
	}
}

func (OAuthClient) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (OAuthClient) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			StorageKey("id").
			Immutable(),
		// Public identifier handed to the client at registration.
		field.String("client_id").
			NotEmpty().
			Unique().
			Immutable(),
		// Self-declared by the (untrusted) client; shown on the consent screen as text.
		field.String("name").
			NotEmpty(),
		field.JSON("redirect_uris", []string{}),
	}
}

func (OAuthClient) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("codes", OAuthCode.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
