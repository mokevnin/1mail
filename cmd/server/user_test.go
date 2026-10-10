package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUserOps struct {
	reset   []string
	changed bool
}

func (f *fakeUserOps) ResetSecondFactor(_ context.Context, email string) (bool, error) {
	f.reset = append(f.reset, email)
	return f.changed, nil
}

func TestUserResetSecondFactorByEmail(t *testing.T) {
	ops := &fakeUserOps{changed: true}
	var out bytes.Buffer

	require.NoError(t, runUser(context.Background(), ops, []string{"reset-second-factor", "sam@globex.test"}, &out))

	assert.Equal(t, []string{"sam@globex.test"}, ops.reset)
	assert.Contains(t, out.String(), "reset")
}

func TestUserResetSecondFactorReportsNoChange(t *testing.T) {
	ops := &fakeUserOps{changed: false}
	var out bytes.Buffer

	require.NoError(t, runUser(context.Background(), ops, []string{"reset-second-factor", "john@acme.test"}, &out))

	assert.Contains(t, out.String(), "nothing changed")
}

func TestUserCommandNeedsAKnownSubcommandAndAnEmail(t *testing.T) {
	ops := &fakeUserOps{}
	for _, args := range [][]string{nil, {"reset-second-factor"}, {"delete", "sam@globex.test"}} {
		require.Error(t, runUser(context.Background(), ops, args, &bytes.Buffer{}), "args %v", args)
	}
	assert.Empty(t, ops.reset)
}
