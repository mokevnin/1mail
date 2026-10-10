package events

import (
	"context"
	"fmt"
	"strings"
)

// QueuePurger removes what the internal queues still hold about a data subject: the
// domain-event outbox and the river job table. The Publisher handed to
// Bus.WithinTx implements it over the same transaction, so a purge commits (or
// rolls back) with the state change that asks for it. Erasure is its only caller
// (ADR 0021); it is a separate interface so ordinary producers cannot reach it.
type QueuePurger interface {
	// PurgeOutbox deletes the Workspace's outbox envelopes that name the Contact,
	// by id or by one of its destinations (matched case-insensitively).
	PurgeOutbox(ctx context.Context, workspaceID, contactID int64, destinations []string) error
	// PurgeJobs deletes the river jobs of kind that are not running and whose
	// args integer field key is one of ids. A running job is left to finish: its
	// worker re-checks that what it refers to still exists.
	PurgeJobs(ctx context.Context, kind, key string, ids []int64) error
}

// outboxTable is the watermill-sql table of the domain-events topic.
const outboxTable = "watermill_" + TopicDomainEvents

func (p *txPublisher) PurgeOutbox(ctx context.Context, workspaceID, contactID int64, destinations []string) error {
	lowered := make([]string, len(destinations))
	for i, d := range destinations {
		lowered[i] = strings.ToLower(strings.TrimSpace(d))
	}
	// contactID 0 is "no Contact" (an address erased on its own); events with no
	// Contact carry contactId 0 or none, so 0 must never match.
	id := ""
	if contactID != 0 {
		id = fmt.Sprint(contactID)
	}
	// The payload is the JSON-encoded Envelope; events name the Contact as data.contactId
	// and the address as data.email.
	_, err := p.tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s
WHERE (payload::jsonb->>'workspaceId')::bigint = $1
  AND ((payload::jsonb->'data'->>'contactId') = $2
       OR lower(payload::jsonb->'data'->>'email') = ANY($3))`, outboxTable),
		workspaceID, id, lowered)
	return err
}

func (p *txPublisher) PurgeJobs(ctx context.Context, kind, key string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := p.tx.ExecContext(ctx, `DELETE FROM river_job
WHERE kind = $1 AND state <> 'running' AND (args->>$2)::bigint = ANY($3)`,
		kind, key, ids)
	return err
}

var _ QueuePurger = (*txPublisher)(nil)
