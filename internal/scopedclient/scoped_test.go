package scopedclient_test

import (
	"errors"
	"testing"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/apitoken"
	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/ent/tag"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopedOverTransaction(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := t.Context()

	t.Run("commit keeps the write", func(t *testing.T) {
		tx, err := env.DB.Tx(ctx)
		require.NoError(t, err)
		created, err := tx.Client().Scoped(acme).Tag().Create().
			SetName("via-tx").
			AddContactIDs(fixtures.ContactAliceID).
			Save(ctx)
		require.NoError(t, err)
		assert.Equal(t, acme, created.WorkspaceID)
		require.NoError(t, tx.Commit())

		got, err := env.DB.Scoped(acme).Tag().Get(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "via-tx", got.Name)
	})

	t.Run("a foreign reference is refused inside the transaction", func(t *testing.T) {
		tx, err := env.DB.Tx(ctx)
		require.NoError(t, err)
		_, err = tx.Client().Scoped(acme).Tag().Create().
			SetName("via-tx-bad").
			AddContactIDs(fixtures.ContactGlobexID).
			Save(ctx)
		require.ErrorIs(t, err, ent.ErrNotInWorkspace)
		require.NoError(t, tx.Rollback())
	})

	t.Run("rollback drops the write", func(t *testing.T) {
		tx, err := env.DB.Tx(ctx)
		require.NoError(t, err)
		_, err = tx.Client().Scoped(acme).Tag().Create().SetName("via-tx-rolled-back").Save(ctx)
		require.NoError(t, err)
		require.NoError(t, tx.Rollback())

		exists, err := env.DB.Scoped(acme).Tag().Query().Where(tag.Name("via-tx-rolled-back")).Exist(ctx)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestScopedOverEventsBusTx(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := t.Context()

	err := env.Bus.WithinTx(ctx, func(c *ent.Client, _ events.Publisher) error {
		_, err := c.Scoped(acme).Tag().Create().
			SetName("via-bus").
			AddContactIDs(fixtures.ContactAliceID).
			Save(ctx)
		return err
	})
	require.NoError(t, err)
	exists, err := env.DB.Scoped(acme).Tag().Query().Where(tag.Name("via-bus")).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, exists)

	err = env.Bus.WithinTx(ctx, func(c *ent.Client, _ events.Publisher) error {
		_, err := c.Scoped(acme).Tag().Create().
			SetName("via-bus-bad").
			AddContactIDs(fixtures.ContactGlobexID).
			Save(ctx)
		return err
	})
	require.ErrorIs(t, err, ent.ErrNotInWorkspace)

	// The scoped read over the bus's client is confined too.
	err = env.Bus.WithinTx(ctx, func(c *ent.Client, _ events.Publisher) error {
		_, err := c.Scoped(acme).Tag().Get(ctx, fixtures.TagGlobexID)
		require.True(t, ent.IsNotFound(err))
		return nil
	})
	require.NoError(t, err)
}

func TestScopedConditionalBulkUpdate(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := t.Context()
	acmeScope := env.DB.Scoped(acme)

	t.Run("a claim wins once", func(t *testing.T) {
		claim := func() int {
			n, err := acmeScope.Broadcast().Update().
				Where(broadcast.ID(fixtures.BroadcastScheduledID), broadcast.StatusEQ(broadcast.StatusScheduled)).
				SetStatus(broadcast.StatusSending).
				Save(ctx)
			require.NoError(t, err)
			return n
		}
		assert.Equal(t, 1, claim())
		assert.Equal(t, 0, claim(), "the fence stops the second claim")
	})

	t.Run("a foreign row is never claimed", func(t *testing.T) {
		n, err := acmeScope.Broadcast().Update().
			Where(broadcast.ID(fixtures.BroadcastGlobexID)).
			SetStatus(broadcast.StatusSending).
			Save(ctx)
		require.NoError(t, err)
		assert.Zero(t, n)
		got, err := env.DB.Broadcast.Get(ctx, fixtures.BroadcastGlobexID)
		require.NoError(t, err)
		assert.NotEqual(t, broadcast.StatusSending, got.Status)
	})

	t.Run("update-by-id carries an extra fence", func(t *testing.T) {
		err := acmeScope.Broadcast().UpdateOneID(fixtures.BroadcastDraftID).
			Where(broadcast.StatusEQ(broadcast.StatusSent)).
			SetName("fenced").
			Exec(ctx)
		assert.True(t, ent.IsNotFound(err))
	})

	t.Run("a reference set by a bulk update is verified", func(t *testing.T) {
		err := acmeScope.Broadcast().Update().
			Where(broadcast.ID(fixtures.BroadcastDraftID)).
			SetSegmentID(fixtures.SegmentGlobexID).
			Exec(ctx)
		assert.ErrorIs(t, err, ent.ErrNotInWorkspace)
	})
}

func TestScopedConditionalBulkDelete(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := t.Context()

	n, err := env.DB.Scoped(acme).Tag().Delete().Where(tag.NameNEQ("")).Exec(ctx)
	require.NoError(t, err)
	assert.Positive(t, n)

	left, err := env.DB.Tag.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, left, 1, "only the foreign tag is left")
	assert.Equal(t, globex, left[0].WorkspaceID)
}

func TestScopedUpsert(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := t.Context()
	acmeScope := env.DB.Scoped(acme)

	t.Run("ignore converges on one row", func(t *testing.T) {
		for range 2 {
			require.NoError(t, acmeScope.Tag().Create().SetName("upserted").
				OnConflictColumns(tag.FieldName, tag.FieldWorkspaceID).Ignore().Exec(ctx))
		}
		n, err := acmeScope.Tag().Query().Where(tag.Name("upserted")).Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, n)
	})

	t.Run("update new values on the unique key", func(t *testing.T) {
		id, err := acmeScope.Contact().Create().SetEmail(fixtures.ContactAliceEmail).SetFirstName("Alicia").
			OnConflictColumns(contact.FieldEmail, contact.FieldWorkspaceID).
			Update(func(u *ent.ContactScopedUpsert) { u.UpdateFirstName() }).
			ID(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(fixtures.ContactAliceID), id)
		got, err := acmeScope.Contact().Get(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, got.FirstName)
		assert.Equal(t, "Alicia", *got.FirstName)
	})

	t.Run("a foreign reference is refused", func(t *testing.T) {
		err := acmeScope.Broadcast().Create().
			SetName("upsert").
			SetSegmentID(fixtures.SegmentGlobexID).
			OnConflictColumns(broadcast.FieldID).
			Ignore().
			Exec(ctx)
		assert.ErrorIs(t, err, ent.ErrNotInWorkspace)
	})

	t.Run("a conflicting row of another Workspace is not updated", func(t *testing.T) {
		// api_tokens.prefix is unique across Workspaces, so this conflicts with Globex's row.
		err := acmeScope.ApiToken().Create().
			SetName("hijack").
			SetPrefix(fixtures.TokenGlobexPrefix).
			SetSecretHash("x").
			OnConflictColumns(apitoken.FieldPrefix).
			UpdateNewValues().
			Exec(ctx)
		// Whether the database reports the unmatched row as an error is its business;
		// the foreign row must be exactly as it was.
		_ = err
		got, err := env.DB.ApiToken.Get(ctx, fixtures.TokenGlobexID)
		require.NoError(t, err)
		assert.Equal(t, fixtures.TokenGlobexName, got.Name)
		assert.Equal(t, globex, got.WorkspaceID)
	})
}

func TestScopedCreateBulk(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := t.Context()
	acmeScope := env.DB.Scoped(acme)
	br := acmeScope.BroadcastRecipient()

	newRecipient := func(contactID int64) *ent.BroadcastRecipientScopedCreate {
		return br.Create().SetBroadcastID(fixtures.BroadcastDraftID).SetContactID(contactID)
	}

	t.Run("one foreign reference refuses the whole batch", func(t *testing.T) {
		err := br.CreateBulk(newRecipient(fixtures.ContactAliceID), newRecipient(fixtures.ContactGlobexID)).Exec(ctx)
		require.ErrorIs(t, err, ent.ErrNotInWorkspace)
		n, err := br.Query().Where(broadcastrecipient.BroadcastID(fixtures.BroadcastDraftID)).Count(ctx)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("creates the rows in the Workspace, idempotently with on-conflict ignore", func(t *testing.T) {
		for range 2 {
			require.NoError(t, br.CreateBulk(
				newRecipient(fixtures.ContactAliceID),
				newRecipient(fixtures.ContactBobID),
			).OnConflictColumns(broadcastrecipient.FieldBroadcastID, broadcastrecipient.FieldContactID).Ignore().Exec(ctx))
		}
		rows, err := br.Query().Where(broadcastrecipient.BroadcastID(fixtures.BroadcastDraftID)).All(ctx)
		require.NoError(t, err)
		require.Len(t, rows, 2)
		for _, r := range rows {
			assert.Equal(t, acme, r.WorkspaceID)
		}
	})

	t.Run("a builder of another scope is refused", func(t *testing.T) {
		other := env.DB.Scoped(globex).BroadcastRecipient().Create().
			SetBroadcastID(fixtures.BroadcastGlobexID).SetContactID(fixtures.ContactGlobexID)
		err := br.CreateBulk(other).Exec(ctx)
		assert.ErrorIs(t, err, ent.ErrNotInWorkspace)
	})
}

func TestScopedGroupBy(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := t.Context()

	actions, err := env.DB.Scoped(acme).Event().Query().GroupBy(event.FieldAction).Strings(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, actions)
	assert.NotContains(t, actions, fixtures.EventGlobexAction)

	foreign, err := env.DB.Scoped(globex).Event().Query().GroupBy(event.FieldAction).Strings(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{fixtures.EventGlobexAction}, foreign)
}

func TestScopedNotInWorkspaceIsTyped(t *testing.T) {
	env := testhelper.Setup(t)
	_, err := env.DB.Scoped(acme).Tag().Create().SetName("x").AddContactIDs(fixtures.ContactGlobexID).Save(t.Context())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ent.ErrNotInWorkspace))
}
