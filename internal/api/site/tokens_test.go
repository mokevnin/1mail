package site_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent/apitoken"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSiteTokensCreateListRevoke(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	// Create returns the full secret once.
	created, err := c.SiteTokensCreate(ctx,
		&siteapi.SiteCreateTokenInput{Name: "CI", Scopes: []string{"contacts:read"}},
		siteapi.SiteTokensCreateParams{Slug: "acme"})
	require.NoError(t, err)
	resp, ok := created.(*siteapi.SiteCreateTokenResponse)
	require.Truef(t, ok, "got %T", created)
	assert.NotEmpty(t, resp.Token)
	assert.NotNil(t, service.ParseToken(resp.Token), "returned a well-formed token value")
	assert.Equal(t, "CI", resp.Resource.Name)

	// The new token is persisted and live (selected from the DB by its public prefix).
	row, err := env.DB.ApiToken.Query().Where(apitoken.Prefix(resp.Resource.Prefix)).Only(ctx)
	require.NoError(t, err)
	assert.Nil(t, row.RevokedAt, "freshly created token is not revoked")

	// Revoke the new token; the row is then marked revoked (a soft delete).
	del, err := c.SiteTokensDelete(ctx, siteapi.SiteTokensDeleteParams{
		Slug: "acme",
		ID:   resp.Resource.ID,
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteTokensDeleteNoContent{}, del)

	revoked, err := env.DB.ApiToken.Query().Where(apitoken.Prefix(resp.Resource.Prefix)).Only(ctx)
	require.NoError(t, err)
	assert.NotNil(t, revoked.RevokedAt, "revoked token carries a revoked_at timestamp")
}

func TestSiteTokensDeleteUnknown(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	del, err := c.SiteTokensDelete(context.Background(), siteapi.SiteTokensDeleteParams{
		Slug: "acme",
		ID:   "999999",
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteTokensDeleteNotFound{}, del)
}

func TestSiteTokensWorkspaceNotFound(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteTokensList(context.Background(), siteapi.SiteTokensListParams{Slug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, res)
}
