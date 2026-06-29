package site_test

import (
	"context"
	"testing"

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

	// Create normalizes the address and defaults the reason to manual.
	created, err := c.SiteSuppressionsCreate(ctx, &siteapi.SiteCreateSuppressionInput{Email: "Blocked@Example.com"},
		siteapi.SiteSuppressionsCreateParams{WorkspaceSlug: slug})
	require.NoError(t, err)
	res, ok := created.(*siteapi.SiteSuppressionResource)
	require.Truef(t, ok, "got %T", created)
	assert.Equal(t, "blocked@example.com", res.Email)
	assert.Equal(t, siteapi.SiteSuppressionReasonManual, res.Reason)

	// Suppressing the same address again is idempotent (no duplicate).
	again, err := c.SiteSuppressionsCreate(ctx, &siteapi.SiteCreateSuppressionInput{Email: "blocked@example.com"},
		siteapi.SiteSuppressionsCreateParams{WorkspaceSlug: slug})
	require.NoError(t, err)
	againRes, ok := again.(*siteapi.SiteSuppressionResource)
	require.Truef(t, ok, "got %T", again)
	assert.Equal(t, res.ID, againRes.ID, "idempotent: same entry returned")

	list, err := c.SiteSuppressionsList(ctx, siteapi.SiteSuppressionsListParams{WorkspaceSlug: slug})
	require.NoError(t, err)
	listed, ok := list.(*siteapi.SiteSuppressionsListOK)
	require.Truef(t, ok, "got %T", list)
	assert.Equal(t, int32(1), listed.TotalItems)

	// Delete removes it.
	del, err := c.SiteSuppressionsDelete(ctx, siteapi.SiteSuppressionsDeleteParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSuppressionsDeleteNoContent{}, del)

	after, err := c.SiteSuppressionsList(ctx, siteapi.SiteSuppressionsListParams{WorkspaceSlug: slug})
	require.NoError(t, err)
	assert.Equal(t, int32(0), after.(*siteapi.SiteSuppressionsListOK).TotalItems)
}

// Suppressions are workspace-scoped: an unowned slug is a 404.
func TestSiteSuppressionsRequireOwnedWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	out, err := c.SiteSuppressionsList(context.Background(), siteapi.SiteSuppressionsListParams{WorkspaceSlug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSuppressionsListNotFound{}, out)
}
