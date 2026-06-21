package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSiteWorkspacesUpdateRenames(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	res, err := c.SiteWorkspacesUpdate(ctx,
		&siteapi.SiteUpdateWorkspaceInput{Name: "Acme Inc"},
		siteapi.SiteWorkspacesUpdateParams{Slug: "acme"})
	require.NoError(t, err)
	updated, ok := res.(*siteapi.SiteWorkspaceResource)
	require.Truef(t, ok, "got %T", res)
	assert.Equal(t, "Acme Inc", updated.Name)
	assert.Equal(t, "acme", updated.Slug, "slug is immutable")

	// The change is reflected in the list.
	list, err := c.SiteWorkspacesList(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Acme Inc", list[0].Name)
}

func TestSiteWorkspacesUpdateNotOwned(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteWorkspacesUpdate(context.Background(),
		&siteapi.SiteUpdateWorkspaceInput{Name: "Hijacked"},
		siteapi.SiteWorkspacesUpdateParams{Slug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteWorkspacesUpdateNotFound{}, res)
}

func TestSiteWorkspacesUpdateRejectsBlankName(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteWorkspacesUpdate(context.Background(),
		&siteapi.SiteUpdateWorkspaceInput{Name: "   "},
		siteapi.SiteWorkspacesUpdateParams{Slug: "acme"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteWorkspacesUpdateUnprocessableEntity{}, res)
}
