package jobs_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/jobs"
	"github.com/mokevnin/1mail/internal/suspension"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// The owner is told that sending is frozen, why, and who did it, through the
// platform's system sender — which must keep working while the workspace itself is
// suspended (ADR 0007, 0015).
func TestNotifyWorkspaceSuspendedEmailsOwners(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	_, err := suspension.SuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "system", "complaint rate above 0.3%")
	require.NoError(t, err)

	require.NoError(t, jobs.NotifyWorkspaceSuspended(ctx, env.DB, env.SystemMail, fixtures.AcmeID))

	msgs := env.SystemMail.Messages()
	require.Len(t, msgs, 1)
	assert.Equal(t, fixtures.OwnerJohnEmail, msgs[0].To, "the workspace owner")
	assert.Contains(t, msgs[0].Subject, "Acme")
	assert.Contains(t, msgs[0].Text, "complaint rate above 0.3%", "the reason is stated")
	assert.Contains(t, msgs[0].Text, "system", "who set it is stated")
}

func TestNotifyWorkspaceSuspendedNilSenderIsNoop(t *testing.T) {
	env := testhelper.Setup(t)
	require.NoError(t, jobs.NotifyWorkspaceSuspended(context.Background(), env.DB, nil, fixtures.AcmeID))
}
