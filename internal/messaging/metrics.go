package messaging

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const otelScope = "github.com/mokevnin/1mail/internal/messaging"

// SendStatus is the outcome label of the email.send.outcomes counter. The set is
// closed: the counter carries only provider and status, never a tenant or
// recipient identifier (ADR 0018).
type SendStatus string

const (
	// SendAccepted: the provider accepted the message.
	SendAccepted SendStatus = "accepted"
	// SendError: the provider call failed.
	SendError SendStatus = "error"
	// SendBounce: the provider reported a bounce for a message it had accepted.
	SendBounce SendStatus = "bounce"
	// SendComplaint: the provider reported a spam complaint.
	SendComplaint SendStatus = "complaint"
)

// RecordSendOutcome counts one send outcome for a provider on the global meter
// provider (a no-op when telemetry is disabled).
func RecordSendOutcome(ctx context.Context, provider Provider, status SendStatus) {
	counter, err := otel.Meter(otelScope).Int64Counter(
		"email.send.outcomes",
		metric.WithDescription("Email send outcomes by provider and status."),
	)
	if err != nil {
		return
	}
	counter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("provider", string(provider)),
		attribute.String("status", string(status)),
	))
}

// instrumentedSender counts accepted/error per provider around Send.
type instrumentedSender struct {
	EmailSender
	provider Provider
}

// DefaultFrom forwards the wrapped sender's DefaultFromer, which the decorator
// would otherwise hide from the send path (see DefaultFromer).
func (s instrumentedSender) DefaultFrom() (string, string) {
	if d, ok := s.EmailSender.(DefaultFromer); ok {
		return d.DefaultFrom()
	}
	return "", ""
}

func (s instrumentedSender) Send(ctx context.Context, msg EmailMessage) (Receipt, error) {
	receipt, err := s.EmailSender.Send(ctx, msg)
	if err != nil {
		RecordSendOutcome(ctx, s.provider, SendError)
		return receipt, err
	}
	RecordSendOutcome(ctx, s.provider, SendAccepted)
	return receipt, nil
}
