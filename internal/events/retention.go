package events

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// analyticalPredicate selects the Events that expire by age (ADR 0019): every
// Event except the evidentiary ones (consent proof, complaints, unsubscribes and
// permanent bounces). A bounce with no bounceKind counts as transient, like
// suppressionReason does. The partial index events_created_at_analytical_idx
// (ent/schema/event.go) carries the same expression, so the batched delete walks
// only expirable rows; keep the two in step.
const analyticalPredicate = `action NOT IN ('marketing.confirmed', 'email.complained', 'email.unsubscribed') AND NOT (action = 'email.bounced' AND COALESCE(properties->>'bounceKind', '') = 'permanent')`

// PruneEvents deletes analytical Events created more than retention ago, in
// batches of batchSize, and returns how many it removed. Age is created_at (the
// insert time), not occurred_at: created_at is never null and is what the table's
// growth follows; an Event backfilled with an old occurred_at is kept until its
// row itself ages out. A retention of zero or less disables the prune. The job is
// idempotent: a rerun finds nothing left to delete.
func PruneEvents(ctx context.Context, db *sql.DB, retention time.Duration, batchSize int) (int64, error) {
	if retention <= 0 {
		return 0, nil
	}
	del := fmt.Sprintf(`
		DELETE FROM events WHERE id IN (
			SELECT id FROM events
			WHERE created_at < now() - make_interval(secs => $1) AND %s
			ORDER BY created_at
			LIMIT $2)`, analyticalPredicate)
	var total int64
	for {
		r, err := db.ExecContext(ctx, del, retention.Seconds(), batchSize)
		if err != nil {
			return total, fmt.Errorf("delete expired events batch: %w", err)
		}
		n, err := r.RowsAffected()
		if err != nil {
			return total, err
		}
		total += n
		if n < int64(batchSize) {
			return total, nil
		}
	}
}
