package eligibility_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent/confirmation"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/ent/unsubscribe"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wsID = int64(1)

func TestCheckEligibleByDefault(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	d, err := eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"fresh@example.com", eligibility.SourceBroadcasts, true, false)
	require.NoError(t, err)
	assert.True(t, d.Eligible)
	assert.Empty(t, d.Reason)
}

func TestCheckSuppressed(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	_, err := env.DB.Suppression.Create().SetWorkspaceID(wsID).
		SetChannel(suppression.ChannelEmail).SetDestination("blocked@example.com").
		SetReason(suppression.ReasonComplaint).Save(ctx)
	require.NoError(t, err)

	d, err := eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"blocked@example.com", eligibility.SourceBroadcasts, true, false)
	require.NoError(t, err)
	assert.False(t, d.Eligible)
	assert.Equal(t, eligibility.ReasonSuppressed, d.Reason)
}

// Transactional sends (respectUnsubscribe=false) skip layers 2–3 but still
// respect Suppression.
func TestCheckTransactionalSkipsUnsubscribeButNotSuppression(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// Unsubscribed from everything — yet a transactional send must still go.
	_, err := env.DB.Unsubscribe.Create().SetWorkspaceID(wsID).
		SetChannel(unsubscribe.ChannelEmail).SetDestination("txn@example.com").
		SetSendingSource(eligibility.SourceEverything).Save(ctx)
	require.NoError(t, err)

	d, err := eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"txn@example.com", eligibility.SourceBroadcasts, false, false)
	require.NoError(t, err)
	assert.True(t, d.Eligible, "transactional skips unsubscribe layers")

	// But a suppressed destination is never sent to, transactional or not.
	_, err = env.DB.Suppression.Create().SetWorkspaceID(wsID).
		SetChannel(suppression.ChannelEmail).SetDestination("txn@example.com").
		SetReason(suppression.ReasonBounce).Save(ctx)
	require.NoError(t, err)
	d, err = eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"txn@example.com", eligibility.SourceBroadcasts, false, false)
	require.NoError(t, err)
	assert.False(t, d.Eligible)
	assert.Equal(t, eligibility.ReasonSuppressed, d.Reason)
}

func TestCheckUnsubscribedEverythingVsSource(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	_, err := env.DB.Unsubscribe.Create().SetWorkspaceID(wsID).
		SetChannel(unsubscribe.ChannelEmail).SetDestination("evt@example.com").
		SetSendingSource(eligibility.SourceEverything).Save(ctx)
	require.NoError(t, err)
	d, err := eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"evt@example.com", eligibility.SourceBroadcasts, true, false)
	require.NoError(t, err)
	assert.Equal(t, eligibility.ReasonUnsubscribedEverything, d.Reason)

	src := eligibility.AutomationSource(42)
	_, err = env.DB.Unsubscribe.Create().SetWorkspaceID(wsID).
		SetChannel(unsubscribe.ChannelEmail).SetDestination("src@example.com").
		SetSendingSource(src).Save(ctx)
	require.NoError(t, err)
	// Ineligible for that source...
	d, err = eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail, "src@example.com", src, true, false)
	require.NoError(t, err)
	assert.Equal(t, eligibility.ReasonUnsubscribedSource, d.Reason)
	// ...but eligible for a different source.
	d, err = eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"src@example.com", eligibility.SourceBroadcasts, true, false)
	require.NoError(t, err)
	assert.True(t, d.Eligible)
}

// The confirmation gate (ADR 0013) is a no-op when requireConfirmed is false: an
// address with no Confirmation is still eligible, and the batch predicate still
// matches it. This is the "confirmed-opt-in off ⇒ behavior unchanged" invariant.
func TestConfirmationGateNoopWhenOff(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	d, err := eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"unconfirmed@example.com", eligibility.SourceBroadcasts, true, false)
	require.NoError(t, err)
	assert.True(t, d.Eligible, "no confirmation required when the gate is off")

	c, err := env.DB.Contact.Create().SetWorkspaceID(wsID).SetEmail("unconf@example.com").Save(ctx)
	require.NoError(t, err)
	matched, err := env.DB.Contact.Query().
		Where(contact.ID(c.ID),
			eligibility.Predicate(eligibility.ChannelEmail, eligibility.SourceBroadcasts, false)).
		Exist(ctx)
	require.NoError(t, err)
	assert.True(t, matched, "unconfirmed contact still in the audience when the gate is off")
}

// When requireConfirmed is true, an unconfirmed address is blocked and confirming
// it makes it mailable — via both the point Check and the batch Predicate.
func TestConfirmationGateWhenOn(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	c, err := env.DB.Contact.Create().SetWorkspaceID(wsID).SetEmail("gate@example.com").Save(ctx)
	require.NoError(t, err)

	inAudience := func() bool {
		ok, err := env.DB.Contact.Query().
			Where(contact.ID(c.ID),
				eligibility.Predicate(eligibility.ChannelEmail, eligibility.SourceBroadcasts, true)).
			Exist(ctx)
		require.NoError(t, err)
		return ok
	}

	// Unconfirmed: blocked.
	d, err := eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"gate@example.com", eligibility.SourceBroadcasts, true, true)
	require.NoError(t, err)
	assert.False(t, d.Eligible)
	assert.Equal(t, eligibility.ReasonUnconfirmed, d.Reason)
	assert.False(t, inAudience(), "unconfirmed contact excluded from the audience")

	// Confirm it: now mailable.
	_, err = env.DB.Confirmation.Create().SetWorkspaceID(wsID).
		SetChannel(confirmation.ChannelEmail).SetDestination("gate@example.com").
		SetProvenance(confirmation.ProvenanceDoubleOptIn).Save(ctx)
	require.NoError(t, err)

	d, err = eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"gate@example.com", eligibility.SourceBroadcasts, true, true)
	require.NoError(t, err)
	assert.True(t, d.Eligible)
	assert.True(t, inAudience(), "confirmed contact included in the audience")
}

// Confirmation is the lowest-priority layer: a negative always dominates. A
// confirmed-but-suppressed address is still ineligible (and reports the negative).
func TestConfirmationDoesNotOverrideNegatives(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	_, err := env.DB.Confirmation.Create().SetWorkspaceID(wsID).
		SetChannel(confirmation.ChannelEmail).SetDestination("both@example.com").
		SetProvenance(confirmation.ProvenanceDoubleOptIn).Save(ctx)
	require.NoError(t, err)
	_, err = env.DB.Suppression.Create().SetWorkspaceID(wsID).
		SetChannel(suppression.ChannelEmail).SetDestination("both@example.com").
		SetReason(suppression.ReasonComplaint).Save(ctx)
	require.NoError(t, err)

	d, err := eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"both@example.com", eligibility.SourceBroadcasts, true, true)
	require.NoError(t, err)
	assert.False(t, d.Eligible)
	assert.Equal(t, eligibility.ReasonSuppressed, d.Reason, "negatives dominate the confirmation")
}

// The batch Predicate folds case in SQL: a mixed-case contact email matches a
// lower-cased suppression destination and is excluded from the query.
func TestPredicateFoldsCaseAndExcludes(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	c, err := env.DB.Contact.Create().SetWorkspaceID(wsID).SetEmail("Caps@Example.com").Save(ctx)
	require.NoError(t, err)
	_, err = env.DB.Suppression.Create().SetWorkspaceID(wsID).
		SetChannel(suppression.ChannelEmail).SetDestination("caps@example.com").
		SetReason(suppression.ReasonBounce).Save(ctx)
	require.NoError(t, err)

	matched, err := env.DB.Contact.Query().
		Where(contact.ID(c.ID),
			eligibility.Predicate(eligibility.ChannelEmail, eligibility.SourceBroadcasts, false)).
		Exist(ctx)
	require.NoError(t, err)
	assert.False(t, matched, "suppressed contact excluded despite case difference")
}

// Stored destinations are lower-cased; a mixed-case contact email must still match.
func TestCheckFoldsCase(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	_, err := env.DB.Suppression.Create().SetWorkspaceID(wsID).
		SetChannel(suppression.ChannelEmail).SetDestination("mixed@example.com").
		SetReason(suppression.ReasonBounce).Save(ctx)
	require.NoError(t, err)

	d, err := eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"  Mixed@Example.com  ", eligibility.SourceBroadcasts, true, false)
	require.NoError(t, err)
	assert.False(t, d.Eligible, "input is normalized before lookup")
}
