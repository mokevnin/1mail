package broadcasts_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/internal/broadcasts"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// Fixtures (workspace acme = 1): broadcast 100 is a draft, 101 is scheduled, 102 is
// already sending.
const (
	acme      = int64(1)
	draftID   = int64(100)
	schedID   = int64(101)
	sendingID = int64(102)
)

type enqueued struct {
	id int64
	at *time.Time
}

// recorder is the job-queue seam: it records what was enqueued, or fails.
type recorder struct {
	calls []enqueued
	err   error
}

func (r *recorder) EnqueueBroadcast(_ context.Context, id int64, at *time.Time) error {
	if r.err != nil {
		return r.err
	}
	r.calls = append(r.calls, enqueued{id, at})
	return nil
}

func TestSendMovesDraftToSendingAndEnqueuesImmediately(t *testing.T) {
	env := testhelper.Setup(t)
	q := &recorder{}
	m := broadcasts.New(env.DB, q)

	b, err := m.Send(context.Background(), acme, draftID)
	require.NoError(t, err)

	assert.Equal(t, broadcast.StatusSending, b.Status)
	assert.Equal(t, []enqueued{{draftID, nil}}, q.calls)
}

func TestSendClearsTheScheduleOfAScheduledBroadcast(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})

	b, err := m.Send(context.Background(), acme, schedID)
	require.NoError(t, err)

	assert.Equal(t, broadcast.StatusSending, b.Status)
	assert.Nil(t, b.ScheduledAt)
}

func TestSendRevertsToDraftWhenEnqueueFails(t *testing.T) {
	env := testhelper.Setup(t)
	boom := errors.New("queue down")
	m := broadcasts.New(env.DB, &recorder{err: boom})

	_, err := m.Send(context.Background(), acme, draftID)

	require.ErrorIs(t, err, boom)
	assert.Equal(t, broadcast.StatusDraft, env.DB.Broadcast.GetX(context.Background(), draftID).Status)
}

func TestScheduleSetsTimeAndEnqueuesForThen(t *testing.T) {
	env := testhelper.Setup(t)
	q := &recorder{}
	m := broadcasts.New(env.DB, q)
	when := time.Now().Add(24 * time.Hour).Truncate(time.Second)

	b, err := m.Schedule(context.Background(), acme, draftID, when)
	require.NoError(t, err)

	assert.Equal(t, broadcast.StatusScheduled, b.Status)
	require.NotNil(t, b.ScheduledAt)
	assert.True(t, when.Equal(*b.ScheduledAt))
	require.Len(t, q.calls, 1)
	assert.Equal(t, draftID, q.calls[0].id)
	require.NotNil(t, q.calls[0].at)
	assert.True(t, when.Equal(*q.calls[0].at))
}

func TestScheduleRevertsToDraftAndClearsTimeWhenEnqueueFails(t *testing.T) {
	env := testhelper.Setup(t)
	boom := errors.New("queue down")
	m := broadcasts.New(env.DB, &recorder{err: boom})

	_, err := m.Schedule(context.Background(), acme, draftID, time.Now().Add(time.Hour))

	require.ErrorIs(t, err, boom)
	got := env.DB.Broadcast.GetX(context.Background(), draftID)
	assert.Equal(t, broadcast.StatusDraft, got.Status)
	assert.Nil(t, got.ScheduledAt)
}

func TestTransitionsOutOfSendingAreRefused(t *testing.T) {
	env := testhelper.Setup(t)
	q := &recorder{}
	m := broadcasts.New(env.DB, q)

	_, err := m.Send(context.Background(), acme, sendingID)
	assert.ErrorIs(t, err, broadcasts.ErrNotSendable)
	_, err = m.Schedule(context.Background(), acme, sendingID, time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, broadcasts.ErrNotSendable)
	assert.Empty(t, q.calls)
}

func TestUnknownOrForeignBroadcastIsNotFound(t *testing.T) {
	env := testhelper.Setup(t)
	m := broadcasts.New(env.DB, &recorder{})

	_, err := m.Send(context.Background(), acme+1000, draftID)
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
	_, err = m.Schedule(context.Background(), acme, 999999, time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, broadcasts.ErrNotFound)
}
