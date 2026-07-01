package site_test

import (
	"context"
	"testing"
	"time"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixture user info@1mail.com owns workspace "acme". Broadcasts live under
// /w/{slug}/broadcasts and start life as drafts.
func TestSiteBroadcastsCRUD(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()
	slug := "acme"

	created, err := c.SiteBroadcastsCreate(ctx, &siteapi.SiteCreateBroadcastInput{
		Name:    "Spring sale",
		Subject: siteapi.NewOptString("Big news"),
		Body:    siteapi.NewOptString("<mjml><mj-body><mj-section><mj-column><mj-text>Hi {{ first_name }}</mj-text></mj-column></mj-section></mj-body></mjml>"),
	}, siteapi.SiteBroadcastsCreateParams{Slug: slug})
	require.NoError(t, err)
	res, ok := created.(*siteapi.SiteBroadcastResource)
	require.Truef(t, ok, "got %T", created)
	assert.Equal(t, "Spring sale", res.Name)
	assert.Equal(t, "Big news", res.Subject)
	assert.Equal(t, siteapi.SiteBroadcastStatusDraft, res.Status)
	assert.Equal(t, int32(0), res.Stats.RecipientsTotal)

	// Fetch the created broadcast back by id (selection by key).
	got, err := c.SiteBroadcastsGet(ctx, siteapi.SiteBroadcastsGetParams{Slug: slug, ID: res.ID})
	require.NoError(t, err)
	gotRes, ok := got.(*siteapi.SiteBroadcastResource)
	require.Truef(t, ok, "got %T", got)
	assert.Equal(t, res.ID, gotRes.ID)

	// Update renames a draft.
	updated, err := c.SiteBroadcastsUpdate(ctx, &siteapi.SiteUpdateBroadcastInput{
		Name: siteapi.NewOptString("Spring sale (v2)"),
	}, siteapi.SiteBroadcastsUpdateParams{Slug: slug, ID: res.ID})
	require.NoError(t, err)
	updRes, ok := updated.(*siteapi.SiteBroadcastResource)
	require.Truef(t, ok, "got %T", updated)
	assert.Equal(t, "Spring sale (v2)", updRes.Name)

	// Schedule moves it to scheduled with a future time.
	when := time.Now().Add(24 * time.Hour)
	sched, err := c.SiteBroadcastsSchedule(ctx, &siteapi.SiteScheduleBroadcastInput{
		ScheduledAt: siteapi.Timestamp(when),
	}, siteapi.SiteBroadcastsScheduleParams{Slug: slug, ID: res.ID})
	require.NoError(t, err)
	schedRes, ok := sched.(*siteapi.SiteBroadcastResource)
	require.Truef(t, ok, "got %T", sched)
	assert.Equal(t, siteapi.SiteBroadcastStatusScheduled, schedRes.Status)
	assert.True(t, schedRes.ScheduledAt.Set)

	// Send moves it into the sending state.
	sent, err := c.SiteBroadcastsSend(ctx, siteapi.SiteBroadcastsSendParams{Slug: slug, ID: res.ID})
	require.NoError(t, err)
	sentRes, ok := sent.(*siteapi.SiteBroadcastResource)
	require.Truef(t, ok, "got %T", sent)
	assert.Equal(t, siteapi.SiteBroadcastStatusSending, sentRes.Status)

	// Sending an already-sending broadcast is rejected with 422.
	again, err := c.SiteBroadcastsSend(ctx, siteapi.SiteBroadcastsSendParams{Slug: slug, ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteBroadcastsSendUnprocessableEntity{}, again)

	// Editing a non-draft broadcast is rejected with 422.
	badUpd, err := c.SiteBroadcastsUpdate(ctx, &siteapi.SiteUpdateBroadcastInput{
		Name: siteapi.NewOptString("nope"),
	}, siteapi.SiteBroadcastsUpdateParams{Slug: slug, ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteBroadcastsUpdateUnprocessableEntity{}, badUpd)

	// Delete removes it; a fetch by id then resolves to 404.
	del, err := c.SiteBroadcastsDelete(ctx, siteapi.SiteBroadcastsDeleteParams{Slug: slug, ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteBroadcastsDeleteNoContent{}, del)

	gone, err := c.SiteBroadcastsGet(ctx, siteapi.SiteBroadcastsGetParams{Slug: slug, ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteBroadcastsGetNotFound{}, gone)
}

// Broadcasts are scoped to the workspace: a slug the user does not own is a 404.
func TestSiteBroadcastsRequireOwnedWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	out, err := c.SiteBroadcastsList(ctx, siteapi.SiteBroadcastsListParams{Slug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteBroadcastsListNotFound{}, out)
}
