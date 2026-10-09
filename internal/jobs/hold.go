package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/outbound"
)

// holdRetryDelay is how long a held send waits before asking Outbound send again.
// A hold (Workspace suspension, an unverified Sending domain, no Integration) is
// reversible and not the recipient's fault, so the job is snoozed rather than
// failed: River's JobSnooze does not count against MaxAttempts, so a long hold
// never exhausts a recipient's retry budget (ADR 0015).
const holdRetryDelay = 15 * time.Minute

// inProgressRetryDelay is how soon to look again at a send whose claim another attempt
// currently holds: it is not a failure and not a hold, just a race to wait out.
const inProgressRetryDelay = 30 * time.Second

// HeldError reports that Outbound send put the source on hold (see
// outbound.Hold*). Nothing was consumed: the caller defers the same work.
type HeldError struct{ Reason string }

func (e *HeldError) Error() string { return "send held: " + e.Reason }

// asHeld returns the HeldError in err's chain, if any.
func asHeld(err error) (*HeldError, bool) {
	var h *HeldError
	return h, errors.As(err, &h)
}

// isDeferrable reports whether err means "try again later" rather than "this attempt
// failed": a hold on the source, or a live claim held by another attempt.
func isDeferrable(err error) bool {
	_, held := asHeld(err)
	return held || errors.Is(err, outbound.ErrInProgress)
}

// snoozeIfDeferrable converts a deferrable error into the River snooze for the job
// (which does not count against MaxAttempts), passing any other error through.
// Workers call it on the error from their pure function.
func snoozeIfDeferrable(err error) error {
	if _, held := asHeld(err); held {
		return river.JobSnooze(holdRetryDelay)
	}
	if errors.Is(err, outbound.ErrInProgress) {
		return river.JobSnooze(inProgressRetryDelay)
	}
	return err
}

// setBroadcastHold records (or, with reason "", clears) why a broadcast is held.
// Best effort: the hold itself is enforced by the send path, this is the visible
// state, so a failed write must not mask the real outcome.
func setBroadcastHold(ctx context.Context, s *ent.Scoped, broadcastID int64, reason string) {
	upd := s.Broadcast().UpdateOneID(broadcastID)
	if reason == "" {
		upd.ClearHoldReason()
	} else {
		upd.SetHoldReason(reason)
	}
	_ = upd.Exec(ctx)
}
