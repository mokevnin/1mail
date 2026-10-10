package events

import (
	"context"

	"github.com/mokevnin/1mail/ent"
)

// WithinScopedTx is WithinTx for a caller that holds a Workspace-scoped client: fn
// receives a scoped client over the same Workspace but bound to the transaction, so
// a module can write through it without ever naming the Workspace as an int64.
func (b *Bus) WithinScopedTx(ctx context.Context, s *ent.Scoped, fn func(ts *ent.Scoped, pub Publisher) error) error {
	return b.WithinTx(ctx, func(tx *ent.Client, pub Publisher) error {
		return fn(tx.Scoped(s.WorkspaceID()), pub)
	})
}

// PurgingPublisher is the Publisher of a transaction that may also clear the internal
// queues (see QueuePurger). Only WithinScopedPurgeTx hands one out.
type PurgingPublisher interface {
	Publisher
	QueuePurger
}

// WithinScopedPurgeTx is WithinScopedTx for the one caller that clears the internal
// queues in the same transaction: Erasure (ADR 0021). Asking for the purging
// publisher here, not by type assertion on a Publisher, makes it a compile-time
// dependency.
func (b *Bus) WithinScopedPurgeTx(ctx context.Context, s *ent.Scoped, fn func(ts *ent.Scoped, pub PurgingPublisher) error) error {
	return b.within(ctx, func(tx *ent.Client, pub *txPublisher) error {
		return fn(tx.Scoped(s.WorkspaceID()), pub)
	})
}
