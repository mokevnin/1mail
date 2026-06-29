package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSiteEventsListMostRecentFirst(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteEventsList(context.Background(), siteapi.SiteEventsListParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	ok, isOK := res.(*siteapi.SiteEventsListOK)
	require.Truef(t, isOK, "got %T", res)

	// Two seeded events for workspace acme, newest (purchase, 2026-01-02) first.
	assert.Equal(t, int32(2), ok.TotalItems)
	require.Len(t, ok.Items, 2)
	assert.Equal(t, "purchase", ok.Items[0].Action)
	assert.Equal(t, "page_view", ok.Items[1].Action)
}

func TestSiteEventsListFilterByAction(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteEventsList(context.Background(), siteapi.SiteEventsListParams{
		WorkspaceSlug: "acme",
		Action:        siteapi.NewOptString("purchase"),
	})
	require.NoError(t, err)
	ok, isOK := res.(*siteapi.SiteEventsListOK)
	require.Truef(t, isOK, "got %T", res)

	assert.Equal(t, int32(1), ok.TotalItems)
	require.Len(t, ok.Items, 1)
	assert.Equal(t, "purchase", ok.Items[0].Action)
}

func TestSiteEventsListFilterByEmail(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	// Upper-cased on purpose: the match must be case-insensitive because contact
	// emails are stored as entered while collect events are lowercased.
	res, err := c.SiteEventsList(context.Background(), siteapi.SiteEventsListParams{
		WorkspaceSlug: "acme",
		Email:         siteapi.NewOptString("ALICE@example.com"),
	})
	require.NoError(t, err)
	ok, isOK := res.(*siteapi.SiteEventsListOK)
	require.Truef(t, isOK, "got %T", res)

	// Only alice's page_view event matches; bob's purchase is excluded.
	assert.Equal(t, int32(1), ok.TotalItems)
	require.Len(t, ok.Items, 1)
	assert.Equal(t, "page_view", ok.Items[0].Action)
	require.True(t, ok.Items[0].Email.Set)
	assert.Equal(t, "alice@example.com", ok.Items[0].Email.Value)
}

func TestSiteEventsListUnknownWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteEventsList(context.Background(), siteapi.SiteEventsListParams{WorkspaceSlug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, res)
}
