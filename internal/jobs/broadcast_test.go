package jobs_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/contact"
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

// erroringSender fails every send, to exercise the delivery-failure path.
type erroringSender struct{}

func (erroringSender) Send(context.Context, messaging.EmailMessage) error {
	return errors.New("smtp unavailable")
}

type fakeResolver struct {
	sender messaging.EmailSender
	err    error
}

func (r fakeResolver) EmailSender(context.Context, int64) (messaging.EmailSender, error) {
	return r.sender, r.err
}

// Fixture IDs (fixtures/broadcasts.yml, segments.yml). Workspace "acme" is id 1.
const (
	acmeWorkspaceID = int64(1)
	// Sendable broadcasts in non-terminal states, no recipient rows yet.
	draftBroadcastID     = int64(100)
	scheduledBroadcastID = int64(101)
	sendingBroadcastID   = int64(102)
	// Draft targeting segment 103, which matches no contact.
	emptyAudienceBroadcastID = int64(104)
	// Draft targeting segment 100 ("plan = pro").
	proSegmentBroadcastID = int64(105)
)

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

	// Send the draft fixture broadcast (100).
	fs := &fakeSender{}
	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{sender: fs}, tracking.New("test-secret", "http://local"), draftBroadcastID))

	// One message per eligible contact; no unrendered merge tags leak through
	// (substitution itself is covered by the emailrender tests).
	assert.Len(t, fs.sent, eligibleCount)
	for _, m := range fs.sent {
		assert.NotEqual(t, "bob@example.com", m.To, "unsubscribed destination must not be sent to")
		assert.NotContains(t, m.Subject, "{{")
		assert.NotContains(t, m.HTML, "{{")
		assert.NotEmpty(t, m.Text, "text part derived from HTML")
	}

	// Broadcast is marked sent with accurate counters.
	got := env.DB.Broadcast.GetX(ctx, draftBroadcastID)
	assert.Equal(t, broadcast.StatusSent, got.Status)
	assert.Equal(t, eligibleCount, got.RecipientsTotal)
	assert.Equal(t, eligibleCount, got.SentCount)
	assert.Equal(t, 0, got.FailedCount)
	require.NotNil(t, got.SentAt)

	// A recipient row exists per eligible contact, all marked sent.
	recs, err := env.DB.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(draftBroadcastID)).
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
	assert.Equal(t, eligibleCount, countOutboxEvents(t, env, "email.sent", draftBroadcastID))
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
// The verified-domain send gate (ADR 0010 slice 3): a broadcast pinned to a From
// whose domain has no verified sending domain fails at plan time — before any
// recipient row is created — rather than failing every recipient at send.
func TestPlanBroadcastRejectsUnverifiedFromDomain(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// news.acme.com exists as an *unverified* sending domain (fixture id 2).
	_, err := env.DB.Broadcast.UpdateOneID(draftBroadcastID).SetFromEmail("noreply@news.acme.com").Save(ctx)
	require.NoError(t, err)

	_, err = jobs.PlanBroadcast(ctx, env.DB, fakeResolver{sender: &fakeSender{}}, draftBroadcastID)
	assert.ErrorIs(t, err, messaging.ErrUnverifiedSendingDomain)

	got := env.DB.Broadcast.GetX(ctx, draftBroadcastID)
	assert.Equal(t, broadcast.StatusFailed, got.Status)

	n, err := env.DB.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(draftBroadcastID)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "no recipient rows created when the broadcast fails the domain gate")
}

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
	// The suppression is an incidental edge no fixture expresses, so it's created
	// inline; the broadcast itself is a fixture (scheduled broadcast 101).
	alice := env.DB.Contact.GetX(ctx, 1)
	_, err = env.DB.Suppression.Create().
		SetWorkspaceID(acmeWorkspaceID).
		SetChannel(suppression.ChannelEmail).
		SetDestination(*alice.Email).
		SetReason(suppression.ReasonManual).
		Save(ctx)
	require.NoError(t, err)

	fs := &fakeSender{}
	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{sender: fs}, tracking.New("s", "http://local"), scheduledBroadcastID))

	// alice gets no message.
	for _, m := range fs.sent {
		assert.NotEqual(t, *alice.Email, m.To, "suppressed address must not be sent to")
	}
	assert.Len(t, fs.sent, eligibleBefore-1)

	// Counters exclude the suppressed contact, and she has no recipient row.
	got := env.DB.Broadcast.GetX(ctx, scheduledBroadcastID)
	assert.Equal(t, eligibleBefore-1, got.RecipientsTotal)
	assert.Equal(t, eligibleBefore-1, got.SentCount)
	n, err := env.DB.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(scheduledBroadcastID), broadcastrecipient.ContactID(alice.ID)).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "no recipient row for a suppressed contact")
}

// A destination unsubscribed from "everything" is excluded from broadcasts even
// though it carries no per-source ("broadcasts") opt-out.
func TestSendBroadcastSkipsUnsubscribedFromEverything(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// The everything-opt-out contact + unsubscribe are the edge under test and no
	// fixture expresses them, so they're created inline; the broadcast is a fixture
	// (sending broadcast 102).
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

	fs := &fakeSender{}
	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{sender: fs}, tracking.New("s", "http://local"), sendingBroadcastID))

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

	// Fixture broadcast 105 targets fixture rule segment 100 ("plan = pro"), so the
	// audience is exactly the pro-plan fixture contacts (e.g. liam id 100), not the
	// free-plan ones (e.g. noah id 102).
	liam := env.DB.Contact.GetX(ctx, 100) // plan=pro
	noah := env.DB.Contact.GetX(ctx, 102) // plan=free

	fs := &fakeSender{}
	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{sender: fs}, tracking.New("s", "http://local"), proSegmentBroadcastID))

	got := make([]string, len(fs.sent))
	for i, m := range fs.sent {
		got[i] = m.To
	}
	require.NotEmpty(t, got, "pro-plan contacts are in the audience")
	assert.Contains(t, got, *liam.Email, "a pro-plan contact receives the broadcast")
	assert.NotContains(t, got, *noah.Email, "a free-plan contact is excluded by the segment")
	// recipients_total matches what was actually sent (all segment matches eligible).
	assert.Equal(t, len(fs.sent), env.DB.Broadcast.GetX(ctx, proSegmentBroadcastID).RecipientsTotal)
}

// A recipient is not re-sent when its send job is retried after a committed
// send: the second SendToRecipient is a no-op (no extra message). Drives the
// draft fixture broadcast through the real plan → send → retry flow.
func TestSendToRecipientIsIdempotent(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	fs := &fakeSender{}
	resolver := fakeResolver{sender: fs}
	tracker := tracking.New("s", "http://local")

	ids, err := jobs.PlanBroadcast(ctx, env.DB, resolver, draftBroadcastID)
	require.NoError(t, err)
	require.NotEmpty(t, ids)

	require.NoError(t, jobs.SendToRecipient(ctx, env.DB, env.Bus, resolver, tracker, ids[0]))
	require.NoError(t, jobs.SendToRecipient(ctx, env.DB, env.Bus, resolver, tracker, ids[0]))

	assert.Len(t, fs.sent, 1, "a re-run of an already-sent recipient must not re-send")
	assert.Equal(t, 1, countOutboxEvents(t, env, "email.sent", draftBroadcastID), "exactly one email.sent event")
}

// A send failure marks the recipient rows failed, finalizes the broadcast with
// failed_count == audience (and sent_count 0), and publishes no email.sent event.
func TestSendBroadcastMarksFailedOnSendError(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	eligible, err := env.DB.Contact.Query().
		Where(contact.WorkspaceID(acmeWorkspaceID),
			eligibility.Predicate(eligibility.ChannelEmail, eligibility.SourceBroadcasts)).
		Count(ctx)
	require.NoError(t, err)
	require.Greater(t, eligible, 0)

	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{sender: erroringSender{}}, tracking.New("s", "http://local"), scheduledBroadcastID))

	got := env.DB.Broadcast.GetX(ctx, scheduledBroadcastID)
	assert.Equal(t, broadcast.StatusSent, got.Status, "broadcast finalizes even when every send fails")
	assert.Equal(t, eligible, got.RecipientsTotal)
	assert.Equal(t, 0, got.SentCount)
	assert.Equal(t, eligible, got.FailedCount)
	require.NotNil(t, got.SentAt)

	recs, err := env.DB.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(scheduledBroadcastID)).
		All(ctx)
	require.NoError(t, err)
	assert.Len(t, recs, eligible)
	for _, r := range recs {
		assert.Equal(t, broadcastrecipient.StatusFailed, r.Status)
		require.NotNil(t, r.Error)
	}

	assert.Zero(t, countOutboxEvents(t, env, "email.sent", scheduledBroadcastID), "no email.sent event when the send fails")
}

// An empty audience finalizes immediately to sent with zero counters, so a
// broadcast never hangs in "sending" with no recipients to complete it. The
// fixture broadcast 104 targets segment 103, which matches no contact.
func TestPlanBroadcastEmptyAudienceFinalizes(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	ids, err := jobs.PlanBroadcast(ctx, env.DB, fakeResolver{sender: &fakeSender{}}, emptyAudienceBroadcastID)
	require.NoError(t, err)
	assert.Empty(t, ids)

	got := env.DB.Broadcast.GetX(ctx, emptyAudienceBroadcastID)
	assert.Equal(t, broadcast.StatusSent, got.Status)
	assert.Equal(t, 0, got.RecipientsTotal)
	assert.Equal(t, 0, got.SentCount)
	require.NotNil(t, got.SentAt)
}

// Finalizing an already-sent broadcast is a no-op: the status=sending guard
// keeps sent_at stable across concurrent/repeated finalizers.
func TestFinalizeBroadcastIsIdempotent(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	require.NoError(t, jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{sender: &fakeSender{}}, tracking.New("s", "http://local"), sendingBroadcastID))
	first := env.DB.Broadcast.GetX(ctx, sendingBroadcastID)
	require.Equal(t, broadcast.StatusSent, first.Status)
	require.NotNil(t, first.SentAt)

	require.NoError(t, jobs.FinalizeBroadcast(ctx, env.DB, sendingBroadcastID))
	second := env.DB.Broadcast.GetX(ctx, sendingBroadcastID)
	assert.True(t, first.SentAt.Equal(*second.SentAt), "sent_at must not move on a repeat finalize")
}

func TestSendBroadcastFailsWithoutProvider(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// Fixture broadcast 100; the resolver reports no usable provider.
	err := jobs.SendBroadcast(ctx, env.DB, env.Bus, fakeResolver{err: messaging.ErrNoProvider}, tracking.New("test-secret", "http://local"), draftBroadcastID)
	require.Error(t, err)

	got := env.DB.Broadcast.GetX(ctx, draftBroadcastID)
	assert.Equal(t, broadcast.StatusFailed, got.Status)
}
