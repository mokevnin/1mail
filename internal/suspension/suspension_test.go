package suspension_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/sphericon/internal/fixtures"
	"github.com/mokevnin/sphericon/internal/suspension"
	"github.com/mokevnin/sphericon/internal/testhelper"
)

func TestSuspendWorkspaceRecordsAttribution(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	changed, err := suspension.SuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "cli", "complaint rate 0.9%")
	require.NoError(t, err)
	assert.True(t, changed)

	ws := env.DB.Workspace.GetX(ctx, fixtures.AcmeID)
	require.NotNil(t, ws.SuspendedAt)
	require.NotNil(t, ws.SuspendedBy)
	assert.Equal(t, "cli", *ws.SuspendedBy)
	require.NotNil(t, ws.SuspensionReason)
	assert.Equal(t, "complaint rate 0.9%", *ws.SuspensionReason)
}

// Suspending an already-suspended workspace changes nothing, so the first actor and
// reason (the ones the owner was told about) are never overwritten.
func TestSuspendWorkspaceIsIdempotent(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	_, err := suspension.SuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "system", "first")
	require.NoError(t, err)
	changed, err := suspension.SuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "cli", "second")
	require.NoError(t, err)
	assert.False(t, changed)

	ws := env.DB.Workspace.GetX(ctx, fixtures.AcmeID)
	assert.Equal(t, "system", *ws.SuspendedBy)
	assert.Equal(t, "first", *ws.SuspensionReason)
}

func TestUnsuspendWorkspaceClearsEverything(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	_, err := suspension.SuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "cli", "abuse")
	require.NoError(t, err)
	changed, err := suspension.UnsuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "cli")
	require.NoError(t, err)
	assert.True(t, changed)

	ws := env.DB.Workspace.GetX(ctx, fixtures.AcmeID)
	assert.Nil(t, ws.SuspendedAt)
	assert.Nil(t, ws.SuspendedBy)
	assert.Nil(t, ws.SuspensionReason)

	changed, err = suspension.UnsuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "cli")
	require.NoError(t, err)
	assert.False(t, changed, "unsuspending a workspace that is not suspended is a no-op")
}

func TestSuspendWorkspaceRequiresAReason(t *testing.T) {
	env := testhelper.Setup(t)

	_, err := suspension.SuspendWorkspace(context.Background(), env.Bus, fixtures.AcmeID, "cli", "  ")
	require.Error(t, err, "attribution needs a reason: the owner is told why")
}
