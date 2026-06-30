// Package eligibility derives whether a message on a channel from a sending
// source may reach a destination (ADR 0001). Eligibility is never a stored flag
// on the Contact; it is computed in layers against two destination-keyed stores:
//
//  1. (channel, destination) in Suppression          → never (global hard floor)
//  2. (channel, destination) unsubscribed "everything" → never
//  3. (channel, destination) unsubscribed from source  → never
//  4. otherwise                                        → send
//
// Transactional sends pass respectUnsubscribe=false: they skip layers 2–3 (you
// cannot opt out of your own password reset) but still respect Suppression.
//
// Stored destinations are normalized (lower-cased + trimmed) on write, but
// contact.email is not normalized at write time, so every comparison folds case
// on the contact side. The batch Predicate does this with lower() in SQL; Check
// normalizes the input before the point lookups.
package eligibility

import (
	"context"
	"entgo.io/ent/dialect/sql"
	"fmt"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/predicate"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/ent/unsubscribe"
)

// Reasons a destination is ineligible.
const (
	ReasonSuppressed             = "suppressed"
	ReasonUnsubscribedEverything = "unsubscribed_everything"
	ReasonUnsubscribedSource     = "unsubscribed_source"
)

// Decision is the outcome of a single eligibility Check.
type Decision struct {
	Eligible bool
	// Reason is empty when Eligible; otherwise one of the Reason* constants.
	Reason string
}

// Predicate ANDs the eligibility floor for (channel, source) onto a Contact
// query: not suppressed AND not unsubscribed from "everything" AND not
// unsubscribed from the source. It composes with a segment predicate (both are
// predicate.Contact). The correlation joins on lower(contacts.email) =
// destination — never on contact_id — because the stores are destination-keyed.
func Predicate(channel, source string) predicate.Contact {
	return func(s *sql.Selector) {
		// Layer 1: Suppression (global hard floor).
		sup := sql.Select(suppression.FieldID).From(sql.Table(suppression.Table))
		sup.Where(sql.And(
			destinationMatches(s, sup, suppression.FieldDestination),
			sql.ColumnsEQ(sup.C(suppression.FieldWorkspaceID), s.C(contact.FieldWorkspaceID)),
			sql.EQ(sup.C(suppression.FieldChannel), channel),
		))
		s.Where(sql.NotExists(sup))

		// Layers 2–3: Unsubscribe from "everything" or the source.
		uns := sql.Select(unsubscribe.FieldID).From(sql.Table(unsubscribe.Table))
		uns.Where(sql.And(
			destinationMatches(s, uns, unsubscribe.FieldDestination),
			sql.ColumnsEQ(uns.C(unsubscribe.FieldWorkspaceID), s.C(contact.FieldWorkspaceID)),
			sql.EQ(uns.C(unsubscribe.FieldChannel), channel),
			sql.In(uns.C(unsubscribe.FieldSendingSource), SourceEverything, source),
		))
		s.Where(sql.NotExists(uns))
	}
}

// GloballyOptedOut matches contacts whose destination is non-mailable on the
// given channel regardless of source: it is suppressed, or unsubscribed from
// "everything". Used for the "unsubscribed" analytics KPI — the derived,
// dashboard-level view of "fully opted out", since there is no stored contact
// status (ADR 0001). A per-source ("broadcasts") opt-out is deliberately not
// counted here: that contact may still be mailable from other sources.
func GloballyOptedOut(channel string) predicate.Contact {
	return func(s *sql.Selector) {
		sup := sql.Select(suppression.FieldID).From(sql.Table(suppression.Table))
		sup.Where(sql.And(
			destinationMatches(s, sup, suppression.FieldDestination),
			sql.ColumnsEQ(sup.C(suppression.FieldWorkspaceID), s.C(contact.FieldWorkspaceID)),
			sql.EQ(sup.C(suppression.FieldChannel), channel),
		))
		uns := sql.Select(unsubscribe.FieldID).From(sql.Table(unsubscribe.Table))
		uns.Where(sql.And(
			destinationMatches(s, uns, unsubscribe.FieldDestination),
			sql.ColumnsEQ(uns.C(unsubscribe.FieldWorkspaceID), s.C(contact.FieldWorkspaceID)),
			sql.EQ(uns.C(unsubscribe.FieldChannel), channel),
			sql.EQ(uns.C(unsubscribe.FieldSendingSource), SourceEverything),
		))
		s.Where(sql.Or(sql.Exists(sup), sql.Exists(uns)))
	}
}

// destinationMatches correlates the subquery's normalized destination to the
// outer contact's email, folding case on the (un-normalized) contact side. It
// defers column rendering to query-build time (sql.P closure) so the dialect is
// applied — eager string formatting would emit mis-quoted identifiers.
func destinationMatches(outer, sub *sql.Selector, destCol string) *sql.Predicate {
	return sql.P(func(b *sql.Builder) {
		b.WriteString("lower(").WriteString(outer.C(contact.FieldEmail)).WriteString(") = ").WriteString(sub.C(destCol))
	})
}

// CheckTransactional is the eligibility check for a transactional send: it
// respects Suppression (the global hard floor) but skips Unsubscribe, because
// transactional mail carries no sending source (ADR 0005). A thin wrapper over
// Check so the call site reads as intent rather than a bare "", false.
func CheckTransactional(ctx context.Context, client *ent.Client, workspaceID int64, channel, dest string) (Decision, error) {
	return Check(ctx, client, workspaceID, channel, dest, "", false)
}

// Check decides eligibility for a single destination via point lookups. Pass
// respectUnsubscribe=false for transactional sends (Suppression only). An empty
// destination is treated as eligible — the caller decides what a missing
// address means (there is nothing to suppress against).
func Check(ctx context.Context, client *ent.Client, workspaceID int64, channel, dest, source string, respectUnsubscribe bool) (Decision, error) {
	d := NormalizeDestination(dest)
	if d == "" {
		return Decision{Eligible: true}, nil
	}

	suppressed, err := client.Suppression.Query().
		Where(
			suppression.WorkspaceID(workspaceID),
			suppression.ChannelEQ(suppression.Channel(channel)),
			suppression.DestinationEQ(d),
		).
		Exist(ctx)
	if err != nil {
		return Decision{}, fmt.Errorf("eligibility suppression lookup: %w", err)
	}
	if suppressed {
		return Decision{Eligible: false, Reason: ReasonSuppressed}, nil
	}

	if !respectUnsubscribe {
		return Decision{Eligible: true}, nil
	}

	everything, err := client.Unsubscribe.Query().
		Where(
			unsubscribe.WorkspaceID(workspaceID),
			unsubscribe.ChannelEQ(unsubscribe.Channel(channel)),
			unsubscribe.DestinationEQ(d),
			unsubscribe.SendingSourceEQ(SourceEverything),
		).
		Exist(ctx)
	if err != nil {
		return Decision{}, fmt.Errorf("eligibility unsubscribe(everything) lookup: %w", err)
	}
	if everything {
		return Decision{Eligible: false, Reason: ReasonUnsubscribedEverything}, nil
	}

	fromSource, err := client.Unsubscribe.Query().
		Where(
			unsubscribe.WorkspaceID(workspaceID),
			unsubscribe.ChannelEQ(unsubscribe.Channel(channel)),
			unsubscribe.DestinationEQ(d),
			unsubscribe.SendingSourceEQ(source),
		).
		Exist(ctx)
	if err != nil {
		return Decision{}, fmt.Errorf("eligibility unsubscribe(source) lookup: %w", err)
	}
	if fromSource {
		return Decision{Eligible: false, Reason: ReasonUnsubscribedSource}, nil
	}

	return Decision{Eligible: true}, nil
}
