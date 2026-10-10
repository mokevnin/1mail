package site_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// seedFilterLog adds three Acme entries that differ in every filterable column, on
// top of the fixture entry (user John, membership.update, 2026-01-01).
func seedFilterLog(t *testing.T, env *testhelper.TestEnv) {
	t.Helper()
	ctx := context.Background()
	s := env.DB.Scoped(fixtures.AcmeID)
	require.NoError(t, s.AuditEntry().Create().
		SetEntryKey("f-1").SetOccurredAt(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)).
		SetActorKind("api_token").SetActorID("77").SetAction("integration.update").
		SetTargetType("integration").SetTargetID("5").SetIP("10.0.0.1").SetRequestID("req-a").Exec(ctx))
	require.NoError(t, s.AuditEntry().Create().
		SetEntryKey("f-2").SetOccurredAt(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)).
		SetActorKind("system").SetAction("webhook_endpoint.create").
		SetTargetType("webhook_endpoint").SetTargetID("9").SetIP("10.0.0.2").SetRequestID("req-b").Exec(ctx))
	require.NoError(t, s.AuditEntry().Create().
		SetEntryKey("f-3").SetOccurredAt(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)).
		SetActorKind("user").SetActorID("1").SetAction("integration.delete").
		SetTargetType("integration").SetTargetID("6").SetIP("10.0.0.1").SetRequestID("req-c").Exec(ctx))
}

func actions(items []siteapi.SiteAuditEntryResource) []string {
	out := make([]string, len(items))
	for i, e := range items {
		out[i] = e.Action
	}
	return out
}

func mustExport(ctx context.Context, t *testing.T, c *siteapi.Client, p siteapi.SiteAuditExportParams) siteapi.SiteAuditExportRes {
	t.Helper()
	res, err := c.SiteAuditExport(ctx, p)
	require.NoError(t, err)
	return res
}

func TestAuditFiltersNarrowListAndExportAlike(t *testing.T) {
	env := testhelper.Setup(t)
	seedFilterLog(t, env)
	owner := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()
	day := func(m time.Month) siteapi.OptTimestamp {
		return siteapi.NewOptTimestamp(siteapi.Timestamp(time.Date(2026, m, 1, 0, 0, 0, 0, time.UTC)))
	}
	str := siteapi.NewOptString
	system := siteapi.NewOptSiteAuditActorKind(siteapi.SiteAuditActorKindSystem)

	cases := []struct {
		name   string
		list   siteapi.SiteAuditListParams
		export siteapi.SiteAuditExportParams
		want   []string
	}{
		{"actor kind",
			siteapi.SiteAuditListParams{ActorKind: system},
			siteapi.SiteAuditExportParams{ActorKind: system},
			[]string{"webhook_endpoint.create"}},
		{"actor id",
			siteapi.SiteAuditListParams{ActorId: str("77")},
			siteapi.SiteAuditExportParams{ActorId: str("77")},
			[]string{"integration.update"}},
		{"action",
			siteapi.SiteAuditListParams{Action: str("integration.delete")},
			siteapi.SiteAuditExportParams{Action: str("integration.delete")},
			[]string{"integration.delete"}},
		{"target type",
			siteapi.SiteAuditListParams{TargetType: str("integration")},
			siteapi.SiteAuditExportParams{TargetType: str("integration")},
			[]string{"integration.delete", "integration.update"}},
		{"target type and id",
			siteapi.SiteAuditListParams{TargetType: str("integration"), TargetId: str("5")},
			siteapi.SiteAuditExportParams{TargetType: str("integration"), TargetId: str("5")},
			[]string{"integration.update"}},
		{"ip",
			siteapi.SiteAuditListParams{IP: str("10.0.0.1")},
			siteapi.SiteAuditExportParams{IP: str("10.0.0.1")},
			[]string{"integration.delete", "integration.update"}},
		{"request id",
			siteapi.SiteAuditListParams{RequestId: str("req-b")},
			siteapi.SiteAuditExportParams{RequestId: str("req-b")},
			[]string{"webhook_endpoint.create"}},
		{"period",
			siteapi.SiteAuditListParams{From: day(time.February), To: day(time.April)},
			siteapi.SiteAuditExportParams{From: day(time.February), To: day(time.April)},
			[]string{"webhook_endpoint.create", "integration.update"}},
		{"filters combine",
			siteapi.SiteAuditListParams{TargetType: str("integration"), From: day(time.March)},
			siteapi.SiteAuditExportParams{TargetType: str("integration"), From: day(time.March)},
			[]string{"integration.delete"}},
		{"no match",
			siteapi.SiteAuditListParams{Action: str("nothing.here")},
			siteapi.SiteAuditExportParams{Action: str("nothing.here")},
			[]string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.list.Slug, tc.export.Slug = fixtures.AcmeSlug, fixtures.AcmeSlug
			res, err := owner.SiteAuditList(ctx, tc.list)
			require.NoError(t, err)
			page, ok := res.(*siteapi.SiteAuditEntryList)
			require.True(t, ok)
			assert.Equal(t, tc.want, actions(page.Items))

			rows := auditCSV(t, mustExport(ctx, t, owner, tc.export))
			assert.Len(t, rows, 1+len(tc.want), "the export matches the filter")
		})
	}
}

func TestAuditFilteredCursorPaginates(t *testing.T) {
	env := testhelper.Setup(t)
	seedFilterLog(t, env)
	owner := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()
	params := siteapi.SiteAuditListParams{
		Slug: fixtures.AcmeSlug, TargetType: siteapi.NewOptString("integration"), Limit: siteapi.NewOptInt32(1),
	}
	res, err := owner.SiteAuditList(ctx, params)
	require.NoError(t, err)
	p1 := res.(*siteapi.SiteAuditEntryList)
	require.Equal(t, []string{"integration.delete"}, actions(p1.Items))
	require.True(t, p1.NextCursor.IsSet())

	params.Cursor = siteapi.NewOptString(p1.NextCursor.Value)
	res, err = owner.SiteAuditList(ctx, params)
	require.NoError(t, err)
	p2 := res.(*siteapi.SiteAuditEntryList)
	assert.Equal(t, []string{"integration.update"}, actions(p2.Items))
	assert.Empty(t, p2.NextCursor.Value)
}
