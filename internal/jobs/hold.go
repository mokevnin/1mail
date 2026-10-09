package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/ent"
)

// holdRetryDelay is how long a held send waits before asking Outbound send again.
// A hold (Workspace suspension, an unverified Sending domain, no Integration) is
// reversible and not the recipient's fault, so the job is snoozed rather than
// failed: River's JobSnooze does not count against MaxAttempts, so a long hold
// never exhausts a recipient's retry budget (ADR 0015).
const holdRetryDelay = 15 * time.Minute

// HeldError reports that Outbound send put the source on hold (see
// outbound.Hold*). Nothing was consumed: the caller defers the same work.
type HeldError struct{ Reason string }

func (e *HeldError) Error() string { return "send held: " + e.Reason }

// asHeld returns the HeldError in err's chain, if any.
func asHeld(err error) (*HeldError, bool) {
	var h *HeldError
	return h, errors.As(err, &h)
}

// snoozeOnHold converts a HeldError into the River snooze for the job, passing any
// other error through. Workers call it on the error from their pure function.
func snoozeOnHold(err error) error {
	if _, ok := asHeld(err); ok {
		return river.JobSnooze(holdRetryDelay)
	}
	return err
}

// setBroadcastHold records (or, with reason "", clears) why a broadcast is held.
// Best effort: the hold itself is enforced by the send path, this is the visible
// state, so a failed write must not mask the real outcome.
func setBroadcastHold(ctx context.Context, client *ent.Client, broadcastID int64, reason string) {
	upd := client.Broadcast.UpdateOneID(broadcastID)
	if reason == "" {
		upd.ClearHoldReason()
	} else {
		upd.SetHoldReason(reason)
	}
	_ = upd.Exec(ctx)
}
