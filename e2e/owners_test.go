//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/accounts"
)

// Two owners demoted at the same moment: without the row lock both would count two
// owners and succeed, leaving the Workspace ownerless. It cannot run under the
// transaction-per-test harness, where both would share one transaction.
func TestConcurrentOwnerDemotionsKeepOneOwner(t *testing.T) {
	t.Parallel()
	w := env.NewWorkspace(t)
	owners := w.AddOwner()
	require.Len(t, owners, 2)

	errs := w.DemoteConcurrently(owners...)

	failed := 0
	for _, err := range errs {
		if err != nil {
			assert.ErrorIs(t, err, accounts.ErrLastOwner)
			failed++
		}
	}
	assert.Equal(t, 1, failed, "exactly one demotion is refused")
	assert.Len(t, w.OwnerMembershipIDs(), 1, "the Workspace keeps an owner")
}
