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
	// PurgeWebhookJobs deletes the queued deliveries to the given webhook endpoints
	// (the Workspace's own) whose body names the Contact (contactId) or one of its
	// destinations (data.email, case-insensitively). The body is the event payload
	// the endpoint would receive. A running delivery is left to finish.
	PurgeWebhookJobs(ctx context.Context, kind string, endpointIDs []int64, contactID int64, destinations []string) error
}

func (p *txPublisher) PurgeOutbox(ctx context.Context, workspaceID, contactID int64, destinations []string) error {
	lowered := lowerAll(destinations)
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
       OR lower(payload::jsonb->'data'->>'email') = ANY($3))`, outboxTable()),
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

func (p *txPublisher) PurgeWebhookJobs(ctx context.Context, kind string, endpointIDs []int64, contactID int64, destinations []string) error {
	if len(endpointIDs) == 0 {
		return nil
	}
	id := ""
	if contactID != 0 {
		id = fmt.Sprint(contactID)
	}
	// DeliverWebhookArgs.Body is []byte, so river stores it as a base64 JSON string; it
	// is the JSON event payload, naming the Contact at contactId / data.contactId and
	// the address at data.email.
	_, err := p.tx.ExecContext(ctx, `DELETE FROM river_job
WHERE kind = $1 AND state <> 'running'
  AND (args->>'endpoint_id')::bigint = ANY($2)
  AND EXISTS (
    SELECT 1 FROM (SELECT convert_from(decode(args->>'body', 'base64'), 'UTF8')::jsonb AS body) b
    WHERE b.body->>'contactId' = $3
       OR b.body->'data'->>'contactId' = $3
       OR lower(b.body->'data'->>'email') = ANY($4))`,
		kind, endpointIDs, id, lowerAll(destinations))
	return err
}

func lowerAll(in []string) []string {
	out := make([]string, len(in))
	for i, d := range in {
		out[i] = strings.ToLower(strings.TrimSpace(d))
	}
	return out
}

var _ QueuePurger = (*txPublisher)(nil)
