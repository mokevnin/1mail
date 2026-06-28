// Package jobs is the river-backed (Postgres) async job queue. It owns the
// worker registry, the typed enqueue API the HTTP handlers call, and river's own
// schema migration (kept out of Atlas — see Migrate).
package jobs

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/secrets"
	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/mokevnin/1mail/internal/webhook"
)

// webhookDeliveryTimeout bounds a single outbound webhook request.
const webhookDeliveryTimeout = 15 * time.Second

// Client wraps the river client and exposes the typed enqueue methods the API
// handlers use, so callers don't depend on river directly.
type Client struct {
	river *river.Client[pgx.Tx]
	ent   *ent.Client
}

// NewClient builds the river client with all workers registered. Workers carry
// their own dependencies (ent client, sender resolver, secrets cipher).
func NewClient(pool *pgxpool.Pool, entClient *ent.Client, resolver *messaging.Resolver, tracker *tracking.Tracker, cipher *secrets.Cipher) (*Client, error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &SendBroadcastWorker{ent: entClient, resolver: resolver, tracker: tracker})
	river.AddWorker(workers, &EvaluateTriggerWorker{ent: entClient})
	river.AddWorker(workers, &RunStepWorker{ent: entClient, resolver: resolver})
	river.AddWorker(workers, &DeliverWebhookWorker{
		ent:    entClient,
		cipher: cipher,
		client: webhook.NewClient(webhookDeliveryTimeout),
	})

	rc, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 5},
		},
		Workers: workers,
	})
	if err != nil {
		return nil, err
	}
	return &Client{river: rc, ent: entClient}, nil
}

// Start begins processing jobs (run in a goroutine; returns once started).
func (c *Client) Start(ctx context.Context) error { return c.river.Start(ctx) }

// Stop drains and stops the workers.
func (c *Client) Stop(ctx context.Context) error { return c.river.Stop(ctx) }

// EnqueueBroadcast schedules a broadcast send. A nil scheduledAt sends ASAP;
// otherwise the job runs at scheduledAt.
func (c *Client) EnqueueBroadcast(ctx context.Context, broadcastID int64, scheduledAt *time.Time) error {
	opts := &river.InsertOpts{}
	if scheduledAt != nil {
		opts.ScheduledAt = *scheduledAt
	}
	_, err := c.river.Insert(ctx, SendBroadcastArgs{BroadcastID: broadcastID}, opts)
	return err
}

// OnEvent enrolls a contact into automations triggered by the given Event action.
// It is the trigger seam injected into the event-recording paths; a no-op
// implementation keeps tests inert (no river runtime, writes roll back with the
// tx). Fire-and-forget: enrollment is best-effort and must never block the write.
func (c *Client) OnEvent(ctx context.Context, workspaceID, contactID int64, action string) error {
	_, err := c.river.Insert(ctx, EvaluateTriggerArgs{
		WorkspaceID: workspaceID,
		ContactID:   contactID,
		Action:      action,
	}, nil)
	return err
}

// Migrate applies river's own schema (river_job, river_leader, …). This is run
// out of band from Atlas on purpose: Atlas diffs against the ent-derived schema
// and would otherwise try to drop river's tables. See cmd/db `river-up`.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}
	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	return err
}
