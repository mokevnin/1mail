package jobs

import (
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/outbound"
	"github.com/mokevnin/1mail/internal/secrets"
	"github.com/mokevnin/1mail/internal/sending"
	"github.com/mokevnin/1mail/internal/webhook"
)

// Worker constructors for the external jobs_test package: Work is exercised
// directly (no river runtime), with the dependencies each worker would carry.

func NewWelcomeWorker(sender messaging.EmailSender) *SendWelcomeWorker {
	return &SendWelcomeWorker{sender: sender}
}

func NewAuthMailWorker(sender messaging.EmailSender, appURL string) *SendAuthMailWorker {
	return &SendAuthMailWorker{sender: sender, appURL: appURL}
}

func NewMemberInviteWorker(sender messaging.EmailSender) *SendMemberInviteWorker {
	return &SendMemberInviteWorker{sender: sender}
}

func NewDeliverWebhookWorker(client *ent.Client, cipher *secrets.Cipher, doer webhook.Doer) *DeliverWebhookWorker {
	return &DeliverWebhookWorker{ent: client, cipher: cipher, client: doer}
}

func NewVerifySendingDomainWorker(client *ent.Client, lookup sending.TXTLookup, sender messaging.EmailSender) *VerifySendingDomainWorker {
	return &VerifySendingDomainWorker{ent: client, lookup: lookup, sender: sender}
}

func NewRecheckSendingDomainsWorker(client *ent.Client) *RecheckSendingDomainsWorker {
	return &RecheckSendingDomainsWorker{ent: client}
}

func NewEvaluateTriggerWorker(client *ent.Client) *EvaluateTriggerWorker {
	return &EvaluateTriggerWorker{ent: client}
}

func NewRunStepWorker(client *ent.Client, mod *outbound.Module) *RunStepWorker {
	return &RunStepWorker{ent: client, mod: mod}
}

func NewSendBroadcastWorker(client *ent.Client, mod *outbound.Module) *SendBroadcastWorker {
	return &SendBroadcastWorker{ent: client, mod: mod}
}

func NewSendRecipientWorker(client *ent.Client, mod *outbound.Module) *SendRecipientWorker {
	return &SendRecipientWorker{ent: client, mod: mod}
}

// River exposes the wrapped client so tests can build a worker context.
func (c *Client) River() *river.Client[pgx.Tx] { return c.river }

// NewErrorHandler builds river's error sink over logger.
func NewErrorHandler(logger *slog.Logger) river.ErrorHandler { return &errorHandler{logger: logger} }
