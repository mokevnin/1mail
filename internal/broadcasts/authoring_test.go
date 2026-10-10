package broadcasts_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/internal/broadcasts"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func canceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// failBroadcastReads makes every Broadcast query fail (writes are unaffected).
func failBroadcastReads(env *testhelper.TestEnv) {
	env.DB.Broadcast.Intercept(ent.InterceptFunc(func(ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(context.Context, ent.Query) (ent.Value, error) {
			return nil, errors.New("read refused")
		})
	}))
}

func TestListIsNewestFirstPagedAndWorkspaceScoped(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(&recorder{})
	ctx := context.Background()

	all, total, err := m.List(ctx, env.DB.Scoped(fixtures.AcmeID), 100, 0)
	require.NoError(t, err)
	require.Equal(t, total, len(all))
	require.Greater(t, total, 2)
	for i := 1; i < len(all); i++ {
		assert.Greater(t, all[i-1].ID, all[i].ID, "newest first")
	}

	page, pageTotal, err := m.List(ctx, env.DB.Scoped(fixtures.AcmeID), 2, 1)
	require.NoError(t, err)
	assert.Equal(t, total, pageTotal, "total ignores the page window")
	require.Len(t, page, 2)
	assert.Equal(t, all[1].ID, page[0].ID)
	assert.Equal(t, all[2].ID, page[1].ID)

	other, otherTotal, err := m.List(ctx, env.DB.Scoped(fixtures.GlobexID), 100, 0)
	require.NoError(t, err)
	assert.Equal(t, len(other), otherTotal)
	for _, b := range other {
		assert.Equal(t, int64(fixtures.GlobexID), b.WorkspaceID)
	}
}

func TestListReportsErrors(t *testing.T) {
	env := testhelper.Setup(t)
	_, _, err := broadcasts.New(&recorder{}).List(canceled(), env.DB.Scoped(fixtures.AcmeID), 10, 0)
	assert.Error(t, err)
}

func TestListReportsPageQueryError(t *testing.T) {
	env := testhelper.Setup(t)
	n := 0
	env.DB.Broadcast.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
			n++
			if n == 2 { // the count passes, the page read fails
				return nil, errors.New("read refused")
			}
			return next.Query(ctx, q)
		})
	}))
	_, _, err := broadcasts.New(&recorder{}).List(context.Background(), env.DB.Scoped(fixtures.AcmeID), 10, 0)
	assert.ErrorContains(t, err, "read refused")
}

func TestCreateReportsErrors(t *testing.T) {
	env := testhelper.Setup(t)
	_, err := broadcasts.New(&recorder{}).Create(canceled(), env.DB.Scoped(fixtures.AcmeID), broadcasts.Fields{})
	assert.Error(t, err)
}

func TestCreateMakesADraftWithTheGivenFields(t *testing.T) {
	env := testhelper.Setup(t)
	name, subject, body, from := "N", "S", "<mjml/>", "a@b.com"

	b, err := broadcasts.New(&recorder{}).Create(context.Background(), env.DB.Scoped(fixtures.AcmeID),
		broadcasts.Fields{Name: &name, Subject: &subject, Body: &body, FromEmail: &from})
	require.NoError(t, err)
	assert.Equal(t, broadcast.StatusDraft, b.Status)
	assert.Equal(t, "N", b.Name)
	assert.Equal(t, "S", b.Subject)
	assert.Equal(t, "<mjml/>", b.Body)
	require.NotNil(t, b.FromEmail)
	assert.Equal(t, "a@b.com", *b.FromEmail)
}

func TestUpdateOnlyEditsDraftsAndKeepsUnsetFields(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(&recorder{})
	ctx := context.Background()
	subject := "New subject"

	before := env.DB.Broadcast.GetX(ctx, fixtures.BroadcastDraftID)
	b, err := m.Update(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID, broadcasts.Fields{Subject: &subject})
	require.NoError(t, err)
	assert.Equal(t, "New subject", b.Subject)
	assert.Equal(t, before.Name, b.Name, "unset fields keep the stored value")

	_, err = m.Update(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastScheduledID, broadcasts.Fields{Subject: &subject})
	assert.ErrorIs(t, err, broadcasts.ErrNotDraft)
	_, err = m.Update(ctx, env.DB.Scoped(fixtures.GlobexID), fixtures.BroadcastDraftID, broadcasts.Fields{Subject: &subject})
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
	_, err = m.Update(canceled(), env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID, broadcasts.Fields{Subject: &subject})
	assert.Error(t, err)
}

func TestSetAudience(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(&recorder{})
	ctx := context.Background()
	seg := int64(fixtures.SegmentProPlanID)

	b, err := m.SetAudience(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID, &seg)
	require.NoError(t, err)
	require.NotNil(t, b.SegmentID)
	assert.Equal(t, seg, *b.SegmentID)

	b, err = m.SetAudience(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID, nil)
	require.NoError(t, err)
	assert.Nil(t, b.SegmentID, "nil audience means all active contacts")

	globexSeg := int64(fixtures.SegmentGlobexID)
	_, err = m.SetAudience(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID, &globexSeg)
	assert.ErrorIs(t, err, broadcasts.ErrSegmentNotFound, "another workspace's segment is not usable")

	_, err = m.SetAudience(ctx, env.DB.Scoped(fixtures.AcmeID), 987654, &globexSeg)
	assert.ErrorIs(t, err, broadcasts.ErrNotFound, "an unknown broadcast is reported before the segment")

	_, err = m.SetAudience(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastScheduledID, &seg)
	assert.ErrorIs(t, err, broadcasts.ErrNotDraft)

	_, err = m.SetAudience(canceled(), env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID, &seg)
	assert.Error(t, err, "segment lookup failure")
	_, err = m.SetAudience(canceled(), env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID, nil)
	assert.Error(t, err, "update failure")
}

func TestDeleteDraft(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(&recorder{})
	ctx := context.Background()

	require.NoError(t, m.DeleteDraft(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID))
	assert.False(t, env.DB.Broadcast.Query().Where(broadcast.ID(fixtures.BroadcastDraftID)).ExistX(ctx))

	assert.ErrorIs(t, m.DeleteDraft(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastScheduledID), broadcasts.ErrNotDraft)
	assert.ErrorIs(t, m.DeleteDraft(ctx, env.DB.Scoped(fixtures.AcmeID), 987654), broadcasts.ErrNotFound)
	assert.Error(t, m.DeleteDraft(canceled(), env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastEmptyAudienceID))
}

func TestReport(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(&recorder{})
	ctx := context.Background()

	env.DB.Broadcast.UpdateOneID(fixtures.BroadcastSentID).
		SetSentCount(4).SetOpenedCount(2).SetClickedCount(1).SetUnsubscribedCount(1).ExecX(ctx)
	r, err := m.Report(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastSentID)
	require.NoError(t, err)
	assert.Equal(t, broadcast.StatusSent, r.Status)
	assert.Equal(t, 4, r.Sent)
	assert.Equal(t, 2, r.Opened)
	assert.Equal(t, 1, r.Unsubscribed)
	assert.InDelta(t, 0.5, r.OpenRate, 1e-6)
	assert.InDelta(t, 0.25, r.ClickRate, 1e-6)

	zero, err := m.Report(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID)
	require.NoError(t, err)
	assert.Zero(t, zero.OpenRate, "no sends means a zero rate, not NaN")
	assert.Zero(t, zero.ClickRate)

	_, err = m.Report(ctx, env.DB.Scoped(fixtures.GlobexID), fixtures.BroadcastSentID)
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
}

func TestGetUnknownAndErrors(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(&recorder{})

	_, err := m.Get(context.Background(), env.DB.Scoped(fixtures.AcmeID), 987654)
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
	_, err = m.Get(canceled(), env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, broadcasts.ErrNotFound)
}

func TestTransitionStoreErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("claim update fails", func(t *testing.T) {
		env := testhelper.Setup(t)
		_, err := broadcasts.New(&recorder{}).Send(canceled(), env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID)
		assert.Error(t, err)
	})

	t.Run("claim existence check fails", func(t *testing.T) {
		env := testhelper.Setup(t)
		failBroadcastReads(env)
		_, err := broadcasts.New(&recorder{}).Send(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastSentID)
		assert.ErrorContains(t, err, "read refused")
	})

	t.Run("unschedule update fails", func(t *testing.T) {
		env := testhelper.Setup(t)
		_, err := broadcasts.New(&recorder{}).Unschedule(canceled(), env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastScheduledID)
		assert.Error(t, err)
	})

	t.Run("unschedule existence check fails", func(t *testing.T) {
		env := testhelper.Setup(t)
		failBroadcastReads(env)
		_, err := broadcasts.New(&recorder{}).Unschedule(ctx, env.DB.Scoped(fixtures.AcmeID), fixtures.BroadcastDraftID)
		assert.ErrorContains(t, err, "read refused")
	})
}

func ptr[T any](v T) *T { return &v }

// Another Workspace's rows are invisible through the scoped client: a foreign
// Broadcast is not found, and a foreign Segment cannot become the audience.
func TestAnotherWorkspacesBroadcastAndSegmentAreRefused(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(&recorder{})
	ctx := context.Background()
	acme := env.DB.Scoped(fixtures.AcmeID)

	_, err := m.Get(ctx, acme, fixtures.BroadcastGlobexID)
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
	_, err = m.Send(ctx, acme, fixtures.BroadcastGlobexID)
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)

	_, err = m.SetAudience(ctx, acme, fixtures.BroadcastDraftID, ptr(int64(fixtures.SegmentGlobexID)))
	assert.ErrorIs(t, err, broadcasts.ErrSegmentNotFound)
}

func TestDeleteRemovesABroadcastOfAnyStatusAndIsWorkspaceScoped(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(&recorder{})
	ctx := context.Background()
	s := env.DB.Scoped(fixtures.AcmeID)

	require.NoError(t, m.Delete(ctx, s, fixtures.BroadcastFailedID))
	_, err := m.Get(ctx, s, fixtures.BroadcastFailedID)
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)

	assert.ErrorIs(t, m.Delete(ctx, env.DB.Scoped(fixtures.GlobexID), fixtures.BroadcastDraftID), broadcasts.ErrNotFound,
		"another Workspace's Broadcast is not found")
	_, err = m.Get(ctx, s, fixtures.BroadcastDraftID)
	assert.NoError(t, err)
}
