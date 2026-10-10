package external_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func TestExternalAuditListNeedsTheAuditReadScope(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	for _, scopes := range [][]string{{}, {"contacts:read", "events:read", "tokens:read"}} {
		c := env.ExternalScoped(t, scopes...)
		res, err := c.AuditEntriesList(ctx, externalapi.AuditEntriesListParams{})
		require.NoError(t, err)
		assert.IsType(t, &externalapi.AuditEntriesListUnauthorized{}, res, "scopes %v are refused", scopes)
	}

	res, err := env.ExternalScoped(t, "audit:read").AuditEntriesList(ctx, externalapi.AuditEntriesListParams{})
	require.NoError(t, err)
	page, ok := res.(*externalapi.AuditEntryList)
	require.Truef(t, ok, "got %T", res)
	require.Len(t, page.Items, 1, "only the Workspace's own fixture entry")
	e := page.Items[0]
	assert.Equal(t, strconv.Itoa(fixtures.AcmeAuditEntryID), string(e.ID))
	assert.Equal(t, fixtures.AcmeAuditEntryAction, e.Action)
	assert.Equal(t, externalapi.AuditActorKindUser, e.Actor.Kind)
	diff, err := json.Marshal(e.Diff.Value)
	require.NoError(t, err)
	assert.JSONEq(t, `{"role":{"from":"admin","to":"member"}}`, string(diff))
}

func TestExternalAuditListIsTenantIsolated(t *testing.T) {
	env := testhelper.Setup(t)
	globex := env.ExternalWithToken(t, env.ScopedBearerFor(t, fixtures.GlobexID, "audit:read"))
	res, err := globex.AuditEntriesList(context.Background(), externalapi.AuditEntriesListParams{})
	require.NoError(t, err)
	page := res.(*externalapi.AuditEntryList)
	require.Len(t, page.Items, 1)
	assert.Equal(t, strconv.Itoa(fixtures.GlobexAuditEntryID), string(page.Items[0].ID))
}

func TestExternalAuditListPaginatesByCursor(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	s := env.DB.Scoped(fixtures.AcmeID)
	for i := range 2 {
		require.NoError(t, s.AuditEntry().Create().
			SetEntryKey("extra-"+strconv.Itoa(i)).SetOccurredAt(time.Now()).
			SetActorKind("system").SetAction("tag.create").SetTargetType("tag").
			Exec(ctx))
	}
	c := env.ExternalScoped(t, "audit:read")

	res, err := c.AuditEntriesList(ctx, externalapi.AuditEntriesListParams{Limit: externalapi.NewOptInt32(2)})
	require.NoError(t, err)
	p1 := res.(*externalapi.AuditEntryList)
	require.Len(t, p1.Items, 2)
	require.NotEmpty(t, p1.NextCursor.Value)

	res, err = c.AuditEntriesList(ctx, externalapi.AuditEntriesListParams{
		Limit: externalapi.NewOptInt32(2), Cursor: externalapi.NewOptString(p1.NextCursor.Value),
	})
	require.NoError(t, err)
	p2 := res.(*externalapi.AuditEntryList)
	require.Len(t, p2.Items, 1)
	assert.Empty(t, p2.NextCursor.Value, "last page")
	assert.Equal(t, strconv.Itoa(fixtures.AcmeAuditEntryID), string(p2.Items[0].ID))
}

func TestExternalAuditListWithoutLicenseIsPaymentRequired(t *testing.T) {
	env := testhelper.Setup(t, testhelper.WithoutLicense())
	res, err := env.ExternalScoped(t, "audit:read").AuditEntriesList(context.Background(), externalapi.AuditEntriesListParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.AuditEntriesListPaymentRequired{}, res)
}

func TestExternalAuditListRejectsABadCursor(t *testing.T) {
	env := testhelper.Setup(t)
	res, err := env.ExternalScoped(t, "audit:read").AuditEntriesList(context.Background(),
		externalapi.AuditEntriesListParams{Cursor: externalapi.NewOptString("nope")})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.AuditEntriesListBadRequest{}, res)
}
