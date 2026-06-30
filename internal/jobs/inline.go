package jobs

import (
	"context"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/tracking"
)

// Inline is the synchronous adapter of the enqueue seam (the Rails `:inline`
// equivalent): instead of inserting into river, it runs each job's pure function
// immediately. It satisfies the same narrow enqueue interfaces the river Client
// does (BroadcastEnqueuer, WelcomeEnqueuer), so the test harness (and a future
// single-process mode) exercise the enqueue→execute path for real rather than
// asserting "a job was enqueued".
type Inline struct {
	ent          *ent.Client
	bus          *events.Bus
	resolver     SenderResolver
	tracker      *tracking.Tracker
	systemSender messaging.EmailSender
}

// NewInline builds the inline adapter. systemSender sends platform mail; resolver
// resolves a workspace's customer sender for broadcasts; bus carries the send's
// transactional outbox publish (email.sent).
func NewInline(entClient *ent.Client, bus *events.Bus, resolver SenderResolver, tracker *tracking.Tracker, systemSender messaging.EmailSender) *Inline {
	return &Inline{ent: entClient, bus: bus, resolver: resolver, tracker: tracker, systemSender: systemSender}
}

// EnqueueBroadcast runs the broadcast send now. A future scheduledAt is skipped:
// the inline adapter has no scheduler, and running a future job now would be
// wrong (e.g. it must not fire a broadcast scheduled for tomorrow).
func (i *Inline) EnqueueBroadcast(ctx context.Context, broadcastID int64, scheduledAt *time.Time) error {
	if scheduledAt != nil && scheduledAt.After(time.Now()) {
		return nil
	}
	return SendBroadcast(ctx, i.ent, i.bus, i.resolver, i.tracker, broadcastID)
}

// EnqueueWelcome sends the welcome email now via the system sender.
func (i *Inline) EnqueueWelcome(ctx context.Context, email, name string) error {
	return SendWelcome(ctx, i.systemSender, email, name)
}
