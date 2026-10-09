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
