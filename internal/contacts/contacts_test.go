package contacts_test

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/contacts"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// Fixtures: workspace acme (1) owns contacts 1 alice@example.com (Alice Smith) and
// 2 bob@example.com. No other workspace owns them.
const (
	acme    = int64(1)
	aliceID = int64(1)
)

func createdEvents(t *testing.T, env *testhelper.TestEnv, contactID int64) int {
	t.Helper()
	var n int
	require.NoError(t, env.SQLDB.QueryRow(
		`SELECT count(*) FROM watermill_domain_events
		   WHERE payload->>'name' = 'contact.created'
		     AND (payload->'data'->>'contactId')::bigint = $1`, contactID,
	).Scan(&n))
	return n
}

func TestCreatePersistsAndPublishesContactCreated(t *testing.T) {
	env := testhelper.Setup(t)
	m := contacts.New(env.Bus)

	c, err := m.Create(context.Background(), acme, contacts.Attributes{
		Email:        lo.ToPtr("  New.Person@Example.com "),
		FirstName:    lo.ToPtr("New"),
		CustomFields: map[string]any{"plan": "pro"},
	})
	require.NoError(t, err)

	assert.Equal(t, "new.person@example.com", lo.FromPtr(c.Email), "email alias key is normalized")
	assert.Equal(t, "New", lo.FromPtr(c.FirstName))
	assert.Equal(t, "pro", c.CustomFields["plan"])
	assert.Equal(t, 1, createdEvents(t, env, c.ID), "contact.created published once, in the same transaction")
}

func TestCreateConflictOnAliasKey(t *testing.T) {
	env := testhelper.Setup(t)
	m := contacts.New(env.Bus)

	// Case and whitespace do not make a different email.
	_, err := m.Create(context.Background(), acme, contacts.Attributes{Email: lo.ToPtr(" ALICE@example.com ")})
	var conflict *contacts.ConflictError
	require.True(t, errors.As(err, &conflict), "got %v", err)
	assert.Equal(t, contacts.FieldEmail, conflict.Field)
	assert.Equal(t, "email already exists", conflict.Message())
}

func TestCreateConflictOnPhone(t *testing.T) {
	env := testhelper.Setup(t)
	m := contacts.New(env.Bus)
	_, err := m.Create(context.Background(), acme, contacts.Attributes{Phone: lo.ToPtr("+15550001")})
	require.NoError(t, err)

	_, err = m.Create(context.Background(), acme, contacts.Attributes{Phone: lo.ToPtr("+15550001")})
	var conflict *contacts.ConflictError
	require.True(t, errors.As(err, &conflict), "got %v", err)
	assert.Equal(t, contacts.FieldPhone, conflict.Field)
}

func TestUpdateChangesOnlyGivenAttributes(t *testing.T) {
	env := testhelper.Setup(t)
	m := contacts.New(env.Bus)

	c, err := m.Update(context.Background(), acme, aliceID, contacts.Attributes{LastName: lo.ToPtr("Jones")})
	require.NoError(t, err)
	assert.Equal(t, "Jones", lo.FromPtr(c.LastName))
	assert.Equal(t, "Alice", lo.FromPtr(c.FirstName), "untouched attribute stays")
	assert.Equal(t, "alice@example.com", lo.FromPtr(c.Email))
	assert.Zero(t, createdEvents(t, env, aliceID), "an update is not a creation")
}

func TestUpdateNotFoundAcrossWorkspaces(t *testing.T) {
	env := testhelper.Setup(t)
	m := contacts.New(env.Bus)

	_, err := m.Update(context.Background(), acme, 999999, contacts.Attributes{})
	assert.ErrorIs(t, err, contacts.ErrNotFound)

	_, err = m.Update(context.Background(), acme+1000, aliceID, contacts.Attributes{LastName: lo.ToPtr("X")})
	assert.ErrorIs(t, err, contacts.ErrNotFound, "another workspace cannot touch the contact")
}

func TestUpdateConflict(t *testing.T) {
	env := testhelper.Setup(t)
	m := contacts.New(env.Bus)

	_, err := m.Update(context.Background(), acme, aliceID, contacts.Attributes{Email: lo.ToPtr("bob@example.com")})
	var conflict *contacts.ConflictError
	require.True(t, errors.As(err, &conflict), "got %v", err)
	assert.Equal(t, contacts.FieldEmail, conflict.Field)
}

func TestUpsertResolvesExistingContactByAliasKey(t *testing.T) {
	env := testhelper.Setup(t)
	m := contacts.New(env.Bus)

	res, err := m.Upsert(context.Background(), acme, contacts.Attributes{
		Email:        lo.ToPtr("Alice@example.com"),
		Phone:        lo.ToPtr("+15550002"),
		FirstName:    lo.ToPtr("Ignored"),
		CustomFields: map[string]any{"plan": "team"},
	})
	require.NoError(t, err)
	assert.False(t, res.Created)
	assert.Equal(t, aliceID, res.Contact.ID)
	assert.Equal(t, "+15550002", lo.FromPtr(res.Contact.Phone), "a missing alias key is filled in")
	assert.Equal(t, "team", res.Contact.CustomFields["plan"])
	assert.Equal(t, "Alice", lo.FromPtr(res.Contact.FirstName), "identity is additive: existing values are not overwritten")
	assert.Zero(t, createdEvents(t, env, aliceID))

	// The newly added phone now resolves the same contact.
	again, err := m.Upsert(context.Background(), acme, contacts.Attributes{Phone: lo.ToPtr("+15550002")})
	require.NoError(t, err)
	assert.Equal(t, aliceID, again.Contact.ID)
}

func TestUpsertCreatesAndPublishesOnce(t *testing.T) {
	env := testhelper.Setup(t)
	m := contacts.New(env.Bus)

	res, err := m.Upsert(context.Background(), acme, contacts.Attributes{SubjectID: lo.ToPtr("user-77"), Email: lo.ToPtr("u77@example.com")})
	require.NoError(t, err)
	assert.True(t, res.Created)
	assert.Equal(t, 1, createdEvents(t, env, res.Contact.ID))

	again, err := m.Upsert(context.Background(), acme, contacts.Attributes{SubjectID: lo.ToPtr("user-77")})
	require.NoError(t, err)
	assert.False(t, again.Created)
	assert.Equal(t, res.Contact.ID, again.Contact.ID)
	assert.Equal(t, 1, createdEvents(t, env, res.Contact.ID), "still published once")
}

func TestUpsertRequiresAnAliasKey(t *testing.T) {
	env := testhelper.Setup(t)
	m := contacts.New(env.Bus)

	_, err := m.Upsert(context.Background(), acme, contacts.Attributes{FirstName: lo.ToPtr("Nobody"), Email: lo.ToPtr("  ")})
	assert.ErrorIs(t, err, contacts.ErrIdentityRequired)
}
