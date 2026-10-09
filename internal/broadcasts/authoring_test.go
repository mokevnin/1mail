package broadcasts_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/internal/broadcasts"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// Fixtures: broadcast 200 is sent; segment 100 is the rule segment "Pro & team members".
const (
	sentID       = int64(200)
	proSegmentID = int64(100)
)

func ptr[T any](v T) *T { return &v }

func TestCreateMakesADraftInTheWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})

	b, err := m.Create(context.Background(), acme, broadcasts.Fields{Name: ptr("Spring"), Subject: ptr("Hi")})
	require.NoError(t, err)

	assert.Equal(t, broadcast.StatusDraft, b.Status)
	assert.Equal(t, "Spring", b.Name)
	assert.Equal(t, "Hi", b.Subject)
	assert.Equal(t, acme, b.WorkspaceID)
	assert.Nil(t, b.SegmentID, "no audience set means all active contacts")
}

func TestUpdateEditsOnlyDrafts(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})
	ctx := context.Background()
	before := env.DB.Broadcast.GetX(ctx, draftID)

	b, err := m.Update(ctx, acme, draftID, broadcasts.Fields{Name: ptr("Renamed")})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", b.Name)
	assert.Equal(t, before.Subject, b.Subject, "unset fields keep their value")

	_, err = m.Update(ctx, acme, schedID, broadcasts.Fields{Name: ptr("x")})
	assert.ErrorIs(t, err, broadcasts.ErrNotDraft)
	_, err = m.Update(ctx, acme, 999999, broadcasts.Fields{Name: ptr("x")})
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
}

func TestDeleteDraftRefusesAnythingPastDraft(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})
	ctx := context.Background()

	require.NoError(t, m.DeleteDraft(ctx, acme, draftID))
	exists, err := env.DB.Broadcast.Query().Where(broadcast.ID(draftID)).Exist(ctx)
	require.NoError(t, err)
	assert.False(t, exists)

	assert.ErrorIs(t, m.DeleteDraft(ctx, acme, sentID), broadcasts.ErrNotDraft)
	assert.ErrorIs(t, m.DeleteDraft(ctx, acme, 999999), broadcasts.ErrNotFound)
}

func TestSetAudiencePointsADraftAtASegmentOrClearsIt(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})
	ctx := context.Background()

	b, err := m.SetAudience(ctx, acme, draftID, ptr(proSegmentID))
	require.NoError(t, err)
	require.NotNil(t, b.SegmentID)
	assert.Equal(t, proSegmentID, *b.SegmentID)

	b, err = m.SetAudience(ctx, acme, draftID, nil)
	require.NoError(t, err)
	assert.Nil(t, b.SegmentID)
	assert.Equal(t, broadcast.StatusDraft, b.Status, "setting an audience never sends or schedules")
}

func TestSetAudienceRefusesUnknownSegmentsAndNonDrafts(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})
	ctx := context.Background()

	_, err := m.SetAudience(ctx, acme, draftID, ptr(int64(999999)))
	assert.ErrorIs(t, err, broadcasts.ErrSegmentNotFound)
	_, err = m.SetAudience(ctx, acme, schedID, ptr(proSegmentID))
	assert.ErrorIs(t, err, broadcasts.ErrNotDraft)
	_, err = m.SetAudience(ctx, acme, 999999, ptr(proSegmentID))
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
}

func TestReportReadsTheDeliveryCounters(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})
	ctx := context.Background()

	r, err := m.Report(ctx, acme, sentID)
	require.NoError(t, err)
	b := env.DB.Broadcast.GetX(ctx, sentID)
	assert.Equal(t, b.SentCount, r.Sent)
	assert.Equal(t, b.OpenedCount, r.Opened)
	assert.Equal(t, b.ClickedCount, r.Clicked)
	assert.Equal(t, b.SkippedCount, r.Skipped)
	assert.InDelta(t, float32(b.OpenedCount)/float32(b.SentCount), r.OpenRate, 0.0001)

	// Broadcast 103 failed 3 recipients and skipped 2 at send time.
	r, err = m.Report(ctx, acme, 103)
	require.NoError(t, err)
	assert.Equal(t, 2, r.Skipped)
	assert.Equal(t, 3, r.Failed)

	r, err = m.Report(ctx, acme, draftID)
	require.NoError(t, err)
	assert.Zero(t, r.OpenRate, "no division by zero before anything is sent")

	_, err = m.Report(ctx, acme, 999999)
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
}
