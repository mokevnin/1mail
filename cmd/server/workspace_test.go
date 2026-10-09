package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOps struct {
	suspended   []string // "slug|by|reason"
	unsuspended []string
	changed     bool
}

func (f *fakeOps) SuspendWorkspace(_ context.Context, slug, by, reason string) (bool, error) {
	f.suspended = append(f.suspended, slug+"|"+by+"|"+reason)
	return f.changed, nil
}

func (f *fakeOps) UnsuspendWorkspace(_ context.Context, slug string) (bool, error) {
	f.unsuspended = append(f.unsuspended, slug)
	return f.changed, nil
}

func TestWorkspaceSuspendJoinsTheReasonWords(t *testing.T) {
	ops := &fakeOps{changed: true}
	var out bytes.Buffer

	require.NoError(t, runWorkspace(context.Background(), ops, []string{"suspend", "acme", "complaint", "rate", "too", "high"}, &out))

	assert.Equal(t, []string{"acme|cli|complaint rate too high"}, ops.suspended)
	assert.Contains(t, out.String(), "suspended")
}

func TestWorkspaceSuspendNeedsASlugAndAReason(t *testing.T) {
	ops := &fakeOps{}
	for _, args := range [][]string{{"suspend"}, {"suspend", "acme"}} {
		err := runWorkspace(context.Background(), ops, args, &bytes.Buffer{})
		require.Error(t, err, "args %v", args)
	}
	assert.Empty(t, ops.suspended)
}

func TestWorkspaceUnsuspendReportsNoChange(t *testing.T) {
	ops := &fakeOps{changed: false}
	var out bytes.Buffer

	require.NoError(t, runWorkspace(context.Background(), ops, []string{"unsuspend", "acme"}, &out))

	assert.Equal(t, []string{"acme"}, ops.unsuspended)
	assert.Contains(t, out.String(), "not suspended")
}

func TestWorkspaceUnknownSubcommand(t *testing.T) {
	require.Error(t, runWorkspace(context.Background(), &fakeOps{}, []string{"delete", "acme"}, &bytes.Buffer{}))
	require.Error(t, runWorkspace(context.Background(), &fakeOps{}, nil, &bytes.Buffer{}))
}
