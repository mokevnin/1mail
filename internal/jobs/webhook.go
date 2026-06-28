package jobs

import (
	"context"
	"net/http"

	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/webhookendpoint"
	"github.com/mokevnin/1mail/internal/secrets"
	"github.com/mokevnin/1mail/internal/webhook"
)

// DeliverWebhookArgs is one delivery attempt of an event to one endpoint. river
// retries on error, so a receiver may see duplicates — DeliveryID lets it dedupe.
type DeliverWebhookArgs struct {
	EndpointID int64  `json:"endpoint_id"`
	EventName  string `json:"event_name"`
	DeliveryID string `json:"delivery_id"`
	Body       []byte `json:"body"`
}

func (DeliverWebhookArgs) Kind() string { return "deliver_webhook" }

type DeliverWebhookWorker struct {
	river.WorkerDefaults[DeliverWebhookArgs]
	ent    *ent.Client
	cipher *secrets.Cipher
	client *http.Client
}

func (w *DeliverWebhookWorker) Work(ctx context.Context, job *river.Job[DeliverWebhookArgs]) error {
	e, err := w.ent.WebhookEndpoint.Get(ctx, job.Args.EndpointID)
	if ent.IsNotFound(err) {
		return nil // endpoint deleted since enqueue; drop the delivery
	}
	if err != nil {
		return err
	}
	if !e.Enabled {
		return nil // disabled since enqueue
	}

	secret, err := w.cipher.Decrypt(e.SecretEncrypted)
	if err != nil {
		return err
	}
	return webhook.Send(ctx, w.client, e.URL, string(secret), job.Args.EventName, job.Args.DeliveryID, job.Args.Body)
}

// Dispatch fans a domain event out to every enabled endpoint in the workspace
// whose filter matches, enqueuing one delivery job each. Implements
// events.WebhookDispatcher.
func (c *Client) Dispatch(ctx context.Context, workspaceID int64, eventName, deliveryID string, body []byte) error {
	endpoints, err := c.ent.WebhookEndpoint.Query().
		Where(webhookendpoint.WorkspaceID(workspaceID), webhookendpoint.Enabled(true)).
		All(ctx)
	if err != nil {
		return err
	}
	for _, e := range endpoints {
		if !matchesEvent(e.EventTypes, eventName) {
			continue
		}
		if _, err := c.river.Insert(ctx, DeliverWebhookArgs{
			EndpointID: e.ID,
			EventName:  eventName,
			DeliveryID: deliveryID,
			Body:       body,
		}, &river.InsertOpts{MaxAttempts: 10}); err != nil {
			return err
		}
	}
	return nil
}

// matchesEvent reports whether an endpoint with the given filter receives the
// event. An empty filter means "all events".
func matchesEvent(filter []string, name string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, f := range filter {
		if f == name {
			return true
		}
	}
	return false
}
