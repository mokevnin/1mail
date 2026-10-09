package events_test

// PROTOTYPE (branch prototype/scoped-client-tag): throwaway, answers ADR 0017's form fork.

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const acme, globex int64 = 1, 2

func TestPrototypeScopedForms(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	foreign := env.DB.Contact.Create().SetEmail("g@globex.io").SetWorkspaceID(globex).SaveX(ctx)

	// --- Form A ---
	a := env.DB.ScopedA(acme).Tag()
	_, err := a.Create().SetName("a-ok").AddContactIDs(fixtures.ContactAliceID).Save(ctx)
	require.NoError(t, err)
	_, err = a.Create().SetName("a-bad").AddContactIDs(foreign.ID).Save(ctx)
	assert.ErrorIs(t, err, ent.ErrNotInWorkspace, "A: foreign edge id caught by hook at Save")
	_, err = a.Create().SetName("a-ws").SetWorkspaceID(globex).Save(ctx) // compiles: leak at type level
	assert.ErrorIs(t, err, ent.ErrNotInWorkspace, "A: SetWorkspaceID is callable, caught only at runtime")
	assert.False(t, a.Query().Where().ExistX(ctx) == false)
	_, err = a.UpdateOneID(900).SetName("x").Save(ctx)
	assert.True(t, ent.IsNotFound(err), "A: foreign tag not updatable")
	_, err = a.Query().Where().All(ctx)
	require.NoError(t, err)
	for _, tg := range a.Query().AllX(ctx) {
		assert.Equal(t, acme, tg.WorkspaceID)
	}

	// --- Form B ---
	b := env.DB.ScopedB(acme).Tag()
	_, err = b.Create().SetName("b-ok").AddContactIDs(fixtures.ContactAliceID).Save(ctx)
	require.NoError(t, err)
	_, err = b.Create().SetName("b-bad").AddContactIDs(foreign.ID).Save(ctx)
	assert.ErrorIs(t, err, ent.ErrNotInWorkspace, "B: foreign edge id caught in Save")
	// b.Create().SetWorkspaceID(...) does not compile: no such method on TagScopedCreate.
	_, err = b.UpdateOneID(900).SetName("x").Save(ctx)
	assert.True(t, ent.IsNotFound(err), "B: foreign tag not updatable")
	assert.Error(t, b.DeleteOneID(900).Exec(ctx), "B: foreign tag not deletable")

	// Leaks common to both: the returned entity and raw traversal ignore scope.
	own := b.Query().FirstX(ctx)
	moved, err := own.Update().SetWorkspaceID(globex).Save(ctx) // bypasses Scoped entirely
	require.NoError(t, err)
	assert.Equal(t, globex, moved.WorkspaceID, "entity.Update() escapes both forms")
}

func TestPrototypeScopedOverTxAndBus(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	tx, err := env.DB.Tx(ctx) // real *ent.Tx
	require.NoError(t, err)
	_, err = tx.Client().ScopedB(acme).Tag().Create().SetName("via-tx").Save(ctx)
	require.NoError(t, err)
	_, err = tx.Client().ScopedA(acme).Tag().Create().SetName("via-tx-a").Save(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	err = env.Bus.WithinTx(ctx, func(c *ent.Client, _ events.Publisher) error {
		_, err := c.ScopedB(acme).Tag().Create().SetName("via-bus").Save(ctx)
		return err
	})
	require.NoError(t, err)
	assert.True(t, env.DB.ScopedB(acme).Tag().Query().Where().ExistX(ctx))
}
