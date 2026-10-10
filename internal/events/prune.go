package events

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// outboxTable / offsetsTable are the quoted names of the domain-events outbox and
// its consumer offsets, derived from the same adapters the producer and the
// subscribers use.
func outboxTable() string  { return outboxSchema.MessagesTable(TopicDomainEvents) }
func offsetsTable() string { return outboxOffsets.MessagesOffsetsTable(TopicDomainEvents) }

// PruneResult reports one PruneOutbox run.
type PruneResult struct {
	// Deleted is the number of outbox rows removed.
	Deleted int64
	// RetiredGroups is the number of offsets rows removed for consumer groups no
	// longer registered in code.
	RetiredGroups int64
	// Skipped is true when the run deleted no outbox rows because a registered
	// consumer group has no usable cursor yet (no offsets row, or a NULL ack).
	Skipped bool
}

// PruneOutbox deletes outbox rows in batches of batchSize that sort strictly
// below the lowest registered consumer group's cursor, by the same
// (transaction_id, offset) tuple the subscribers read in, and are older than
// floor (ADR 0019). The row at the cursor is kept so an at-least-once redelivery
// still finds its place; the events table is the durable record, and its unique
// source_id dedupes a redelivered message.
//
// It first drops the offsets rows of groups that are no longer registered in
// code (ConsumerGroups): a retired group's frozen cursor would otherwise pin the
// outbox forever. It then skips the run when any registered group has no offsets
// row or a NULL ack, because that group has not consumed anything and may still
// need every row.
func PruneOutbox(ctx context.Context, db *sql.DB, floor time.Duration, batchSize int) (PruneResult, error) {
	var res PruneResult
	groups := ConsumerGroups()
	holders := make([]string, len(groups))
	args := make([]any, len(groups))
	for i, g := range groups {
		holders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = g
	}
	registered := strings.Join(holders, ", ")

	retired, err := db.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE consumer_group NOT IN (%s)`, offsetsTable(), registered), args...)
	if err != nil {
		return res, fmt.Errorf("remove retired consumer groups: %w", err)
	}
	if res.RetiredGroups, err = retired.RowsAffected(); err != nil {
		return res, err
	}

	// The registered groups' cursors, lowest first: the first row bounds the prune.
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT last_processed_transaction_id::text, offset_acked FROM %s
		WHERE consumer_group IN (%s)
		ORDER BY last_processed_transaction_id, offset_acked NULLS FIRST`, offsetsTable(), registered), args...)
	if err != nil {
		return res, fmt.Errorf("read consumer cursors: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var (
		cursors      int
		cursorTx     string
		cursorOffset int64
		incomplete   bool
	)
	for rows.Next() {
		var tx string
		var acked sql.NullInt64
		if err := rows.Scan(&tx, &acked); err != nil {
			return res, err
		}
		if !acked.Valid {
			incomplete = true
		}
		if cursors == 0 {
			cursorTx, cursorOffset = tx, acked.Int64
		}
		cursors++
	}
	if err := rows.Err(); err != nil {
		return res, err
	}
	_ = rows.Close()
	if incomplete || cursors != len(groups) {
		res.Skipped = true
		return res, nil
	}

	del := fmt.Sprintf(`
		DELETE FROM %[1]s WHERE (transaction_id, "offset") IN (
			SELECT transaction_id, "offset" FROM %[1]s
			WHERE (transaction_id, "offset") < ($1::xid8, $2::int)
			  AND created_at < now() - make_interval(secs => $3)
			ORDER BY transaction_id, "offset"
			LIMIT $4)`, outboxTable())
	for {
		r, err := db.ExecContext(ctx, del, cursorTx, cursorOffset, floor.Seconds(), batchSize)
		if err != nil {
			return res, fmt.Errorf("delete outbox batch: %w", err)
		}
		n, err := r.RowsAffected()
		if err != nil {
			return res, err
		}
		res.Deleted += n
		if n < int64(batchSize) {
			return res, nil
		}
	}
}
