package messaging_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/testhelper/metricstest"
)

type outcomeSender struct{ err error }

func (s outcomeSender) Send(context.Context, messaging.EmailMessage) (messaging.Receipt, error) {
	return messaging.Receipt{MessageID: "m1"}, s.err
}

func (outcomeSender) DefaultFrom() (string, string) { return "from@example.com", "From" }

// Senders built through the catalog count accepted and error per provider, with
// no other label.
func TestCatalogSenderCountsSendOutcomes(t *testing.T) {
	metricstest.StartMetrics(t)
	failing := errors.New("provider down")
	catalog := messaging.NewCatalog(
		messaging.ProviderDescriptor{
			Channel: messaging.ChannelEmail, Provider: messaging.ProviderSMTP,
			Build: func([]byte, messaging.Signer) (any, error) { return outcomeSender{}, nil },
		},
		messaging.ProviderDescriptor{
			Channel: messaging.ChannelEmail, Provider: messaging.ProviderSES,
			Build: func([]byte, messaging.Signer) (any, error) { return outcomeSender{err: failing}, nil },
		},
	)
	ctx := t.Context()

	smtp, err := catalog.BuildEmail(messaging.ProviderSMTP, nil, nil)
	require.NoError(t, err)
	_, err = smtp.Send(ctx, messaging.EmailMessage{})
	require.NoError(t, err)
	_, err = smtp.Send(ctx, messaging.EmailMessage{})
	require.NoError(t, err)

	ses, err := catalog.BuildEmail(messaging.ProviderSES, nil, nil)
	require.NoError(t, err)
	_, err = ses.Send(ctx, messaging.EmailMessage{})
	require.ErrorIs(t, err, failing)

	scrape := metricstest.ScrapeMetrics(t)
	assert.InDelta(t, 2, scrape.Value(t, "email_send_outcomes_total", map[string]string{"provider": "smtp", "status": "accepted"}), 0)
	assert.InDelta(t, 1, scrape.Value(t, "email_send_outcomes_total", map[string]string{"provider": "ses", "status": "error"}), 0)

	// The decorator keeps the optional DefaultFromer the send path relies on.
	from, ok := smtp.(messaging.DefaultFromer)
	require.True(t, ok)
	addr, _ := from.DefaultFrom()
	assert.Equal(t, "from@example.com", addr)
}

// SES bounce and complaint notifications count under provider "ses".
func TestRecordSendOutcomeBounceAndComplaint(t *testing.T) {
	metricstest.StartMetrics(t)
	messaging.RecordSendOutcome(t.Context(), messaging.ProviderSES, messaging.SendBounce)
	messaging.RecordSendOutcome(t.Context(), messaging.ProviderSES, messaging.SendComplaint)

	scrape := metricstest.ScrapeMetrics(t)
	assert.InDelta(t, 1, scrape.Value(t, "email_send_outcomes_total", map[string]string{"provider": "ses", "status": "bounce"}), 0)
	assert.InDelta(t, 1, scrape.Value(t, "email_send_outcomes_total", map[string]string{"provider": "ses", "status": "complaint"}), 0)
}
