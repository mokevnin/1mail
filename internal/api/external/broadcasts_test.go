package external_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/broadcast"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// Fixtures (workspace acme = 1): broadcast 100 is a draft, 101 scheduled, 103 failed
// with 2 skipped, 200 sent; segment 100 is the rule segment "Pro & team members".
const (
	draftBroadcast  = "100"
	schedBroadcast  = "101"
	failedBroadcast = "103"
	sentBroadcast   = "200"
	proSegment      = "100"
)

var authorScopes = []string{"broadcasts:read", "broadcasts:write"}

func TestExternalBroadcastsAreCreatedAsDraftsAndEditedWhileDraft(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, authorScopes))
	ctx := context.Background()

	created, err := c.BroadcastsCreate(ctx, &externalapi.CreateBroadcastInput{
		Name:    "Spring sale",
		Subject: externalapi.NewOptString("Big news"),
	})
	require.NoError(t, err)
	b, ok := created.(*externalapi.BroadcastResource)
	require.Truef(t, ok, "got %T", created)
	assert.Equal(t, externalapi.BroadcastStatusDraft, b.Status)
	assert.Equal(t, "Big news", b.Subject)
	assert.False(t, b.SegmentId.IsSet(), "no audience yet")

	got, err := c.BroadcastsGet(ctx, externalapi.BroadcastsGetParams{ID: b.ID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastResource{}, got)

	updated, err := c.BroadcastsUpdate(ctx, &externalapi.UpdateBroadcastInput{Name: externalapi.NewOptString("Spring sale v2")},
		externalapi.BroadcastsUpdateParams{ID: b.ID})
	require.NoError(t, err)
	assert.Equal(t, "Spring sale v2", updated.(*externalapi.BroadcastResource).Name)

	// A scheduled broadcast is no longer editable.
	locked, err := c.BroadcastsUpdate(ctx, &externalapi.UpdateBroadcastInput{Name: externalapi.NewOptString("x")},
		externalapi.BroadcastsUpdateParams{ID: schedBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsUpdateUnprocessableEntity{}, locked)

	list, err := c.BroadcastsList(ctx, externalapi.BroadcastsListParams{})
	require.NoError(t, err)
	listed, ok := list.(*externalapi.BroadcastsListOK)
	require.Truef(t, ok, "got %T", list)
	assert.NotEmpty(t, listed.Items)
}

func TestExternalBroadcastsDeleteOnlyDrafts(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, authorScopes))
	ctx := context.Background()

	out, err := c.BroadcastsDelete(ctx, externalapi.BroadcastsDeleteParams{ID: draftBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsDeleteNoContent{}, out)

	out, err = c.BroadcastsDelete(ctx, externalapi.BroadcastsDeleteParams{ID: sentBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsDeleteUnprocessableEntity{}, out, "a sent broadcast is history")
}

func TestExternalBroadcastAudienceIsASegmentOrEveryone(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, authorScopes))
	ctx := context.Background()

	set, err := c.BroadcastsSetAudience(ctx, &externalapi.SetBroadcastAudienceInput{SegmentId: externalapi.NewNilEntityId(proSegment)},
		externalapi.BroadcastsSetAudienceParams{ID: draftBroadcast})
	require.NoError(t, err)
	b := set.(*externalapi.BroadcastResource)
	id, ok := b.SegmentId.Get()
	require.True(t, ok)
	assert.Equal(t, externalapi.EntityId(proSegment), id)
	assert.Equal(t, externalapi.BroadcastStatusDraft, b.Status, "audience never sends or schedules")

	cleared, err := c.BroadcastsSetAudience(ctx, &externalapi.SetBroadcastAudienceInput{SegmentId: externalapi.NilEntityId{Null: true}},
		externalapi.BroadcastsSetAudienceParams{ID: draftBroadcast})
	require.NoError(t, err)
	assert.False(t, cleared.(*externalapi.BroadcastResource).SegmentId.IsSet())

	unknown, err := c.BroadcastsSetAudience(ctx, &externalapi.SetBroadcastAudienceInput{SegmentId: externalapi.NewNilEntityId("999999")},
		externalapi.BroadcastsSetAudienceParams{ID: draftBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsSetAudienceUnprocessableEntity{}, unknown)

	locked, err := c.BroadcastsSetAudience(ctx, &externalapi.SetBroadcastAudienceInput{SegmentId: externalapi.NewNilEntityId(proSegment)},
		externalapi.BroadcastsSetAudienceParams{ID: schedBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsSetAudienceUnprocessableEntity{}, locked)
}

func TestExternalBroadcastTestSendGoesToOneAddressOnly(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, authorScopes))
	ctx := context.Background()

	out, err := c.BroadcastsTestSend(ctx, &externalapi.TestSendBroadcastInput{Email: "qa@test.dev"},
		externalapi.BroadcastsTestSendParams{ID: draftBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsTestSendNoContent{}, out)

	msgs := env.CustomerMail.Messages()
	require.Len(t, msgs, 1)
	assert.Equal(t, "qa@test.dev", msgs[0].To)
	assert.Equal(t, broadcast.StatusDraft, env.DB.Broadcast.GetX(ctx, 100).Status, "a test send does not touch the lifecycle")
}

func TestExternalBroadcastReport(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, authorScopes))
	ctx := context.Background()

	got, err := c.BroadcastsReport(ctx, externalapi.BroadcastsReportParams{ID: failedBroadcast})
	require.NoError(t, err)
	r, ok := got.(*externalapi.BroadcastReport)
	require.Truef(t, ok, "got %T", got)
	assert.Equal(t, externalapi.BroadcastStatusFailed, r.Status)
	assert.EqualValues(t, 2, r.SkippedCount)
	assert.EqualValues(t, 3, r.FailedCount)

	sent, err := c.BroadcastsReport(ctx, externalapi.BroadcastsReportParams{ID: sentBroadcast})
	require.NoError(t, err)
	sr := sent.(*externalapi.BroadcastReport)
	b := env.DB.Broadcast.GetX(ctx, 200)
	assert.EqualValues(t, b.SentCount, sr.SentCount)
	assert.EqualValues(t, b.OpenedCount, sr.OpenedCount)
	assert.EqualValues(t, b.ClickedCount, sr.ClickedCount)
}

func TestExternalBroadcastsRequireScopes(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	readOnly := client(t, env, seedToken(t, env.DB, []string{"broadcasts:read"}))
	noScope := client(t, env, seedToken(t, env.DB, []string{"contacts:read"}))

	// read scope reads...
	got, err := readOnly.BroadcastsGet(ctx, externalapi.BroadcastsGetParams{ID: draftBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastResource{}, got)
	rep, err := readOnly.BroadcastsReport(ctx, externalapi.BroadcastsReportParams{ID: sentBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastReport{}, rep)

	// ...but cannot write, set the audience or test-send.
	created, err := readOnly.BroadcastsCreate(ctx, &externalapi.CreateBroadcastInput{Name: "x"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsCreateUnauthorized{}, created)
	upd, err := readOnly.BroadcastsUpdate(ctx, &externalapi.UpdateBroadcastInput{}, externalapi.BroadcastsUpdateParams{ID: draftBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsUpdateUnauthorized{}, upd)
	del, err := readOnly.BroadcastsDelete(ctx, externalapi.BroadcastsDeleteParams{ID: draftBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsDeleteUnauthorized{}, del)
	aud, err := readOnly.BroadcastsSetAudience(ctx, &externalapi.SetBroadcastAudienceInput{SegmentId: externalapi.NilEntityId{Null: true}},
		externalapi.BroadcastsSetAudienceParams{ID: draftBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsSetAudienceUnauthorized{}, aud)
	ts, err := readOnly.BroadcastsTestSend(ctx, &externalapi.TestSendBroadcastInput{Email: "qa@test.dev"}, externalapi.BroadcastsTestSendParams{ID: draftBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsTestSendUnauthorized{}, ts)
	assert.Empty(t, env.CustomerMail.Messages())

	// A token without any broadcast scope reads nothing.
	list, err := noScope.BroadcastsList(ctx, externalapi.BroadcastsListParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsListUnauthorized{}, list)
	rep, err = noScope.BroadcastsReport(ctx, externalapi.BroadcastsReportParams{ID: sentBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsReportUnauthorized{}, rep)
}

func TestExternalBroadcastsAreIsolatedToTheTokensWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// A second workspace with its own broadcast and segment (incidental: the
	// fixtures hold a single workspace).
	ws2, err := env.DB.Workspace.Create().
		SetName("Globex").SetSlug("globex").SetCollectKey("globex-collect-key").SetIngestKey("globex-ingest-key").
		Save(ctx)
	require.NoError(t, err)
	foreign, err := env.DB.Broadcast.Create().SetWorkspaceID(ws2.ID).SetName("Globex only").Save(ctx)
	require.NoError(t, err)
	foreignID := externalapi.EntityId(strconv.FormatInt(foreign.ID, 10))
	foreignSeg, err := env.DB.Segment.Create().SetWorkspaceID(ws2.ID).SetName("Globex segment").Save(ctx)
	require.NoError(t, err)

	c := client(t, env, seedToken(t, env.DB, authorScopes)) // bound to workspace 1

	list, err := c.BroadcastsList(ctx, externalapi.BroadcastsListParams{})
	require.NoError(t, err)
	for _, item := range list.(*externalapi.BroadcastsListOK).Items {
		assert.NotEqual(t, foreignID, item.ID, "another tenant's broadcast leaked")
	}

	get, err := c.BroadcastsGet(ctx, externalapi.BroadcastsGetParams{ID: foreignID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsGetNotFound{}, get)
	upd, err := c.BroadcastsUpdate(ctx, &externalapi.UpdateBroadcastInput{Name: externalapi.NewOptString("hijack")}, externalapi.BroadcastsUpdateParams{ID: foreignID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsUpdateNotFound{}, upd)
	del, err := c.BroadcastsDelete(ctx, externalapi.BroadcastsDeleteParams{ID: foreignID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsDeleteNotFound{}, del)
	rep, err := c.BroadcastsReport(ctx, externalapi.BroadcastsReportParams{ID: foreignID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsReportNotFound{}, rep)
	ts, err := c.BroadcastsTestSend(ctx, &externalapi.TestSendBroadcastInput{Email: "qa@test.dev"}, externalapi.BroadcastsTestSendParams{ID: foreignID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsTestSendNotFound{}, ts)
	assert.Empty(t, env.CustomerMail.Messages())
	aud, err := c.BroadcastsSetAudience(ctx, &externalapi.SetBroadcastAudienceInput{SegmentId: externalapi.NilEntityId{Null: true}}, externalapi.BroadcastsSetAudienceParams{ID: foreignID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsSetAudienceNotFound{}, aud)

	// Another workspace's segment cannot be the audience of our broadcast.
	cross, err := c.BroadcastsSetAudience(ctx, &externalapi.SetBroadcastAudienceInput{SegmentId: externalapi.NewNilEntityId(externalapi.EntityId(strconv.FormatInt(foreignSeg.ID, 10)))},
		externalapi.BroadcastsSetAudienceParams{ID: draftBroadcast})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.BroadcastsSetAudienceUnprocessableEntity{}, cross)

	assert.Equal(t, "Globex only", env.DB.Broadcast.GetX(ctx, foreign.ID).Name, "the foreign broadcast is untouched")
}
