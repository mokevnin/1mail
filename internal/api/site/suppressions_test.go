package site_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent/suppression"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Manual suppression CRUD: create (normalized + idempotent), list, delete.
func TestSiteSuppressionsCRUD(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()
	slug := "acme"

	// Create normalizes the destination and defaults the reason to manual.
	created, err := c.SiteSuppressionsCreate(ctx, &siteapi.SiteCreateSuppressionInput{Destination: "Blocked@Example.com"},
		siteapi.SiteSuppressionsCreateParams{Slug: slug})
	require.NoError(t, err)
	res, ok := created.(*siteapi.SiteSuppressionResource)
	require.Truef(t, ok, "got %T", created)
	assert.Equal(t, "blocked@example.com", res.Destination)
	assert.Equal(t, siteapi.SiteSuppressionReasonManual, res.Reason)

	// Suppressing the same destination again is idempotent (no duplicate).
	again, err := c.SiteSuppressionsCreate(ctx, &siteapi.SiteCreateSuppressionInput{Destination: "blocked@example.com"},
		siteapi.SiteSuppressionsCreateParams{Slug: slug})
	require.NoError(t, err)
	againRes, ok := again.(*siteapi.SiteSuppressionResource)
	require.Truef(t, ok, "got %T", again)
	assert.Equal(t, res.ID, againRes.ID, "idempotent: same entry returned")

	// The row is persisted under the workspace (selected from the DB by its key).
	exists, err := env.DB.Suppression.Query().
		Where(suppression.WorkspaceID(1), suppression.Destination("blocked@example.com")).
		Exist(ctx)
	require.NoError(t, err)
	assert.True(t, exists, "created suppression is persisted")

	// Delete removes it.
	del, err := c.SiteSuppressionsDelete(ctx, siteapi.SiteSuppressionsDeleteParams{Slug: slug, ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSuppressionsDeleteNoContent{}, del)

	stillThere, err := env.DB.Suppression.Query().
		Where(suppression.WorkspaceID(1), suppression.Destination("blocked@example.com")).
		Exist(ctx)
	require.NoError(t, err)
	assert.False(t, stillThere, "deleted suppression is gone")
}

// Suppressions are workspace-scoped: an unowned slug is a 404.
func TestSiteSuppressionsRequireOwnedWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	out, err := c.SiteSuppressionsList(context.Background(), siteapi.SiteSuppressionsListParams{Slug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSuppressionsListNotFound{}, out)
}
