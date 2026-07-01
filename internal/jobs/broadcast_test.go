package jobs_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/segment"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/ent/unsubscribe"
	"github.com/mokevnin/1mail/internal/eligibility"
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

func TestSendBroadcastDeliversToEligibleContacts(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// Eligibility is derived (ADR 0001): the audience is the workspace contacts
	// minus the ineligible. Bob is unsubscribed from "broadcasts" in the fixtures,
	// so alice + carol remain.
	eligible, err := env.DB.Contact.Query().
		Where(contact.WorkspaceID(acmeWorkspaceID),
			eligibility.Predicate(eligibility.ChannelEmail, eligibility.SourceBroadcasts)).
		All(ctx)
	require.NoError(t, err)
	eligibleCount := len(eligible)
	require.Greater(t, eligibleCount, 0, "the workspace has broadcast-eligible contacts")
	// Bob is unsubscribed from "broadcasts", so he must not be in the eligible set.
	for _, e := range eligible {
		require.NotEqual(t, "bob@example.com", *e.Email, "unsubscribed contact is excluded from the audience")
	}

	b, err := env.DB.Broadcast.Create().
		SetWorkspaceID(acmeWorkspaceID).
		SetName("Engine test").
		SetSubject("Hello {{ first_name }}").
		SetBody("<mjml><mj-body><mj-section><mj-column><mj-text>Hi {{ first_name }}, welcome!</mj-text></mj-column></mj-section></mj-body></mjml>").
		SetStatus(broadcast.StatusSending).
		Save(ctx)
	require.NoError(t, err)

	fs := &fakeSender{}
	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{sender: fs}, tracking.New("test-secret", "http://local"), b.ID))

	// One message per eligible contact, with merge tags rendered (no raw braces).
	assert.Len(t, fs.sent, eligibleCount)
	for _, m := range fs.sent {
		assert.NotEqual(t, "bob@example.com", m.To, "unsubscribed destination must not be sent to")
		assert.NotContains(t, m.Subject, "{{")
		assert.NotContains(t, m.HTML, "{{")
		assert.NotEmpty(t, m.Text, "text part derived from HTML")
	}

	// Broadcast is marked sent with accurate counters.
	got := env.DB.Broadcast.GetX(ctx, b.ID)
	assert.Equal(t, broadcast.StatusSent, got.Status)
	assert.Equal(t, eligibleCount, got.RecipientsTotal)
	assert.Equal(t, eligibleCount, got.SentCount)
	assert.Equal(t, 0, got.FailedCount)
	require.NotNil(t, got.SentAt)

	// A recipient row exists per eligible contact, all marked sent.
	recs, err := env.DB.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(b.ID)).
		All(ctx)
	require.NoError(t, err)
	assert.Len(t, recs, eligibleCount)
	for _, r := range recs {
		assert.Equal(t, broadcastrecipient.StatusSent, r.Status)
	}

	// Each send publishes an email.sent event onto the transactional outbox, so the
	// send fact reaches the Event log (Events are the source of truth; persist runs
	// off the bus, not under txdb — projection is covered by the events tests). One
	// per sent recipient, no more (exactly-once).
	assert.Equal(t, eligibleCount, countOutboxEvents(t, env, "email.sent", b.ID))
}

// countOutboxEvents returns how many events of the given name for the given
// broadcast sit in the transactional outbox.
func countOutboxEvents(t *testing.T, env *testhelper.TestEnv, name string, broadcastID int64) int {
	t.Helper()
	var n int
	err := env.SQLDB.QueryRow(
		`SELECT count(*) FROM watermill_domain_events
		   WHERE payload->>'name' = $1 AND payload->'data'->>'broadcastId' = $2`,
		name, strconv.FormatInt(broadcastID, 10),
	).Scan(&n)
	require.NoError(t, err)
	return n
}

// Suppressed addresses are skipped: no message, no recipient row, and the
// suppressed contact is excluded from the recipients_total.
func TestSendBroadcastSkipsSuppressed(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// Eligible for broadcasts before suppression: alice + carol (bob is
	// unsubscribed from broadcasts in the fixtures).
	eligibleBefore, err := env.DB.Contact.Query().
		Where(contact.WorkspaceID(acmeWorkspaceID),
			eligibility.Predicate(eligibility.ChannelEmail, eligibility.SourceBroadcasts)).
		Count(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, eligibleBefore, 2)

	// Suppress alice (id 1 in fixtures) — a global hard floor on every surface.
	alice := env.DB.Contact.GetX(ctx, 1)
	_, err = env.DB.Suppression.Create().
		SetWorkspaceID(acmeWorkspaceID).
		SetChannel(suppression.ChannelEmail).
		SetDestination(*alice.Email).
		SetReason(suppression.ReasonManual).
		Save(ctx)
	require.NoError(t, err)

	b, err := env.DB.Broadcast.Create().
		SetWorkspaceID(acmeWorkspaceID).
		SetName("Suppress test").
		SetSubject("Hi").
		SetBody("<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>").
		SetStatus(broadcast.StatusSending).
		Save(ctx)
	require.NoError(t, err)

	fs := &fakeSender{}
	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{sender: fs}, tracking.New("s", "http://local"), b.ID))

	// alice gets no message.
	for _, m := range fs.sent {
		assert.NotEqual(t, *alice.Email, m.To, "suppressed address must not be sent to")
	}
	assert.Len(t, fs.sent, eligibleBefore-1)

	// Counters exclude the suppressed contact, and she has no recipient row.
	got := env.DB.Broadcast.GetX(ctx, b.ID)
	assert.Equal(t, eligibleBefore-1, got.RecipientsTotal)
	assert.Equal(t, eligibleBefore-1, got.SentCount)
	n, err := env.DB.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(b.ID), broadcastrecipient.ContactID(alice.ID)).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "no recipient row for a suppressed contact")
}

// A destination unsubscribed from "everything" is excluded from broadcasts even
// though it carries no per-source ("broadcasts") opt-out.
func TestSendBroadcastSkipsUnsubscribedFromEverything(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	c, err := env.DB.Contact.Create().SetWorkspaceID(acmeWorkspaceID).
		SetEmail("gone@everything.test").SetFirstName("Gone").Save(ctx)
	require.NoError(t, err)
	_, err = env.DB.Unsubscribe.Create().
		SetWorkspaceID(acmeWorkspaceID).
		SetChannel(unsubscribe.ChannelEmail).
		SetDestination(*c.Email).
		SetSendingSource(eligibility.SourceEverything).
		SetContactID(c.ID).
		Save(ctx)
	require.NoError(t, err)

	b, err := env.DB.Broadcast.Create().
		SetWorkspaceID(acmeWorkspaceID).
		SetName("Everything test").
		SetSubject("Hi").
		SetBody("<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>").
		SetStatus(broadcast.StatusSending).
		Save(ctx)
	require.NoError(t, err)

	fs := &fakeSender{}
	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{sender: fs}, tracking.New("s", "http://local"), b.ID))

	// Positive control: the eligible audience (alice + carol; bob is broadcasts-
	// unsubscribed) is still delivered to, and the everything-opt-out is excluded.
	require.NotEmpty(t, fs.sent, "eligible contacts must still receive the broadcast")
	got := make([]string, len(fs.sent))
	for i, m := range fs.sent {
		got[i] = m.To
	}
	assert.NotContains(t, got, *c.Email, "everything-unsubscribed destination must not be sent to")
	assert.Contains(t, got, "alice@example.com")
	assert.Contains(t, got, "carol@example.com")
	assert.NotContains(t, got, "bob@example.com")
}

func TestSendBroadcastToRuleSegment(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// A sentinel plan value unique to this test's contact, so the segment audience
	// is exactly the one created here regardless of how many fixture contacts exist.
	_, err := env.DB.Contact.Create().SetWorkspaceID(acmeWorkspaceID).
		SetEmail("pro@seg2.test").SetFirstName("Pro").
		SetCustomFields(map[string]any{"plan": "seg2pro"}).Save(ctx)
	require.NoError(t, err)
	_, err = env.DB.Contact.Create().SetWorkspaceID(acmeWorkspaceID).
		SetEmail("free@seg2.test").SetFirstName("Free").
		SetCustomFields(map[string]any{"plan": "free"}).Save(ctx)
	require.NoError(t, err)

	seg, err := env.DB.Segment.Create().SetWorkspaceID(acmeWorkspaceID).
		SetName("Pro plan").SetType(segment.TypeRule).
		SetDefinition(`{"combinator":"and","rules":[{"field":"custom:plan","operator":"=","value":"seg2pro"}]}`).
		Save(ctx)
	require.NoError(t, err)

	b, err := env.DB.Broadcast.Create().SetWorkspaceID(acmeWorkspaceID).
		SetName("Segmented").SetSubject("Hi").SetBody("<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>").
		SetSegmentID(seg.ID).Save(ctx)
	require.NoError(t, err)

	fs := &fakeSender{}
	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{sender: fs}, tracking.New("s", "http://local"), b.ID))

	// Only the pro-plan contact is in the audience.
	require.Len(t, fs.sent, 1)
	assert.Equal(t, "pro@seg2.test", fs.sent[0].To)
	assert.Equal(t, 1, env.DB.Broadcast.GetX(ctx, b.ID).RecipientsTotal)
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

	err = jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{err: messaging.ErrNoProvider}, tracking.New("test-secret", "http://local"), b.ID)
	require.Error(t, err)

	got := env.DB.Broadcast.GetX(ctx, b.ID)
	assert.Equal(t, broadcast.StatusFailed, got.Status)
}
