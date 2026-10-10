package events_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// Outbox lag is the age of the oldest row past each group's cursor, one sample
// per registered consumer group, labelled by group only.
func TestOutboxLagIsReportedPerConsumerGroup(t *testing.T) {
	env := testhelper.Setup(t)
	testhelper.StartMetrics(t)
	unregister, err := events.RegisterLagGauge(env.SQLDB)
	require.NoError(t, err)
	t.Cleanup(func() { _ = unregister() })

	err = env.Bus.WithinTx(t.Context(), func(_ *ent.Client, pub events.Publisher) error {
		return pub.Publish(t.Context(), &events.ContactCreated{WorkspaceID: fixtures.AcmeID, ContactID: 7, Email: "lag@example.com"})
	})
	require.NoError(t, err)
	env.AgeOutbox(t, 2*time.Minute)
	env.AckOutbox(t, events.GroupPersist)

	scrape := testhelper.ScrapeMetrics(t)
	assert.Zero(t, scrape.Value(t, "outbox_lag_seconds", map[string]string{"consumer_group": events.GroupPersist}), "caught up")
	for _, g := range []string{events.GroupAutomations, events.GroupWebhooks, events.GroupSuppression} {
		assert.InDelta(t, 120, scrape.Value(t, "outbox_lag_seconds", map[string]string{"consumer_group": g}), 5, g)
	}
}
