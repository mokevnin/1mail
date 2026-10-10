package events

import (
	"context"

	"github.com/mokevnin/1mail/ent"
)

// WithinScopedTx is WithinTx for a caller that holds a Workspace-scoped client: fn
// receives a scoped client over the same Workspace but bound to the transaction, so
// a module can write through it without ever naming the Workspace as an int64. The
// scope keeps its actor, and an audited write through it joins this transaction (its
// entry is published here) instead of opening another one (ADR 0022).
func (b *Bus) WithinScopedTx(ctx context.Context, s *ent.Scoped, fn func(ts *ent.Scoped, pub Publisher) error) error {
	return b.WithinTx(ctx, func(tx *ent.Client, pub Publisher) error {
		return fn(s.InTx(tx, auditPublisher{pub: pub}), pub)
	})
}

// WithinAuditTx makes the Bus the transaction opener of an audited scope
// (ent.AuditOpener): the wrapper of an audited entity runs its mutation in a
// transaction opened here and publishes the entry in it.
func (b *Bus) WithinAuditTx(ctx context.Context, s *ent.Scoped, fn func(ts *ent.Scoped) error) error {
	return b.WithinScopedTx(ctx, s, func(ts *ent.Scoped, _ Publisher) error { return fn(ts) })
}

// Act returns the scope with the actor of its writes (ADR 0022). Every construction
// point of a scope sets it; a scope without an actor, or with ActorIngest, is never
// audited.
func (b *Bus) Act(s *ent.Scoped, actor Actor) *ent.Scoped { return s.As(actor, b) }

// Ingest returns the scope of a secret-built caller (collect key, provider hook,
// signed link, bootstrap token): its writes are never audited.
func Ingest(s *ent.Scoped) *ent.Scoped { return s.As(Actor{Kind: ActorIngest}, nil) }

// auditPublisher publishes the scoped client's audit changes through the seam, in the
// transaction of the underlying Publisher.
type auditPublisher struct{ pub Publisher }

func (p auditPublisher) PublishAudit(ctx context.Context, workspaceID int64, c ent.AuditChange) error {
	return RecordAudit(ctx, p.pub, &AuditEntry{
		WorkspaceID: workspaceID,
		Actor:       c.Actor,
		Action:      c.Action,
		TargetType:  c.TargetType,
		TargetID:    c.TargetID,
		TargetName:  c.TargetName,
		Diff:        c.Diff,
	})
}
