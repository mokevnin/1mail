package events_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// workspace id 1 (slug "acme") is seeded by fixtures.
const fixtureWorkspace = int64(1)

func outboxCount(t *testing.T, env *testhelper.TestEnv) int {
	t.Helper()
	var n int
	err := env.SQLDB.QueryRow(`SELECT count(*) FROM watermill_domain_events`).Scan(&n)
	require.NoError(t, err)
	return n
}

// dataFor marshals a typed event into the envelope Data, as the publisher does.
func dataFor(t *testing.T, ev events.DomainEvent) []byte {
	t.Helper()
	b, err := json.Marshal(ev)
	require.NoError(t, err)
	return b
}

// The load-bearing requirement: the state row and the outbox row commit together.
func TestBusWithinTxCommitsStateAndOutbox(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	before := outboxCount(t, env)

	var created *ent.Contact
	err := env.Bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		c, err := tx.Contact.Create().
			SetWorkspaceID(fixtureWorkspace).
			SetEmail("outbox-commit@example.com").
			Save(ctx)
		if err != nil {
			return err
		}
		created = c
		return pub.Publish(ctx, &events.ContactCreated{WorkspaceID: fixtureWorkspace, ContactID: c.ID, Email: c.Email})
	})
	require.NoError(t, err)

	// State row committed (visible through the shared txdb connection)...
	got, err := env.DB.Contact.Query().Where(contact.Email("outbox-commit@example.com")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	// ...and exactly one outbox row was appended in the same transaction.
	assert.Equal(t, before+1, outboxCount(t, env))
}

// If the unit of work fails, neither the state nor the event is published.
func TestBusWithinTxRollsBackBoth(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	before := outboxCount(t, env)

	boom := errors.New("boom")
	err := env.Bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		if _, err := tx.Contact.Create().
			SetWorkspaceID(fixtureWorkspace).
			SetEmail("outbox-rollback@example.com").
			Save(ctx); err != nil {
			return err
		}
		if err := pub.Publish(ctx, &events.ContactCreated{WorkspaceID: fixtureWorkspace, Email: "outbox-rollback@example.com"}); err != nil {
			return err
		}
		return boom
	})
	require.ErrorIs(t, err, boom)

	exists, err := env.DB.Contact.Query().Where(contact.Email("outbox-rollback@example.com")).Exist(ctx)
	require.NoError(t, err)
	assert.False(t, exists, "contact must not be committed when the unit of work fails")
	assert.Equal(t, before, outboxCount(t, env), "no event published on rollback")
}

// Persist projects an envelope to the durable Event row. This is the load-bearing
// check on the projection (the router doesn't run under txdb): every column —
// including phone, prospect, a non-now OccurredAt, and opaque Properties — must
// land. Uses a CollectedEvent because it exercises all of them.
func TestPersistWritesEventProjection(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	occurred := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	prospect := true
	collected := &events.CollectedEvent{
		WorkspaceID: fixtureWorkspace,
		SubjectID:   "visitor:abc",
		Action:      "page_view",
		Email:       "persist@example.com",
		Phone:       "+15550100",
		Prospect:    &prospect,
		Properties:  map[string]any{"path": "/pricing"},
		OccurredAt:  occurred,
	}
	envlp := events.Envelope{
		ID:          "01J0PERSISTTEST00000000000",
		Name:        events.NameCollected,
		Version:     1,
		WorkspaceID: fixtureWorkspace,
		OccurredAt:  occurred,
		Data:        dataFor(t, collected),
	}
	require.NoError(t, events.Persist(ctx, env.DB, envlp))

	row, err := env.DB.Event.Query().Where(event.SourceID(envlp.ID)).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "page_view", row.Action)
	assert.Equal(t, "visitor:abc", row.SubjectID)
	require.NotNil(t, row.Email)
	assert.Equal(t, "persist@example.com", *row.Email)
	require.NotNil(t, row.Phone)
	assert.Equal(t, "+15550100", *row.Phone)
	require.NotNil(t, row.Prospect)
	assert.True(t, *row.Prospect)
	require.NotNil(t, row.OccurredAt)
	assert.Equal(t, occurred, row.OccurredAt.UTC())
	assert.Equal(t, "/pricing", row.Properties["path"])
}

// At-least-once delivery can redeliver an envelope; persist must dedupe on the
// source id (envelope ULID) so the projection row is written at most once.
func TestPersistIsIdempotent(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	envlp := events.Envelope{
		ID:          "01J0IDEMPOTENT00000000000",
		Name:        events.NameContactCreated,
		Version:     1,
		WorkspaceID: fixtureWorkspace,
		OccurredAt:  time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		Data:        dataFor(t, &events.ContactCreated{WorkspaceID: fixtureWorkspace, ContactID: 7, Email: "dedupe@example.com"}),
	}
	require.NoError(t, events.Persist(ctx, env.DB, envlp))
	require.NoError(t, events.Persist(ctx, env.DB, envlp)) // redelivery

	n, err := env.DB.Event.Query().Where(event.SourceID(envlp.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "redelivered envelope must not write a second row")
}
