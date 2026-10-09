package consent

import (
	"context"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/confirmation"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/tracking"
)

// RecordConfirmation writes the derived Confirmation read-model row (provenance
// double_opt_in) and publishes the immutable marketing.confirmed Event in one
// transaction (ADR 0013) — the positive mirror of RecordUnsubscribe. It is keyed
// by destination, so it records even if the contact was deleted between send and
// click. The existence check makes a repeated POST (mailbox retry, double click)
// a complete no-op: the confirmation stands and no second Event is logged.
func RecordConfirmation(ctx context.Context, bus *events.Bus, target tracking.ConfirmTarget, ip string) error {
	dest := eligibility.NormalizeDestination(target.Destination)
	if dest == "" || target.WorkspaceID == 0 {
		return nil
	}

	return bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		// The Workspace comes from the signed confirmation token (no membership or api
		// token here), so this is a scope source outside the site/external/job list.
		sc := tx.Scoped(target.WorkspaceID)
		exists, err := sc.Confirmation().Query().Where(
			confirmation.ChannelEQ(confirmation.ChannelEmail),
			confirmation.DestinationEQ(dest),
		).Exist(ctx)
		if err != nil || exists {
			return err
		}
		create := sc.Confirmation().Create().
			SetChannel(confirmation.ChannelEmail).
			SetDestination(dest).
			SetProvenance(confirmation.ProvenanceDoubleOptIn)
		if target.ContactID != 0 {
			create.SetContactID(target.ContactID)
		}
		if _, err := create.Save(ctx); err != nil {
			return err
		}
		return pub.Publish(ctx, &events.MarketingConfirmed{
			WorkspaceID: target.WorkspaceID,
			ContactID:   target.ContactID,
			Email:       dest,
			Provenance:  string(confirmation.ProvenanceDoubleOptIn),
			IP:          ip,
		})
	})
}
