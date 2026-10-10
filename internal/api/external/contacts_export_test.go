package external_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func readExport(t *testing.T, res externalapi.ContactsExportRes) (map[string]json.RawMessage, string) {
	t.Helper()
	ok, isOK := res.(*externalapi.ContactsExportOKHeaders)
	require.Truef(t, isOK, "got %T", res)
	raw, err := io.ReadAll(ok.Response)
	require.NoError(t, err)
	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &doc), string(raw))
	return doc, ok.ContentDisposition
}

func list(t *testing.T, doc map[string]json.RawMessage, key string) []map[string]any {
	t.Helper()
	var rows []map[string]any
	require.NoError(t, json.Unmarshal(doc[key], &rows), key)
	require.NotNil(t, rows, key+" is an array, never null")
	return rows
}

func TestExternalContactsExportByIDReturnsTheFullBundle(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.ExternalScoped(t, "contacts:read")

	res, err := c.ContactsExport(context.Background(), externalapi.ContactsExportParams{
		ID: externalapi.NewOptEntityId(entityIDString(fixtures.ContactExportSubjectID)),
	})
	require.NoError(t, err)
	doc, disposition := readExport(t, res)

	assert.Contains(t, disposition, "attachment")
	var contact map[string]any
	require.NoError(t, json.Unmarshal(doc["contact"], &contact))
	assert.Equal(t, fixtures.ContactExportSubjectEmail, contact["email"])
	assert.Equal(t, "user:export-subject", contact["subject_id"])
	assert.Equal(t, map[string]any{"plan": "pro", "seats": float64(4)}, contact["custom_fields"])

	tags := list(t, doc, "tags")
	assert.ElementsMatch(t, []any{"vip", "newsletter"}, []any{tags[0]["name"], tags[1]["name"]})
	assert.Len(t, tags, 2)
	assert.Len(t, list(t, doc, "visitors"), 2)

	actions := []any{}
	for _, e := range list(t, doc, "events") {
		actions = append(actions, e["action"])
	}
	assert.ElementsMatch(t, []any{"export_subject_custom", "email.sent", "marketing.confirmed"}, actions)

	assert.Len(t, list(t, doc, "unsubscribes"), 1)
	assert.Len(t, list(t, doc, "suppressions"), 1)
	assert.Len(t, list(t, doc, "confirmations"), 1)
	assert.Len(t, list(t, doc, "outbound_messages"), 2, "includes the send to the address that has no Contact link")
	assert.Len(t, list(t, doc, "broadcast_recipients"), 1)

	for _, m := range list(t, doc, "outbound_messages") {
		for _, body := range []string{"body", "html", "text", "rendered_body", "subject"} {
			assert.NotContains(t, m, body, "rendered message content is not exported")
		}
	}
}

func TestExternalContactsExportByEmailMatchesByID(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.ExternalScoped(t, "contacts:read")

	res, err := c.ContactsExport(context.Background(), externalapi.ContactsExportParams{
		Email: externalapi.NewOptEmailAddress("Export.Subject@Example.com"),
	})
	require.NoError(t, err)
	doc, _ := readExport(t, res)

	var contact map[string]any
	require.NoError(t, json.Unmarshal(doc["contact"], &contact))
	assert.EqualValues(t, fixtures.ContactExportSubjectID, contact["id"])
	assert.Len(t, list(t, doc, "events"), 3)
}

func TestExternalContactsExportEmptyMembersAreArrays(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.ExternalScoped(t, "contacts:read")

	res, err := c.ContactsExport(context.Background(), externalapi.ContactsExportParams{
		ID: externalapi.NewOptEntityId(entityIDString(fixtures.ContactBobID)),
	})
	require.NoError(t, err)
	doc, _ := readExport(t, res)
	assert.Empty(t, list(t, doc, "visitors"))
	assert.Empty(t, list(t, doc, "tags"))
	assert.Len(t, list(t, doc, "unsubscribes"), 1, "bob's broadcasts opt-out belongs to his Destination")
}

func TestExternalContactsExportStreamsLargeContacts(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	const extra = 1200 // more than two pages of the keyset reader
	s := env.DB.Scoped(fixtures.AcmeID)
	bulk := make([]*ent.EventScopedCreate, 0, extra)
	for range extra {
		bulk = append(bulk, s.Event().Create().SetContactID(fixtures.ContactExportSubjectID).SetAction("bulk_event"))
	}
	require.NoError(t, s.Event().CreateBulk(bulk...).Exec(ctx))

	c := env.ExternalScoped(t, "contacts:read")
	res, err := c.ContactsExport(ctx, externalapi.ContactsExportParams{
		ID: externalapi.NewOptEntityId(entityIDString(fixtures.ContactExportSubjectID)),
	})
	require.NoError(t, err)
	ok, isOK := res.(*externalapi.ContactsExportOKHeaders)
	require.True(t, isOK)

	// The document stays one valid JSON value across the keyset pages and holds
	// every Event of the Contact.
	var doc map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(ok.Response).Decode(&doc))
	assert.Len(t, list(t, doc, "events"), extra+3)
}

func TestExternalContactsExportNeedsTheReadScope(t *testing.T) {
	env := testhelper.Setup(t)

	res, err := env.ExternalScoped(t, "contacts:write", "events:read").ContactsExport(context.Background(), externalapi.ContactsExportParams{
		ID: externalapi.NewOptEntityId(entityIDString(fixtures.ContactExportSubjectID)),
	})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsExportUnauthorized{}, res)
}

func TestExternalContactsExportIsWorkspaceScoped(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.ExternalScoped(t, "contacts:read")
	ctx := context.Background()

	byID, err := c.ContactsExport(ctx, externalapi.ContactsExportParams{
		ID: externalapi.NewOptEntityId(entityIDString(fixtures.ContactGlobexID)),
	})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsExportNotFound{}, byID)

	byEmail, err := c.ContactsExport(ctx, externalapi.ContactsExportParams{
		Email: externalapi.NewOptEmailAddress(fixtures.ContactGlobexEmail),
	})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsExportNotFound{}, byEmail)
}

func TestExternalContactsExportNeedsExactlyOneIdentifier(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.ExternalScoped(t, "contacts:read")
	ctx := context.Background()

	none, err := c.ContactsExport(ctx, externalapi.ContactsExportParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsExportBadRequest{}, none)

	both, err := c.ContactsExport(ctx, externalapi.ContactsExportParams{
		ID:    externalapi.NewOptEntityId(entityIDString(fixtures.ContactExportSubjectID)),
		Email: externalapi.NewOptEmailAddress(fixtures.ContactExportSubjectEmail),
	})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsExportBadRequest{}, both)
}
