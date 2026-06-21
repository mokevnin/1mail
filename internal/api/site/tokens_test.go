package site_test

import (
	"context"
	"testing"

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
		siteapi.SiteTokensCreateParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	resp, ok := created.(*siteapi.SiteCreateTokenResponse)
	require.Truef(t, ok, "got %T", created)
	assert.NotEmpty(t, resp.Token)
	assert.NotNil(t, service.ParseToken(resp.Token), "returned a well-formed token value")
	assert.Equal(t, "CI", resp.Resource.Name)

	// List includes the fixture token plus the new one, with no secrets.
	listed, err := c.SiteTokensList(ctx, siteapi.SiteTokensListParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	list, ok := listed.(*siteapi.SiteTokensListOKApplicationJSON)
	require.Truef(t, ok, "got %T", listed)
	require.Len(t, *list, 2)

	// Revoke the new token; it disappears from the list.
	del, err := c.SiteTokensDelete(ctx, siteapi.SiteTokensDeleteParams{
		WorkspaceSlug: "acme",
		ID:            resp.Resource.ID,
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteTokensDeleteNoContent{}, del)

	after, err := c.SiteTokensList(ctx, siteapi.SiteTokensListParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	list2 := after.(*siteapi.SiteTokensListOKApplicationJSON)
	assert.Len(t, *list2, 1)
}

func TestSiteTokensDeleteUnknown(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	del, err := c.SiteTokensDelete(context.Background(), siteapi.SiteTokensDeleteParams{
		WorkspaceSlug: "acme",
		ID:            "999999",
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteTokensDeleteNotFound{}, del)
}

func TestSiteTokensWorkspaceNotFound(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteTokensList(context.Background(), siteapi.SiteTokensListParams{WorkspaceSlug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, res)
}
