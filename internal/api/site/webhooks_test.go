package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSiteWebhooksCRUD(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()
	slug := "acme"

	// Invalid URL is rejected.
	bad, err := c.SiteWebhooksCreate(ctx, &siteapi.SiteCreateWebhookEndpointInput{URL: "not-a-url"},
		siteapi.SiteWebhooksCreateParams{WorkspaceSlug: slug})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteWebhooksCreateUnprocessableEntity{}, bad)

	// Create with an event filter.
	created, err := c.SiteWebhooksCreate(ctx, &siteapi.SiteCreateWebhookEndpointInput{
		URL:        "https://example.com/hook",
		EventTypes: []string{"contact.created"},
	}, siteapi.SiteWebhooksCreateParams{WorkspaceSlug: slug})
	require.NoError(t, err)
	res, ok := created.(*siteapi.SiteWebhookEndpointResource)
	require.Truef(t, ok, "got %T", created)
	assert.Equal(t, "https://example.com/hook", res.URL)
	assert.Equal(t, []string{"contact.created"}, res.EventTypes)
	assert.True(t, res.Enabled)
	assert.NotEmpty(t, res.Secret, "signing secret is returned for verification")

	// Fetch it back by id (selection by key).
	got, err := c.SiteWebhooksGet(ctx, siteapi.SiteWebhooksGetParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	gotRes, ok := got.(*siteapi.SiteWebhookEndpointResource)
	require.Truef(t, ok, "got %T", got)
	assert.Equal(t, res.ID, gotRes.ID)

	// Update: disable and broaden to all events.
	upd, err := c.SiteWebhooksUpdate(ctx, &siteapi.SiteUpdateWebhookEndpointInput{
		Enabled:    siteapi.NewOptBool(false),
		EventTypes: []string{},
	}, siteapi.SiteWebhooksUpdateParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	updRes, ok := upd.(*siteapi.SiteWebhookEndpointResource)
	require.Truef(t, ok, "got %T", upd)
	assert.False(t, updRes.Enabled)
	assert.Empty(t, updRes.EventTypes)
	assert.Equal(t, res.Secret, updRes.Secret, "secret is stable across updates")

	// Delete; a fetch by id then resolves to 404.
	del, err := c.SiteWebhooksDelete(ctx, siteapi.SiteWebhooksDeleteParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteWebhooksDeleteNoContent{}, del)

	gone, err := c.SiteWebhooksGet(ctx, siteapi.SiteWebhooksGetParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteWebhooksGetNotFound{}, gone)
}
