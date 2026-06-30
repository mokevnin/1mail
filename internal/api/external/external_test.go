package external_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-faster/jx"
	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectedRow is an outbox row decoded back into its CollectedEvent.
type collectedRow struct {
	envelope events.Envelope
	event    *events.CollectedEvent
}

// outboxCollected reads every domain-event outbox row and decodes it as a
// CollectedEvent. External ingest now publishes through the bus (the persist
// subscriber writes the Event row asynchronously; the router isn't run under
// txdb), so the outbox is where the assertions land.
func outboxCollected(t *testing.T, env *testhelper.TestEnv) []collectedRow {
	t.Helper()
	rows, err := env.SQLDB.Query(`SELECT payload FROM watermill_domain_events ORDER BY "offset"`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var out []collectedRow
	for rows.Next() {
		var payload []byte
		require.NoError(t, rows.Scan(&payload))
		var envlp events.Envelope
		require.NoError(t, json.Unmarshal(payload, &envlp))
		require.Equal(t, events.NameCollected, envlp.Name)
		decoded, err := events.Decode(envlp)
		require.NoError(t, err)
		ce, ok := decoded.(*events.CollectedEvent)
		require.Truef(t, ok, "got %T", decoded)
		out = append(out, collectedRow{envelope: envlp, event: ce})
	}
	require.NoError(t, rows.Err())
	return out
}

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
		SetWorkspaceID(1).
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

// TestExternalEventsCreate verifies a batch ingest persists events scoped to the
// token's workspace with all optional fields mapped — querying the DB afterwards
// is what catches the NOT-NULL workspace_id bug (a bare 204 assertion would not).
func TestExternalEventsCreate(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"events:write"}))
	ctx := context.Background()

	occurred := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	res, err := c.EventsCreate(ctx, &externalapi.RecordEventsInput{
		Events: []externalapi.EventInput{
			{
				SubjectId:  "user:dave@example.com",
				Action:     "signup",
				Email:      externalapi.NewOptNilEmailAddress("dave@example.com"),
				Prospect:   externalapi.NewOptNilBool(true),
				OccurredAt: externalapi.NewOptNilTimestamp(externalapi.Timestamp(occurred)),
				Properties: externalapi.NewOptNilEventInputProperties(externalapi.EventInputProperties{
					"plan":   jx.Raw(`"pro"`),
					"amount": jx.Raw(`42`),
				}),
			},
			{SubjectId: "user:erin@example.com", Action: "login"},
		},
	})
	require.NoError(t, err)
	require.IsType(t, &externalapi.EventsCreateNoContent{}, res)

	// Both events are published to the outbox as CollectedEvents, scoped to the
	// token's workspace, with every optional field carried as-is.
	byID := map[string]collectedRow{}
	rows := outboxCollected(t, env)
	require.Len(t, rows, 2)
	for _, r := range rows {
		assert.EqualValues(t, 1, r.envelope.WorkspaceID) // token belongs to workspace 1
		byID[r.event.SubjectID] = r
	}

	dave := byID["user:dave@example.com"].event
	require.NotNil(t, dave)
	assert.Equal(t, "signup", dave.Action)
	assert.Equal(t, "dave@example.com", dave.Email)
	require.NotNil(t, dave.Prospect)
	assert.True(t, *dave.Prospect)
	assert.True(t, dave.OccurredAt.Equal(occurred))
	assert.Equal(t, "pro", dave.Properties["plan"])
	assert.EqualValues(t, 42, dave.Properties["amount"])

	erin := byID["user:erin@example.com"].event
	require.NotNil(t, erin)
	assert.Equal(t, "login", erin.Action)
}

// TestExternalEventsScopes verifies events:write/read scopes are enforced.
func TestExternalEventsScopes(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// events:read cannot write.
	ro := client(t, env, seedToken(t, env.DB, []string{"events:read"}))
	createRes, err := ro.EventsCreate(ctx, &externalapi.RecordEventsInput{
		Events: []externalapi.EventInput{{SubjectId: "s", Action: "a"}},
	})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.EventsCreateUnauthorized{}, createRes)

	// events:write cannot read actions.
	wo := client(t, env, seedToken(t, env.DB, []string{"events:write"}))
	listRes, err := wo.EventActionsList(ctx, externalapi.EventActionsListParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.EventActionsListUnauthorized{}, listRes)
}

// TestExternalEventActionsList verifies distinct actions for the token's
// workspace are returned, sorted, with correct pagination totals. The two
// fixture events (page_view, purchase) belong to workspace 1.
func TestExternalEventActionsList(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"events:read"}))

	res, err := c.EventActionsList(context.Background(), externalapi.EventActionsListParams{})
	require.NoError(t, err)
	ok, isOK := res.(*externalapi.EventActionsListOK)
	require.Truef(t, isOK, "got %T", res)

	assert.Equal(t, int32(2), ok.TotalItems)
	actions := make([]string, len(ok.Items))
	for i, item := range ok.Items {
		actions[i] = item.Action
	}
	assert.Equal(t, []string{"page_view", "purchase"}, actions)
}

// TestExternalEventsWorkspaceIsolation verifies event reads and writes are
// scoped to the token's workspace and never leak across tenants.
func TestExternalEventsWorkspaceIsolation(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// A second workspace with an event carrying an action unique to it.
	ws2, err := env.DB.Workspace.Create().
		SetName("Globex").SetSlug("globex").SetCollectKey("globex-collect-key").
		SetIngestKey("globex-ingest-key").
		Save(ctx)
	require.NoError(t, err)
	_, err = env.DB.Event.Create().
		SetWorkspaceID(ws2.ID).
		SetSubjectID("user:other@example.com").
		SetAction("ws2_only_action").
		Save(ctx)
	require.NoError(t, err)

	// seedToken always binds to workspace 1.
	c := client(t, env, seedToken(t, env.DB, []string{"events:read", "events:write"}))

	// Listing for workspace 1 must not surface workspace 2's action.
	res, err := c.EventActionsList(ctx, externalapi.EventActionsListParams{})
	require.NoError(t, err)
	ok := res.(*externalapi.EventActionsListOK)
	for _, item := range ok.Items {
		assert.NotEqual(t, "ws2_only_action", item.Action)
	}
	assert.Equal(t, int32(2), ok.TotalItems)

	// Writes land in workspace 1, not the token-agnostic global pool.
	_, err = c.EventsCreate(ctx, &externalapi.RecordEventsInput{
		Events: []externalapi.EventInput{{SubjectId: "user:fred@example.com", Action: "ws1_write"}},
	})
	require.NoError(t, err)
	rows := outboxCollected(t, env)
	require.Len(t, rows, 1)
	assert.Equal(t, "user:fred@example.com", rows[0].event.SubjectID)
	assert.EqualValues(t, 1, rows[0].envelope.WorkspaceID)
}

func TestExternalRequestValidation(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"contacts:write"}))

	// Invalid email → ogen validation rejects with an undocumented 400, which
	// the typed client surfaces as an error.
	_, err := c.ContactsCreate(context.Background(), &externalapi.CreateContactInput{Email: "not-an-email"})
	require.Error(t, err)
}
