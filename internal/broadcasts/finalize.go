package broadcasts

import (
	"context"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
)

// Finalize flips a Broadcast to "sent" once none of its recipients are still
// pending, deriving the aggregate counters from the recipient rows. The
// conditional WHERE status=sending makes concurrent finalizers (one per
// per-recipient job, or an Erasure that removed the last pending recipient) a no-op
// after the first, so sent_at is set exactly once and the counters self-heal
// against any retry drift.
func Finalize(ctx context.Context, s *ent.Scoped, broadcastID int64) error {
	count := func(status broadcastrecipient.Status) (int, error) {
		return s.BroadcastRecipient().Query().
			Where(broadcastrecipient.BroadcastID(broadcastID), broadcastrecipient.StatusEQ(status)).
			Count(ctx)
	}
	pending, err := count(broadcastrecipient.StatusPending)
	if err != nil {
		return err
	}
	if pending > 0 {
		return nil // not all recipients resolved yet
	}
	sent, err := count(broadcastrecipient.StatusSent)
	if err != nil {
		return err
	}
	failed, err := count(broadcastrecipient.StatusFailed)
	if err != nil {
		return err
	}
	skipped, err := count(broadcastrecipient.StatusSkipped)
	if err != nil {
		return err
	}

	_, err = s.Broadcast().Update().
		Where(broadcast.IDEQ(broadcastID), broadcast.StatusEQ(broadcast.StatusSending)).
		SetStatus(broadcast.StatusSent).
		SetSentAt(time.Now()).
		SetSentCount(sent).
		SetFailedCount(failed).
		SetSkippedCount(skipped).
		ClearHoldReason().
		Save(ctx)
	return err
}
