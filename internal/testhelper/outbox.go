package testhelper

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/events"
)

// The domain-event outbox is a watermill table, not an ent entity, so tests read it
// through these helpers instead of querying it themselves.

// OutboxEnvelopes returns the envelopes published on the outbox in this test's
// transaction, oldest first. With names, only envelopes of those event names.
func (env *TestEnv) OutboxEnvelopes(t *testing.T, names ...string) []events.Envelope {
	t.Helper()
	envelopes, err := readOutbox(t.Context(), env.SQLDB, names)
	require.NoError(t, err)
	return envelopes
}

// DeliverToEE hands every envelope on this test's outbox to the Enterprise
// subscribers, as the router would (no router runs under txdb). Each call redelivers
// the whole outbox, so calling it twice exercises at-least-once redelivery.
func (env *TestEnv) DeliverToEE(t *testing.T) {
	t.Helper()
	for _, e := range env.OutboxEnvelopes(t) {
		for _, c := range env.edition.Consumers {
			require.NoError(t, c.Handle(t.Context(), e), c.Name)
		}
	}
}

// OutboxEvents is OutboxEnvelopes decoded into their typed events.
func (env *TestEnv) OutboxEvents(t *testing.T, names ...string) []events.DomainEvent {
	t.Helper()
	var out []events.DomainEvent
	for _, e := range env.OutboxEnvelopes(t, names...) {
		ev, err := events.Decode(e)
		require.NoError(t, err)
		out = append(out, ev)
	}
	return out
}

// OutboxMessage is one outbox envelope with its payload decoded generically
// (numbers as json.Number, so ids compare exactly).
type OutboxMessage struct {
	Envelope events.Envelope
	Data     map[string]any
}

// Outbox returns the outbox messages of this test's transaction, oldest first,
// optionally only those of the given event names.
func (env *TestEnv) Outbox(t *testing.T, names ...string) []OutboxMessage {
	t.Helper()
	var out []OutboxMessage
	for _, e := range env.OutboxEnvelopes(t, names...) {
		dec := json.NewDecoder(bytes.NewReader(e.Data))
		dec.UseNumber()
		var data map[string]any
		require.NoError(t, dec.Decode(&data))
		out = append(out, OutboxMessage{Envelope: e, Data: data})
	}
	return out
}

// OutboxCount counts the outbox messages named name whose payload has every
// key/value of match (values compare by their printed form: 42 matches "42").
func (env *TestEnv) OutboxCount(t *testing.T, name string, match map[string]any) int {
	t.Helper()
	n := 0
	for _, m := range env.Outbox(t, name) {
		ok := true
		for k, want := range match {
			if fmt.Sprint(m.Data[k]) != fmt.Sprint(want) {
				ok = false
				break
			}
		}
		if ok {
			n++
		}
	}
	return n
}

// PurgeOutbox empties the outbox of a database a test wrote to for real (outside
// the per-test transaction), such as the router tests.
func PurgeOutbox(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DELETE FROM watermill_domain_events`)
	return err
}

// OutboxEnvelopesOn is OutboxEnvelopes for a database a test wrote to for real.
func OutboxEnvelopesOn(ctx context.Context, db *sql.DB, names ...string) ([]events.Envelope, error) {
	return readOutbox(ctx, db, names)
}

func readOutbox(ctx context.Context, db *sql.DB, names []string) ([]events.Envelope, error) {
	rows, err := db.QueryContext(ctx, `SELECT payload FROM watermill_domain_events ORDER BY "offset"`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []events.Envelope
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var e events.Envelope
		if err := json.Unmarshal(payload, &e); err != nil {
			return nil, err
		}
		if len(names) == 0 || slices.Contains(names, e.Name) {
			out = append(out, e)
		}
	}
	return out, rows.Err()
}
