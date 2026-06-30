package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An unsubscribe event adds the address to the workspace suppression list,
// normalized (lower-cased), with reason "unsubscribed".
func TestSuppressOnUnsubscribe(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	envlp := events.Envelope{
		ID:          "01J0SUPPRESS00000000000000",
		Name:        events.NameEmailUnsubscribed,
		Version:     1,
		WorkspaceID: fixtureWorkspace,
		OccurredAt:  time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		Data: dataFor(t, &events.EmailEngagement{
			Action:      events.NameEmailUnsubscribed,
			WorkspaceID: fixtureWorkspace,
			ContactID:   7,
			Email:       "Unsub@Example.com",
		}),
	}
	require.NoError(t, events.Suppress(ctx, env.DB, envlp))

	s, err := env.DB.Suppression.Query().
		Where(suppression.WorkspaceID(fixtureWorkspace), suppression.EmailEQ("unsub@example.com")).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, suppression.ReasonUnsubscribed, s.Reason)
	require.NotNil(t, s.ContactID)
	assert.EqualValues(t, 7, *s.ContactID)
}

// Opens/clicks (and any non-suppressing action) do not create a suppression.
func TestSuppressIgnoresNonSuppressingActions(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	envlp := events.Envelope{
		ID:          "01J0SUPPRESS00000000000001",
		Name:        events.NameEmailOpened,
		Version:     1,
		WorkspaceID: fixtureWorkspace,
		OccurredAt:  time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		Data: dataFor(t, &events.EmailEngagement{
			Action:      events.NameEmailOpened,
			WorkspaceID: fixtureWorkspace,
			ContactID:   7,
			Email:       "open@example.com",
		}),
	}
	require.NoError(t, events.Suppress(ctx, env.DB, envlp))

	n, err := env.DB.Suppression.Query().Where(suppression.WorkspaceID(fixtureWorkspace)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n)
}

// Publishing an EmailDeliveryFailure with a long natural id must succeed: the
// id becomes the envelope DedupKey, while the (short ULID) message id fills the
// outbox uuid column. Regression test for overflowing that column with the id.
func TestPublishDeliveryFailureWithLongDedupID(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	err := env.Bus.WithinTx(ctx, func(_ *ent.Client, pub events.Publisher) error {
		return pub.Publish(ctx, &events.EmailDeliveryFailure{
			Action:      events.NameEmailBounced,
			WorkspaceID: fixtureWorkspace,
			Email:       "recipient@example.com",
			BounceKind:  events.BounceKindPermanent,
			Provider:    "ses",
			// SNS messageId (36-char UUID) + "/" + email — well over 36 chars.
			DedupID: "9c8e1f2a-3b4d-4e5f-8a9b-0c1d2e3f4a5b/recipient@example.com",
		})
	})
	require.NoError(t, err)
}

// The DedupKey makes persist idempotent across at-least-once redelivery.
func TestPersistDedupesOnDedupKey(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	envlp := events.Envelope{
		ID:          "01J0DELIVERYFAILURE0000001",
		Name:        events.NameEmailBounced,
		Version:     1,
		WorkspaceID: fixtureWorkspace,
		OccurredAt:  time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		DedupKey:    "sns-msg-1/bounce@example.com",
		Data: dataFor(t, &events.EmailDeliveryFailure{
			Action: events.NameEmailBounced, WorkspaceID: fixtureWorkspace,
			Email: "bounce@example.com", BounceKind: events.BounceKindPermanent,
		}),
	}
	// Same DedupKey, different envelope ID (a redelivery) → one persisted row.
	require.NoError(t, events.Persist(ctx, env.DB, envlp))
	envlp.ID = "01J0DELIVERYFAILURE0000002"
	require.NoError(t, events.Persist(ctx, env.DB, envlp))

	n, err := env.DB.Event.Query().Where(event.SourceID(envlp.DedupKey)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "same DedupKey must persist at most once")
}

// A permanent bounce suppresses the address with reason "bounce".
func TestSuppressOnPermanentBounce(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	envlp := events.Envelope{
		ID:          "01J0SUPPRESS00000000000010",
		Name:        events.NameEmailBounced,
		Version:     1,
		WorkspaceID: fixtureWorkspace,
		OccurredAt:  time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		Data: dataFor(t, &events.EmailDeliveryFailure{
			Action:      events.NameEmailBounced,
			WorkspaceID: fixtureWorkspace,
			Email:       "hardbounce@example.com",
			BounceKind:  events.BounceKindPermanent,
			Provider:    "ses",
		}),
	}
	require.NoError(t, events.Suppress(ctx, env.DB, envlp))

	s, err := env.DB.Suppression.Query().
		Where(suppression.WorkspaceID(fixtureWorkspace), suppression.EmailEQ("hardbounce@example.com")).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, suppression.ReasonBounce, s.Reason)
}

// A transient (soft) bounce does NOT suppress — it is temporary.
func TestSuppressIgnoresTransientBounce(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	envlp := events.Envelope{
		ID:          "01J0SUPPRESS00000000000011",
		Name:        events.NameEmailBounced,
		Version:     1,
		WorkspaceID: fixtureWorkspace,
		OccurredAt:  time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		Data: dataFor(t, &events.EmailDeliveryFailure{
			Action:      events.NameEmailBounced,
			WorkspaceID: fixtureWorkspace,
			Email:       "softbounce@example.com",
			BounceKind:  events.BounceKindTransient,
			Provider:    "ses",
		}),
	}
	require.NoError(t, events.Suppress(ctx, env.DB, envlp))

	n, err := env.DB.Suppression.Query().Where(suppression.WorkspaceID(fixtureWorkspace)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n)
}

// A spam complaint suppresses the address with reason "complaint".
func TestSuppressOnComplaint(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	envlp := events.Envelope{
		ID:          "01J0SUPPRESS00000000000012",
		Name:        events.NameEmailComplained,
		Version:     1,
		WorkspaceID: fixtureWorkspace,
		OccurredAt:  time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		Data: dataFor(t, &events.EmailDeliveryFailure{
			Action:      events.NameEmailComplained,
			WorkspaceID: fixtureWorkspace,
			Email:       "complaint@example.com",
			Provider:    "ses",
		}),
	}
	require.NoError(t, events.Suppress(ctx, env.DB, envlp))

	s, err := env.DB.Suppression.Query().
		Where(suppression.WorkspaceID(fixtureWorkspace), suppression.EmailEQ("complaint@example.com")).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, suppression.ReasonComplaint, s.Reason)
}

// Redelivery or a repeat unsubscribe keeps a single entry (idempotent upsert).
func TestSuppressIsIdempotent(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	envlp := events.Envelope{
		ID:          "01J0SUPPRESS00000000000002",
		Name:        events.NameEmailUnsubscribed,
		Version:     1,
		WorkspaceID: fixtureWorkspace,
		OccurredAt:  time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		Data: dataFor(t, &events.EmailEngagement{
			Action:      events.NameEmailUnsubscribed,
			WorkspaceID: fixtureWorkspace,
			Email:       "dupe@example.com",
		}),
	}
	require.NoError(t, events.Suppress(ctx, env.DB, envlp))
	require.NoError(t, events.Suppress(ctx, env.DB, envlp)) // redelivery

	n, err := env.DB.Suppression.Query().
		Where(suppression.WorkspaceID(fixtureWorkspace), suppression.EmailEQ("dupe@example.com")).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}
