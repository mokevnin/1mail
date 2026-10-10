package jobs_test

import (
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/jobs"
)

// recipientJobTimes returns the scheduled time of every queued recipient job, in
// scheduled order.
func (e *riverEnv) recipientJobTimes(t *testing.T) []time.Time {
	t.Helper()
	res, err := e.client.River().JobList(e.workCtx(),
		river.NewJobListParams().Kinds("send_broadcast_recipient").
			OrderBy(river.JobListOrderByScheduledAt, river.SortOrderAsc).First(1000))
	require.NoError(t, err)
	times := make([]time.Time, 0, len(res.Jobs))
	for _, j := range res.Jobs {
		times = append(times, j.ScheduledAt)
	}
	return times
}

func (e *riverEnv) planBroadcast(t *testing.T, id int64) {
	t.Helper()
	e.DB.Broadcast.UpdateOneID(id).SetStatus(broadcast.StatusSending).ExecX(e.workCtx())
	w := jobs.NewSendBroadcastWorker(e.DB, newMod(e.TestEnv, fakeResolver{sender: e.sender}))
	require.NoError(t, w.Work(e.workCtx(), job(jobs.SendBroadcastArgs{BroadcastID: id})))
}

// A Broadcast through an Integration limited to 2 per second is planned at one over the
// rate: recipient jobs are spread 500ms apart from now, and the last scheduled time is
// recorded for the ETA.
func TestPlannerSpreadsRecipientJobsAtTheEffectiveRate(t *testing.T) {
	e := newRiverEnv(t)
	ctx := e.workCtx()
	e.DB.Integration.UpdateOneID(fixtures.IntegrationAcmeDefaultID).SetMaxPerSecond(2).ExecX(ctx)

	before := time.Now()
	e.planBroadcast(t, fixtures.BroadcastDraftID)

	times := e.recipientJobTimes(t)
	require.Greater(t, len(times), 2)
	assert.WithinDuration(t, before, times[0], 5*time.Second, "starts now")
	for i := 1; i < len(times); i++ {
		assert.InDelta(t, 500*time.Millisecond, times[i].Sub(times[i-1]), float64(5*time.Millisecond), "one over the rate")
	}
	b := e.DB.Broadcast.GetX(ctx, fixtures.BroadcastDraftID)
	require.NotNil(t, b.LastScheduledAt)
	assert.WithinDuration(t, times[len(times)-1], *b.LastScheduledAt, time.Millisecond)
	assert.Equal(t, broadcast.StatusSending, b.Status, "pacing adds no status")
}

// A Broadcast scheduled for later starts pacing at its scheduled time, not at now.
func TestPlannerStartsAtTheScheduledTimeWhenLater(t *testing.T) {
	e := newRiverEnv(t)
	ctx := e.workCtx()
	e.DB.Integration.UpdateOneID(fixtures.IntegrationAcmeDefaultID).SetMaxPerDay(86400).ExecX(ctx)
	start := time.Now().Add(2 * time.Hour).Truncate(time.Microsecond)
	e.DB.Broadcast.UpdateOneID(fixtures.BroadcastDraftID).SetScheduledAt(start).ExecX(ctx)

	e.planBroadcast(t, fixtures.BroadcastDraftID)

	times := e.recipientJobTimes(t)
	require.Greater(t, len(times), 2)
	assert.WithinDuration(t, start, times[0], time.Millisecond)
	assert.InDelta(t, time.Second, times[1].Sub(times[0]), float64(5*time.Millisecond), "86400 a day is one a second")
}

// With no limit the Broadcast is planned as before: every job at once, no pacing recorded.
func TestPlannerLeavesAnUnlimitedBroadcastUnpaced(t *testing.T) {
	e := newRiverEnv(t)
	ctx := e.workCtx()

	e.planBroadcast(t, fixtures.BroadcastDraftID)

	times := e.recipientJobTimes(t)
	require.Greater(t, len(times), 2)
	assert.Less(t, times[len(times)-1].Sub(times[0]), time.Second)
	assert.Nil(t, e.DB.Broadcast.GetX(ctx, fixtures.BroadcastDraftID).LastScheduledAt)
}

// Another Workspace's limit never paces this one: the ceiling lives on the Integration.
func TestAnotherWorkspacesLimitDoesNotPaceThisBroadcast(t *testing.T) {
	e := newRiverEnv(t)
	ctx := e.workCtx()
	e.DB.Integration.UpdateOneID(fixtures.IntegrationGlobexID).SetMaxPerSecond(1).ExecX(ctx)

	times, err := jobs.PaceRecipients(ctx, e.DB, fixtures.BroadcastDraftID, 50, time.Now())
	require.NoError(t, err)
	assert.Nil(t, times)
}

// Pacing only decides when a job runs: an unsubscribe or Suppression that lands after
// planning is honoured when the job runs, and the recipient is skipped, never mailed.
func TestSuppressionDuringPacingIsHonouredAtSendTime(t *testing.T) {
	e := newRiverEnv(t)
	ctx := e.workCtx()
	e.DB.Integration.UpdateOneID(fixtures.IntegrationAcmeDefaultID).SetMaxPerSecond(1).ExecX(ctx)
	e.planBroadcast(t, fixtures.BroadcastDraftID)

	alice := e.DB.Contact.GetX(ctx, fixtures.ContactAliceID)
	e.DB.Suppression.Create().SetWorkspaceID(fixtures.AcmeID).
		SetChannel(suppression.ChannelEmail).SetDestination(*alice.Email).
		SetReason(suppression.ReasonManual).ExecX(ctx)
	rec := e.DB.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(fixtures.BroadcastDraftID), broadcastrecipient.ContactID(alice.ID)).OnlyX(ctx)

	rw := jobs.NewSendRecipientWorker(e.DB, newMod(e.TestEnv, fakeResolver{sender: e.sender}))
	require.NoError(t, rw.Work(ctx, job(jobs.SendRecipientArgs{RecipientID: rec.ID, BroadcastID: fixtures.BroadcastDraftID})))

	assert.Equal(t, broadcastrecipient.StatusSkipped, e.DB.BroadcastRecipient.GetX(ctx, rec.ID).Status)
	for _, m := range e.sender.sent {
		assert.NotEqual(t, *alice.Email, m.To)
	}
}

// Lowering the limit after a Broadcast was planned takes effect through Deferral: the
// jobs already queued snooze instead of exceeding the new ceiling.
func TestLoweringTheLimitMidSendDefersTheRemainingJobs(t *testing.T) {
	e := newRiverEnv(t)
	ctx := e.workCtx()
	e.planBroadcast(t, fixtures.BroadcastDraftID) // planned unlimited: every job due now
	e.DB.Integration.UpdateOneID(fixtures.IntegrationAcmeDefaultID).SetMaxPerSecond(1).ExecX(ctx)

	rw := jobs.NewSendRecipientWorker(e.DB, newMod(e.TestEnv, fakeResolver{sender: e.sender}))
	recs := e.DB.BroadcastRecipient.Query().Where(broadcastrecipient.BroadcastID(fixtures.BroadcastDraftID)).AllX(ctx)
	require.Greater(t, len(recs), 2)
	var deferred int
	for _, r := range recs {
		err := rw.Work(ctx, job(jobs.SendRecipientArgs{RecipientID: r.ID, BroadcastID: fixtures.BroadcastDraftID}))
		if err == nil {
			continue
		}
		var snooze *river.JobSnoozeError
		require.ErrorAs(t, err, &snooze)
		deferred++
	}
	assert.Len(t, e.sender.sent, 1, "one second's worth went out")
	assert.Equal(t, len(recs)-1, deferred)
	assert.Nil(t, e.DB.Broadcast.GetX(ctx, fixtures.BroadcastDraftID).HoldReason, "busy is not a hold")
}
