package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func TestSuspendWorkspaceRecordsAttribution(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	changed, err := service.SuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "cli", "complaint rate 0.9%")
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

	_, err := service.SuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "system", "first")
	require.NoError(t, err)
	changed, err := service.SuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "cli", "second")
	require.NoError(t, err)
	assert.False(t, changed)

	ws := env.DB.Workspace.GetX(ctx, fixtures.AcmeID)
	assert.Equal(t, "system", *ws.SuspendedBy)
	assert.Equal(t, "first", *ws.SuspensionReason)
}

func TestUnsuspendWorkspaceClearsEverything(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	_, err := service.SuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "cli", "abuse")
	require.NoError(t, err)
	changed, err := service.UnsuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "cli")
	require.NoError(t, err)
	assert.True(t, changed)

	ws := env.DB.Workspace.GetX(ctx, fixtures.AcmeID)
	assert.Nil(t, ws.SuspendedAt)
	assert.Nil(t, ws.SuspendedBy)
	assert.Nil(t, ws.SuspensionReason)

	changed, err = service.UnsuspendWorkspace(ctx, env.Bus, fixtures.AcmeID, "cli")
	require.NoError(t, err)
	assert.False(t, changed, "unsuspending a workspace that is not suspended is a no-op")
}

func TestSuspendWorkspaceRequiresAReason(t *testing.T) {
	env := testhelper.Setup(t)

	_, err := service.SuspendWorkspace(context.Background(), env.Bus, fixtures.AcmeID, "cli", "  ")
	require.Error(t, err, "attribution needs a reason: the owner is told why")
}

func TestWorkspaceIDBySlug(t *testing.T) {
	env := testhelper.Setup(t)

	id, err := service.WorkspaceIDBySlug(context.Background(), env.DB, fixtures.AcmeSlug)
	require.NoError(t, err)
	assert.EqualValues(t, fixtures.AcmeID, id)
	_, err = service.WorkspaceIDBySlug(context.Background(), env.DB, "no-such-workspace")
	require.Error(t, err)
}
