package jobs_test

import (
	"context"
	"net"
	"slices"
	"testing"

	"github.com/mokevnin/1mail/internal/jobs"
	"github.com/mokevnin/1mail/internal/sending"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture sending_domains: id 1 mail.acme.com (verified), id 2 news.acme.com (unverified).
const (
	verifiedDomainID   = int64(1)
	unverifiedDomainID = int64(2)
)

func lookupReturning(records []string, err error) sending.TXTLookup {
	return func(context.Context, string) ([]string, error) { return records, err }
}

func TestVerifySendingDomainByID_becomesVerified(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	dom, err := env.DB.SendingDomain.Get(ctx, unverifiedDomainID)
	require.NoError(t, err)
	require.False(t, dom.Verified)

	// DNS now publishes the matching key.
	ok, flipped, err := jobs.VerifySendingDomainByID(ctx, env.DB, lookupReturning([]string{dom.DkimPublicKey}, nil), unverifiedDomainID)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.False(t, flipped, "becoming verified is not a flip-to-unverified")

	reloaded, err := env.DB.SendingDomain.Get(ctx, unverifiedDomainID)
	require.NoError(t, err)
	assert.True(t, reloaded.Verified)
	require.NotNil(t, reloaded.VerifiedAt)
	require.NotNil(t, reloaded.LastCheckedAt)
}

func TestVerifySendingDomainByID_flipsToUnverifiedWhenRecordGone(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// Record disappeared → NXDOMAIN → verifies false, no error.
	ok, flipped, err := jobs.VerifySendingDomainByID(ctx, env.DB, lookupReturning(nil, &net.DNSError{IsNotFound: true}), verifiedDomainID)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.True(t, flipped, "losing verification of a live domain is the notify-worthy flip")

	reloaded, err := env.DB.SendingDomain.Get(ctx, verifiedDomainID)
	require.NoError(t, err)
	assert.False(t, reloaded.Verified, "a verified domain must flip back when its DKIM DNS is gone")
}

// On a verified→unverified flip the workspace owner is emailed (ADR 0010 slice 3).
// Workspace 1's owner is user 1 (info@1mail.com), per the membership fixtures.
func TestNotifySendingDomainUnverified_emailsOwner(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	err := jobs.NotifySendingDomainUnverified(ctx, env.DB, env.SystemMail, verifiedDomainID)
	require.NoError(t, err)

	msgs := env.SystemMail.Messages()
	require.Len(t, msgs, 1)
	assert.Equal(t, "info@1mail.com", msgs[0].To)
	assert.Contains(t, msgs[0].Subject, "mail.acme.com")
}

func TestNotifySendingDomainUnverified_nilSenderIsNoop(t *testing.T) {
	env := testhelper.Setup(t)
	require.NoError(t, jobs.NotifySendingDomainUnverified(context.Background(), env.DB, nil, verifiedDomainID))
}

// DevTXTLookup echoes each domain's own stored key, so a seeded/added domain
// self-verifies in dev without real DNS (the local send-gate escape, ADR 0010).
func TestDevTXTLookup_trustsSeededDomains(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	dom, err := env.DB.SendingDomain.Get(ctx, unverifiedDomainID)
	require.NoError(t, err)
	require.False(t, dom.Verified)

	ok, _, err := jobs.VerifySendingDomainByID(ctx, env.DB, jobs.DevTXTLookup(env.DB), unverifiedDomainID)
	require.NoError(t, err)
	assert.True(t, ok, "dev lookup makes a seeded domain verify")
}

func TestDomainsDueForRecheck_neverCheckedFirst(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// Fixtures: id 1 has last_checked_at set; id 2 (news.acme.com) is never
	// checked (NULL). NULLS FIRST must surface id 2 ahead of id 1.
	ids, err := jobs.DomainsDueForRecheck(ctx, env.DB, 100)
	require.NoError(t, err)
	require.Contains(t, ids, unverifiedDomainID)
	require.Contains(t, ids, verifiedDomainID)

	posNull := slices.Index(ids, unverifiedDomainID)
	posChecked := slices.Index(ids, verifiedDomainID)
	assert.Less(t, posNull, posChecked, "never-checked domain must be re-checked before an already-checked one")
}

func TestVerifySendingDomainByID_resolverErrorDoesNotChangeState(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	_, _, err := jobs.VerifySendingDomainByID(ctx, env.DB, lookupReturning(nil, &net.DNSError{IsTemporary: true}), verifiedDomainID)
	require.Error(t, err)

	reloaded, err := env.DB.SendingDomain.Get(ctx, verifiedDomainID)
	require.NoError(t, err)
	assert.True(t, reloaded.Verified, "a transient resolver failure must not flip verified")
}
