package jobs_test

import (
	"context"
	"net"
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
	ok, err := jobs.VerifySendingDomainByID(ctx, env.DB, lookupReturning([]string{dom.DkimPublicKey}, nil), unverifiedDomainID)
	require.NoError(t, err)
	assert.True(t, ok)

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
	ok, err := jobs.VerifySendingDomainByID(ctx, env.DB, lookupReturning(nil, &net.DNSError{IsNotFound: true}), verifiedDomainID)
	require.NoError(t, err)
	assert.False(t, ok)

	reloaded, err := env.DB.SendingDomain.Get(ctx, verifiedDomainID)
	require.NoError(t, err)
	assert.False(t, reloaded.Verified, "a verified domain must flip back when its DKIM DNS is gone")
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

	posNull := indexOf(ids, unverifiedDomainID)
	posChecked := indexOf(ids, verifiedDomainID)
	assert.Less(t, posNull, posChecked, "never-checked domain must be re-checked before an already-checked one")
}

func indexOf(ids []int64, want int64) int {
	for i, id := range ids {
		if id == want {
			return i
		}
	}
	return -1
}

func TestVerifySendingDomainByID_resolverErrorDoesNotChangeState(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	_, err := jobs.VerifySendingDomainByID(ctx, env.DB, lookupReturning(nil, &net.DNSError{IsTemporary: true}), verifiedDomainID)
	require.Error(t, err)

	reloaded, err := env.DB.SendingDomain.Get(ctx, verifiedDomainID)
	require.NoError(t, err)
	assert.True(t, reloaded.Verified, "a transient resolver failure must not flip verified")
}
