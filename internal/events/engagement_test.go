package events_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The send path stamps EmailEngagement with a deterministic DedupID; the publisher
// must surface it as the envelope DedupKey (via the identifiable interface), which
// is what makes persist idempotent under a job retry. Open/click leave DedupID
// empty and must keep falling back to the ULID — unchanged behavior.
func TestEmailEngagementDedupIDBecomesDedupKey(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	require.NoError(t, env.Bus.WithinTx(ctx, func(_ *ent.Client, pub events.Publisher) error {
		if err := pub.Publish(ctx, &events.EmailEngagement{
			Action: events.NameEmailSent, WorkspaceID: fixtureWorkspace,
			Email: "sent@example.com", BroadcastID: 7, DedupID: "email.sent:42",
		}); err != nil {
			return err
		}
		return pub.Publish(ctx, &events.EmailEngagement{
			Action: events.NameEmailOpened, WorkspaceID: fixtureWorkspace,
			Email: "sent@example.com", BroadcastID: 7,
		})
	}))

	dedupOf := func(name string) string {
		t.Helper()
		var v *string
		require.NoError(t, env.SQLDB.QueryRow(
			`SELECT payload->>'dedupKey' FROM watermill_domain_events WHERE payload->>'name' = $1`,
			name,
		).Scan(&v))
		if v == nil {
			return ""
		}
		return *v
	}

	assert.Equal(t, "email.sent:42", dedupOf(events.NameEmailSent),
		"email.sent carries its DedupID as the envelope DedupKey so a retry persists once")
	assert.Empty(t, dedupOf(events.NameEmailOpened),
		"open carries no DedupID → DedupKey stays empty (ULID fallback, unchanged)")
}
