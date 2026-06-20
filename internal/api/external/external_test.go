package external_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedToken inserts an ApiToken with the given scopes and returns the plaintext
// bearer value (omtk_<prefix>_<secret>), exercising the real token crypto.
func seedToken(t *testing.T, db *ent.Client, scopes []string) string {
	t.Helper()
	prefix, err := service.GenerateTokenPrefix()
	require.NoError(t, err)
	secret, err := service.GenerateTokenSecret()
	require.NoError(t, err)
	hash, err := service.HashTokenSecret(secret)
	require.NoError(t, err)

	_, err = db.ApiToken.Create().
		SetName("test-token").SetPrefix(prefix).SetSecretHash(hash).SetScopes(scopes).
		Save(context.Background())
	require.NoError(t, err)
	return service.TokenValue(prefix, secret)
}

// staticToken is the client-side SecuritySource (supplies the bearer token).
type staticToken struct{ token string }

func (s staticToken) BearerAuth(context.Context, externalapi.OperationName) (externalapi.BearerAuth, error) {
	return externalapi.BearerAuth{Token: s.token}, nil
}

// client returns the generated typed client wired to dispatch in-memory to the
// server (no socket). It builds URLs and encodes/decodes DTOs itself.
func client(t *testing.T, env *testhelper.TestEnv, token string) *externalapi.Client {
	t.Helper()
	c, err := externalapi.NewClient("http://local/api", staticToken{token}, externalapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)
	return c
}

func TestExternalContactsCRUD(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"contacts:read", "contacts:write"}))
	ctx := context.Background()

	list, err := c.ContactsList(ctx, externalapi.ContactsListParams{})
	require.NoError(t, err)
	listed, ok := list.(*externalapi.ContactsListOK)
	require.Truef(t, ok, "got %T", list)
	assert.Equal(t, int32(3), listed.TotalItems)

	created, err := c.ContactsCreate(ctx, &externalapi.CreateContactInput{
		Email:     "new@example.com",
		FirstName: externalapi.NewOptNilString("New"),
	})
	require.NoError(t, err)
	contact, ok := created.(*externalapi.ContactResource)
	require.Truef(t, ok, "got %T", created)
	assert.Equal(t, externalapi.EmailAddress("new@example.com"), contact.Email)

	got, err := c.ContactsGet(ctx, externalapi.ContactsGetParams{ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactResource{}, got)

	missing, err := c.ContactsGet(ctx, externalapi.ContactsGetParams{ID: "999999"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsGetNotFound{}, missing)

	deleted, err := c.ContactsDelete(ctx, externalapi.ContactsDeleteParams{ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsDeleteNoContent{}, deleted)
}

// Isolated: the unique violation aborts this test's transaction.
func TestExternalContactsConflict(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"contacts:write"}))

	res, err := c.ContactsCreate(context.Background(), &externalapi.CreateContactInput{Email: "alice@example.com"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsCreateConflict{}, res)
}

func TestExternalAuthAndScopes(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// No token / bad token → 401 Unauthorized variant.
	res, err := client(t, env, "").ContactsList(ctx, externalapi.ContactsListParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsListUnauthorized{}, res)

	res, err = client(t, env, "omtk_deadbeef_invalidsecret").ContactsList(ctx, externalapi.ContactsListParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsListUnauthorized{}, res)

	// Read-only token: can read, cannot create.
	ro := client(t, env, seedToken(t, env.DB, []string{"contacts:read"}))
	createRes, err := ro.ContactsCreate(ctx, &externalapi.CreateContactInput{Email: "x@example.com"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsCreateUnauthorized{}, createRes)

	listRes, err := ro.ContactsList(ctx, externalapi.ContactsListParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsListOK{}, listRes)
}

func TestExternalRequestValidation(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"contacts:write"}))

	// Invalid email → ogen validation rejects with an undocumented 400, which
	// the typed client surfaces as an error.
	_, err := c.ContactsCreate(context.Background(), &externalapi.CreateContactInput{Email: "not-an-email"})
	require.Error(t, err)
}
