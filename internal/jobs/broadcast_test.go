package jobs_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/internal/jobs"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSender struct {
	sent []messaging.EmailMessage
}

func (f *fakeSender) Send(_ context.Context, msg messaging.EmailMessage) error {
	f.sent = append(f.sent, msg)
	return nil
}

type fakeResolver struct {
	sender messaging.EmailSender
	err    error
}

func (r fakeResolver) EmailSender(context.Context, int64) (messaging.EmailSender, error) {
	return r.sender, r.err
}

// Fixture workspace "acme" is id 1.
const acmeWorkspaceID = int64(1)

func TestSendBroadcastDeliversToActiveContacts(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	activeCount, err := env.DB.Contact.Query().
		Where(contact.WorkspaceID(acmeWorkspaceID), contact.StatusEQ(contact.StatusActive)).
		Count(ctx)
	require.NoError(t, err)
	require.Positive(t, activeCount, "fixtures must have active contacts")

	b, err := env.DB.Broadcast.Create().
		SetWorkspaceID(acmeWorkspaceID).
		SetName("Engine test").
		SetSubject("Hello {{ first_name }}").
		SetBodyHTML("<p>Hi {{ first_name }}, welcome!</p>").
		SetStatus(broadcast.StatusSending).
		Save(ctx)
	require.NoError(t, err)

	fs := &fakeSender{}
	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, fakeResolver{sender: fs}, tracking.New("test-secret", "http://local"), b.ID))

	// One message per active contact, with merge tags rendered (no raw braces).
	assert.Len(t, fs.sent, activeCount)
	for _, m := range fs.sent {
		assert.NotContains(t, m.Subject, "{{")
		assert.NotContains(t, m.HTML, "{{")
		assert.NotEmpty(t, m.Text, "text part derived from HTML")
	}

	// Broadcast is marked sent with accurate counters.
	got := env.DB.Broadcast.GetX(ctx, b.ID)
	assert.Equal(t, broadcast.StatusSent, got.Status)
	assert.Equal(t, activeCount, got.RecipientsTotal)
	assert.Equal(t, activeCount, got.SentCount)
	assert.Equal(t, 0, got.FailedCount)
	require.NotNil(t, got.SentAt)

	// A recipient row exists per contact, all marked sent.
	recs, err := env.DB.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(b.ID)).
		All(ctx)
	require.NoError(t, err)
	assert.Len(t, recs, activeCount)
	for _, r := range recs {
		assert.Equal(t, broadcastrecipient.StatusSent, r.Status)
	}
}

func TestSendBroadcastFailsWithoutProvider(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	b, err := env.DB.Broadcast.Create().
		SetWorkspaceID(acmeWorkspaceID).
		SetName("No provider").
		SetSubject("Hi").
		Save(ctx)
	require.NoError(t, err)

	err = jobs.SendBroadcast(ctx, env.DB, fakeResolver{err: messaging.ErrNoProvider}, tracking.New("test-secret", "http://local"), b.ID)
	require.Error(t, err)

	got := env.DB.Broadcast.GetX(ctx, b.ID)
	assert.Equal(t, broadcast.StatusFailed, got.Status)
}
