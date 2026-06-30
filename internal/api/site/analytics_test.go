package site_test

import (
	"context"
	"testing"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The analytics overview requires a valid JWT cookie like the rest of the site API.
func TestSiteAnalyticsRequireAuth(t *testing.T) {
	env := testhelper.Setup(t)
	c, err := siteapi.NewClient("http://local/site", noJWT{}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)

	_, err = c.SiteAnalyticsOverview(context.Background(), siteapi.SiteAnalyticsOverviewParams{WorkspaceSlug: "acme"})
	require.Error(t, err)
}

// A slug the user does not own is a 404, not a data leak.
func TestSiteAnalyticsRequireOwnedWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	out, err := c.SiteAnalyticsOverview(context.Background(), siteapi.SiteAnalyticsOverviewParams{WorkspaceSlug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, out)
}

// seedRecipient inserts a delivery-log row with the given timestamps (nil = unset).
func seedRecipient(t *testing.T, db *ent.Client, ws, broadcastID, contactID int64, sent, opened, clicked *time.Time) {
	t.Helper()
	cr := db.BroadcastRecipient.Create().
		SetWorkspaceID(ws).
		SetBroadcastID(broadcastID).
		SetContactID(contactID).
		SetStatus(broadcastrecipient.StatusSent)
	if sent != nil {
		cr.SetSentAt(*sent)
	}
	if opened != nil {
		cr.SetOpenedAt(*opened)
	}
	if clicked != nil {
		cr.SetClickedAt(*clicked)
	}
	_, err := cr.Save(context.Background())
	require.NoError(t, err)
}

// Engagement KPIs and the time series are range-scoped from the delivery log and
// must reconcile; contacts/automations are point-in-time snapshots. Fixture
// workspace "acme" (id 1) has 3 contacts (2 active, 1 unsubscribed) created in
// January, so newInRange is 0 within any recent window.
func TestSiteAnalyticsOverview(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()
	db := env.DB
	const acme = int64(1)

	now := time.Now().UTC()
	d2 := now.AddDate(0, 0, -2)
	d1 := now.AddDate(0, 0, -1)
	d5 := now.AddDate(0, 0, -5)
	d60 := now.AddDate(0, 0, -60)   // outside 30d, inside 90d
	d100 := now.AddDate(0, 0, -100) // outside both windows

	// A broadcast with two in-range recipients, plus one stale (60d) recipient on
	// a second broadcast (same contact would collide on the unique recipient index).
	b1, err := db.Broadcast.Create().SetWorkspaceID(acme).SetName("b1").Save(ctx)
	require.NoError(t, err)
	b2, err := db.Broadcast.Create().SetWorkspaceID(acme).SetName("b2").Save(ctx)
	require.NoError(t, err)

	seedRecipient(t, db, acme, b1.ID, 1, &d2, &d2, &d1)   // sent, opened, clicked in range
	seedRecipient(t, db, acme, b1.ID, 3, &d5, nil, nil)   // sent only, in range
	seedRecipient(t, db, acme, b2.ID, 1, &d60, &d60, nil) // stale: outside 30d window

	// Sent long ago but opened recently: counting opens by open-time (not by the
	// send cohort) would inflate the open rate above 100%. It must be excluded
	// from every window it was not *sent* in.
	late, err := db.Contact.Create().SetWorkspaceID(acme).SetEmail("late@example.com").Save(ctx)
	require.NoError(t, err)
	seedRecipient(t, db, acme, b2.ID, late.ID, &d100, &d2, nil)

	// A second workspace's delivery must not leak into acme's numbers.
	other, err := db.Workspace.Create().
		SetName("Other").SetSlug("other").SetCollectKey("omck_other").SetIngestKey("omik_other").Save(ctx)
	require.NoError(t, err)
	oc, err := db.Contact.Create().SetWorkspaceID(other.ID).SetEmail("z@other.test").Save(ctx)
	require.NoError(t, err)
	ob, err := db.Broadcast.Create().SetWorkspaceID(other.ID).SetName("ob").Save(ctx)
	require.NoError(t, err)
	seedRecipient(t, db, other.ID, ob.ID, oc.ID, &d2, &d2, &d2)

	// --- 30d window ---
	res, err := c.SiteAnalyticsOverview(ctx, siteapi.SiteAnalyticsOverviewParams{
		WorkspaceSlug: "acme",
		Range:         siteapi.NewOptSiteAnalyticsRange(siteapi.SiteAnalyticsRange30d),
	})
	require.NoError(t, err)
	ov, ok := res.(*siteapi.SiteAnalyticsOverview)
	require.Truef(t, ok, "got %T", res)

	// Contacts snapshot (the other workspace is excluded). The 3 fixtures plus the
	// freshly created "late" contact give 4 total / 3 active; "late" was created
	// just now, so it is the only one inside the range.
	assert.Equal(t, int32(4), ov.Contacts.Total)
	assert.Equal(t, int32(3), ov.Contacts.Active)
	assert.Equal(t, int32(1), ov.Contacts.Unsubscribed)
	assert.Equal(t, int32(1), ov.Contacts.NewInRange)

	// Email engagement, send-cohort scoped: the stale (60d/100d) and other-workspace
	// rows drop. The 100d-sent/2d-opened row must NOT inflate the open rate.
	assert.Equal(t, int32(2), ov.Email.SentCount)
	assert.Equal(t, int32(1), ov.Email.OpenedCount)
	assert.Equal(t, int32(1), ov.Email.ClickedCount)
	assert.InDelta(t, 0.5, ov.Email.OpenRate, 0.001)
	assert.InDelta(t, 0.5, ov.Email.ClickRate, 0.001)
	assert.InDelta(t, 1.0, ov.Email.ClickToOpenRate, 0.001)
	assert.LessOrEqual(t, ov.Email.OpenRate, float32(1.0))

	// No automation fixtures.
	assert.Equal(t, int32(0), ov.Automations.Total)
	assert.Equal(t, int32(0), ov.Automations.RunsActive)

	// Time series: one zero-filled point per day, and it reconciles with the KPIs.
	assert.Len(t, ov.Timeseries, 30)
	var sumSent, sumOpened, sumClicked int32
	for _, p := range ov.Timeseries {
		sumSent += p.Sent
		sumOpened += p.Opened
		sumClicked += p.Clicked
	}
	assert.Equal(t, ov.Email.SentCount, sumSent, "series sent must reconcile with KPI")
	assert.Equal(t, ov.Email.OpenedCount, sumOpened, "series opened must reconcile with KPI")
	assert.Equal(t, ov.Email.ClickedCount, sumClicked, "series clicked must reconcile with KPI")

	// --- 90d window widens to include the stale recipient ---
	res90, err := c.SiteAnalyticsOverview(ctx, siteapi.SiteAnalyticsOverviewParams{
		WorkspaceSlug: "acme",
		Range:         siteapi.NewOptSiteAnalyticsRange(siteapi.SiteAnalyticsRange90d),
	})
	require.NoError(t, err)
	ov90 := res90.(*siteapi.SiteAnalyticsOverview)
	assert.Equal(t, int32(3), ov90.Email.SentCount)
	assert.Equal(t, int32(2), ov90.Email.OpenedCount)
	assert.Len(t, ov90.Timeseries, 90)
}
