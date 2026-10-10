package erasure

import (
	"context"
	"errors"

	"github.com/samber/lo"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/automationrun"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/internal/broadcasts"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/jobs"
)

// errNoQueuePurger means the transaction's Publisher cannot clear the internal
// queues: erasing without clearing them would leave personal data behind.
var errNoQueuePurger = errors.New("erasure: publisher cannot purge the internal queues")

// cancelInFlight stops work already under way for the subject and clears the internal
// queues of what refers to it: the Contact's recipients in unsent Broadcasts are
// removed (a Broadcast left with nothing pending settles), and the domain-event outbox
// rows and river job arguments naming the Contact, its runs or its recipients are
// deleted. A send a worker already handed to the provider is not recalled; one queued
// but not started finds its row gone and, at the send chokepoint, its Contact gone.
//
// It must run before the steps that delete the rows it reads ids from, and before the
// erasure publishes its own signal, which names no Contact.
func cancelInFlight(ctx context.Context, s *ent.Scoped, t *Target, pub events.Publisher) error {
	purger, ok := pub.(events.QueuePurger)
	if !ok {
		return errNoQueuePurger
	}

	runIDs, err := s.AutomationRun().Query().Where(automationrun.ContactID(t.ContactID)).IDs(ctx)
	if err != nil {
		return err
	}
	unsent, err := s.BroadcastRecipient().Query().
		Where(broadcastrecipient.ContactID(t.ContactID), broadcastrecipient.StatusEQ(broadcastrecipient.StatusPending)).
		All(ctx)
	if err != nil {
		return err
	}
	recipientIDs := lo.Map(unsent, func(r *ent.BroadcastRecipient, _ int) int64 { return r.ID })
	affected := lo.Uniq(lo.Map(unsent, func(r *ent.BroadcastRecipient, _ int) int64 { return r.BroadcastID }))

	if err := purger.PurgeOutbox(ctx, s.WorkspaceID(), t.ContactID, t.Destinations); err != nil {
		return err
	}
	if err := purger.PurgeJobs(ctx, jobs.EvaluateTriggerArgs{}.Kind(), "contact_id", []int64{t.ContactID}); err != nil {
		return err
	}
	if err := purger.PurgeJobs(ctx, jobs.RunStepArgs{}.Kind(), "run_id", runIDs); err != nil {
		return err
	}
	if err := purger.PurgeJobs(ctx, jobs.SendRecipientArgs{}.Kind(), "recipient_id", recipientIDs); err != nil {
		return err
	}

	if len(recipientIDs) == 0 {
		return nil
	}
	if _, err := s.BroadcastRecipient().Delete().Where(broadcastrecipient.IDIn(recipientIDs...)).Exec(ctx); err != nil {
		return err
	}
	for _, id := range affected {
		if err := broadcasts.Finalize(ctx, s, id); err != nil {
			return err
		}
	}
	return nil
}
