package jobs

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/accounts"
)

// PurgeAuthAttemptsArgs is the periodic tick that removes stale failed-attempt rows
// (ADR 0018): a row whose last failure is past its window no longer throttles
// anything, and without the purge the table would grow with every address ever tried.
type PurgeAuthAttemptsArgs struct{}

func (PurgeAuthAttemptsArgs) Kind() string { return "auth_attempts_purge" }

type PurgeAuthAttemptsWorker struct {
	river.WorkerDefaults[PurgeAuthAttemptsArgs]
	ent *ent.Client
}

func (w *PurgeAuthAttemptsWorker) Work(ctx context.Context, _ *river.Job[PurgeAuthAttemptsArgs]) error {
	_, err := accounts.NewAttempts(w.ent).Purge(ctx)
	return err
}
