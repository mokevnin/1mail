package events_test

import (
	"context"
	"testing"
	"time"

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
