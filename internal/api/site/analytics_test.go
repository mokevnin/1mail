package site_test

import (
	"context"
	"testing"

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

	_, err = c.SiteAnalyticsOverview(context.Background(), siteapi.SiteAnalyticsOverviewParams{Slug: "acme"})
	require.Error(t, err)
}

// A slug the user does not own is a 404, not a data leak.
func TestSiteAnalyticsRequireOwnedWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	out, err := c.SiteAnalyticsOverview(context.Background(), siteapi.SiteAnalyticsOverviewParams{Slug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, out)
}

// The overview reads entirely from the seeded fixtures (broadcast_recipients with
// recent, templated sent_at). Rather than pin exact totals — which would couple
// the test to fixture volume — it asserts the invariants the dashboard relies on:
// rates are well-formed ratios, the time series reconciles with the KPIs, and the
// 90-day window is a superset of the 30-day window.
func TestSiteAnalyticsOverview(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	res, err := c.SiteAnalyticsOverview(ctx, siteapi.SiteAnalyticsOverviewParams{
		Slug:  "acme",
		Range: siteapi.NewOptSiteAnalyticsRange(siteapi.SiteAnalyticsRange30d),
	})
	require.NoError(t, err)
	ov, ok := res.(*siteapi.SiteAnalyticsOverview)
	require.Truef(t, ok, "got %T", res)

	// Contacts snapshot: the workspace has contacts, and the derived split adds up.
	assert.Greater(t, ov.Contacts.Total, int32(0))
	assert.Equal(t, ov.Contacts.Total, ov.Contacts.Active+ov.Contacts.Unsubscribed, "active + unsubscribed == total")
	assert.Greater(t, ov.Contacts.Unsubscribed, int32(0), "seeded suppressions / everything opt-outs are non-mailable")
	assert.GreaterOrEqual(t, ov.Contacts.NewInRange, int32(0))

	// Engagement KPIs are send-cohort scoped and form valid ratios.
	assert.Greater(t, ov.Email.SentCount, int32(0), "the 30d window has seeded sends")
	assert.LessOrEqual(t, ov.Email.OpenedCount, ov.Email.SentCount, "opened <= sent")
	assert.LessOrEqual(t, ov.Email.ClickedCount, ov.Email.OpenedCount, "clicked <= opened")
	assert.InDelta(t, ratioOf(ov.Email.OpenedCount, ov.Email.SentCount), ov.Email.OpenRate, 0.001)
	assert.InDelta(t, ratioOf(ov.Email.ClickedCount, ov.Email.SentCount), ov.Email.ClickRate, 0.001)
	assert.LessOrEqual(t, ov.Email.OpenRate, float32(1.0))

	// Automations snapshot reflects the seeded active automations + runs.
	assert.Greater(t, ov.Automations.Total, int32(0))
	assert.GreaterOrEqual(t, ov.Automations.RunsCompleted, int32(0))

	// Time series: one zero-filled point per day in the window, reconciling with KPIs.
	assert.Len(t, ov.Timeseries, 30)
	var sumSent, sumOpened, sumClicked int32
	for _, p := range ov.Timeseries {
		sumSent += p.Sent
		sumOpened += p.Opened
		sumClicked += p.Clicked
	}
	assert.Equal(t, ov.Email.SentCount, sumSent, "series sent reconciles with KPI")
	assert.Equal(t, ov.Email.OpenedCount, sumOpened, "series opened reconciles with KPI")
	assert.Equal(t, ov.Email.ClickedCount, sumClicked, "series clicked reconciles with KPI")

	// The 90-day window widens the cohort: it includes everything the 30-day one did.
	res90, err := c.SiteAnalyticsOverview(ctx, siteapi.SiteAnalyticsOverviewParams{
		Slug:  "acme",
		Range: siteapi.NewOptSiteAnalyticsRange(siteapi.SiteAnalyticsRange90d),
	})
	require.NoError(t, err)
	ov90 := res90.(*siteapi.SiteAnalyticsOverview)
	assert.GreaterOrEqual(t, ov90.Email.SentCount, ov.Email.SentCount, "90d sent >= 30d sent")
	assert.Len(t, ov90.Timeseries, 90)
}

func ratioOf(num, den int32) float32 {
	if den == 0 {
		return 0
	}
	return float32(num) / float32(den)
}
