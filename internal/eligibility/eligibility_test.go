package eligibility_test

import (
	"context"
	"testing"

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
		"fresh@example.com", eligibility.SourceBroadcasts, true)
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
		"blocked@example.com", eligibility.SourceBroadcasts, true)
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
		"txn@example.com", eligibility.SourceBroadcasts, false)
	require.NoError(t, err)
	assert.True(t, d.Eligible, "transactional skips unsubscribe layers")

	// But a suppressed destination is never sent to, transactional or not.
	_, err = env.DB.Suppression.Create().SetWorkspaceID(wsID).
		SetChannel(suppression.ChannelEmail).SetDestination("txn@example.com").
		SetReason(suppression.ReasonBounce).Save(ctx)
	require.NoError(t, err)
	d, err = eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"txn@example.com", eligibility.SourceBroadcasts, false)
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
		"evt@example.com", eligibility.SourceBroadcasts, true)
	require.NoError(t, err)
	assert.Equal(t, eligibility.ReasonUnsubscribedEverything, d.Reason)

	src := eligibility.AutomationSource(42)
	_, err = env.DB.Unsubscribe.Create().SetWorkspaceID(wsID).
		SetChannel(unsubscribe.ChannelEmail).SetDestination("src@example.com").
		SetSendingSource(src).Save(ctx)
	require.NoError(t, err)
	// Ineligible for that source...
	d, err = eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail, "src@example.com", src, true)
	require.NoError(t, err)
	assert.Equal(t, eligibility.ReasonUnsubscribedSource, d.Reason)
	// ...but eligible for a different source.
	d, err = eligibility.Check(ctx, env.DB, wsID, eligibility.ChannelEmail,
		"src@example.com", eligibility.SourceBroadcasts, true)
	require.NoError(t, err)
	assert.True(t, d.Eligible)
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
			eligibility.Predicate(eligibility.ChannelEmail, eligibility.SourceBroadcasts)).
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
		"  Mixed@Example.com  ", eligibility.SourceBroadcasts, true)
	require.NoError(t, err)
	assert.False(t, d.Eligible, "input is normalized before lookup")
}
