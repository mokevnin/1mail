package jobs_test

import (
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/jobs"
	"github.com/mokevnin/1mail/internal/outbound"
	"github.com/mokevnin/1mail/internal/tracking"
)

// A Broadcast through an Integration limited to 2 per second: recipient jobs past the
// limit snooze (no attempt consumed, no hold reason), and every recipient is still
// sent exactly once as capacity returns.
func TestBroadcastThroughALimitedIntegrationLosesNoRecipient(t *testing.T) {
	e := newRiverEnv(t)
	ctx := e.workCtx()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	mod := outbound.New(e.Bus, fakeResolver{sender: e.sender}, tracking.New("test-secret", "http://local"),
		outbound.WithClock(func() time.Time { return now }))
	e.DB.Integration.UpdateOneID(fixtures.IntegrationAcmeDefaultID).SetMaxPerSecond(2).ExecX(ctx)

	ids, err := jobs.PlanBroadcast(ctx, e.DB, mod, fixtures.BroadcastDraftID)
	require.NoError(t, err)
	require.Greater(t, len(ids), 2, "the fixture audience must outgrow one second of capacity")
	w := jobs.NewSendRecipientWorker(e.DB, mod)

	// Drive the recipient jobs the way river does: a snoozed job is not run again
	// before its wake time, so the simulated clock jumps to the earliest wake time.
	start := now
	wake := map[int64]time.Time{}
	for _, id := range ids {
		wake[id] = now
	}
	for len(wake) > 0 {
		next := now.Add(24 * time.Hour)
		for _, at := range wake {
			if at.Before(next) {
				next = at
			}
		}
		now = next
		for _, id := range ids {
			at, queued := wake[id]
			if !queued || at.After(now) {
				continue
			}
			err := w.Work(ctx, job(jobs.SendRecipientArgs{RecipientID: id, BroadcastID: fixtures.BroadcastDraftID}))
			if err == nil {
				delete(wake, id)
				continue
			}
			var snooze *river.JobSnoozeError
			require.ErrorAs(t, err, &snooze, "a Deferral is a snooze, never a failure")
			assert.GreaterOrEqual(t, snooze.Duration, time.Second, "the wait is floored")
			wake[id] = now.Add(snooze.Duration)
		}
		assert.Nil(t, e.DB.Broadcast.GetX(ctx, fixtures.BroadcastDraftID).HoldReason, "busy is not a hold")
	}

	// At 2 per second the Broadcast takes about N/2 seconds: the snoozes track the
	// limit instead of running far below it or bursting past it.
	elapsed := now.Sub(start)
	n := float64(len(ids))
	assert.GreaterOrEqual(t, elapsed.Seconds(), n/2-1, "never faster than the limit")
	assert.LessOrEqual(t, elapsed.Seconds(), n/2*1.2+2, "close to the limit, not far below it")
	assert.Len(t, e.sender.sent, len(ids), "every recipient sent exactly once")
	assert.Equal(t, broadcast.StatusSent, e.DB.Broadcast.GetX(ctx, fixtures.BroadcastDraftID).Status)
}
