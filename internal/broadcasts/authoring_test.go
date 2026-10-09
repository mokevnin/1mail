package broadcasts_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/internal/broadcasts"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func ptr[T any](v T) *T { return &v }

func TestCreateMakesADraftInTheWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})

	b, err := m.Create(context.Background(), fixtures.AcmeID, broadcasts.Fields{Name: ptr("Spring"), Subject: ptr("Hi")})
	require.NoError(t, err)

	assert.Equal(t, broadcast.StatusDraft, b.Status)
	assert.Equal(t, "Spring", b.Name)
	assert.Equal(t, "Hi", b.Subject)
	assert.Equal(t, int64(fixtures.AcmeID), b.WorkspaceID)
	assert.Nil(t, b.SegmentID, "no audience set means all active contacts")
}

func TestUpdateEditsOnlyDrafts(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})
	ctx := context.Background()
	before := env.DB.Broadcast.GetX(ctx, fixtures.BroadcastDraftID)

	b, err := m.Update(ctx, fixtures.AcmeID, fixtures.BroadcastDraftID, broadcasts.Fields{Name: ptr("Renamed")})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", b.Name)
	assert.Equal(t, before.Subject, b.Subject, "unset fields keep their value")

	_, err = m.Update(ctx, fixtures.AcmeID, fixtures.BroadcastScheduledID, broadcasts.Fields{Name: ptr("x")})
	assert.ErrorIs(t, err, broadcasts.ErrNotDraft)
	_, err = m.Update(ctx, fixtures.AcmeID, 999999, broadcasts.Fields{Name: ptr("x")})
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
}

func TestDeleteDraftRefusesAnythingPastDraft(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})
	ctx := context.Background()

	require.NoError(t, m.DeleteDraft(ctx, fixtures.AcmeID, fixtures.BroadcastDraftID))
	exists, err := env.DB.Broadcast.Query().Where(broadcast.ID(fixtures.BroadcastDraftID)).Exist(ctx)
	require.NoError(t, err)
	assert.False(t, exists)

	assert.ErrorIs(t, m.DeleteDraft(ctx, fixtures.AcmeID, fixtures.BroadcastSentID), broadcasts.ErrNotDraft)
	assert.ErrorIs(t, m.DeleteDraft(ctx, fixtures.AcmeID, 999999), broadcasts.ErrNotFound)
}

func TestSetAudiencePointsADraftAtASegmentOrClearsIt(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})
	ctx := context.Background()

	b, err := m.SetAudience(ctx, fixtures.AcmeID, fixtures.BroadcastDraftID, ptr(int64(fixtures.SegmentProPlanID)))
	require.NoError(t, err)
	require.NotNil(t, b.SegmentID)
	assert.EqualValues(t, fixtures.SegmentProPlanID, *b.SegmentID)

	b, err = m.SetAudience(ctx, fixtures.AcmeID, fixtures.BroadcastDraftID, nil)
	require.NoError(t, err)
	assert.Nil(t, b.SegmentID)
	assert.Equal(t, broadcast.StatusDraft, b.Status, "setting an audience never sends or schedules")
}

func TestSetAudienceRefusesUnknownSegmentsAndNonDrafts(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})
	ctx := context.Background()

	_, err := m.SetAudience(ctx, fixtures.AcmeID, fixtures.BroadcastDraftID, ptr(int64(999999)))
	assert.ErrorIs(t, err, broadcasts.ErrSegmentNotFound)
	_, err = m.SetAudience(ctx, fixtures.AcmeID, fixtures.BroadcastScheduledID, ptr(int64(fixtures.SegmentProPlanID)))
	assert.ErrorIs(t, err, broadcasts.ErrNotDraft)
	_, err = m.SetAudience(ctx, fixtures.AcmeID, 999999, ptr(int64(fixtures.SegmentProPlanID)))
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
}

func TestReportReadsTheDeliveryCounters(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})
	ctx := context.Background()

	r, err := m.Report(ctx, fixtures.AcmeID, fixtures.BroadcastSentID)
	require.NoError(t, err)
	b := env.DB.Broadcast.GetX(ctx, fixtures.BroadcastSentID)
	assert.Equal(t, b.SentCount, r.Sent)
	assert.Equal(t, b.OpenedCount, r.Opened)
	assert.Equal(t, b.ClickedCount, r.Clicked)
	assert.Equal(t, b.SkippedCount, r.Skipped)
	assert.InDelta(t, float32(b.OpenedCount)/float32(b.SentCount), r.OpenRate, 0.0001)

	// Broadcast BroadcastFailed (103) failed 3 recipients and skipped 2 at send time.
	r, err = m.Report(ctx, fixtures.AcmeID, fixtures.BroadcastFailedID)
	require.NoError(t, err)
	assert.Equal(t, 2, r.Skipped)
	assert.Equal(t, 3, r.Failed)

	r, err = m.Report(ctx, fixtures.AcmeID, fixtures.BroadcastDraftID)
	require.NoError(t, err)
	assert.Zero(t, r.OpenRate, "no division by zero before anything is sent")

	_, err = m.Report(ctx, fixtures.AcmeID, 999999)
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
}
