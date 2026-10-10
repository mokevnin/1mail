package site_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func readSiteExport(t *testing.T, res siteapi.SiteContactsExportRes) map[string]json.RawMessage {
	t.Helper()
	ok, isOK := res.(*siteapi.SiteContactsExportOKHeaders)
	require.Truef(t, isOK, "got %T", res)
	assert.Contains(t, ok.ContentDisposition, "attachment")
	raw, err := io.ReadAll(ok.Response)
	require.NoError(t, err)
	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &doc), string(raw))
	return doc
}

// Export needs only Workspace access: a plain member (no owner/admin role) can run
// a subject-access request.
func TestSiteContactsExportIsOpenToAnyMember(t *testing.T) {
	env := testhelper.Setup(t)
	member := env.SiteActor(t, fixtures.MemberMaryEmail)

	res, err := member.SiteContactsExport(context.Background(), siteapi.SiteContactsExportParams{
		Slug: fixtures.AcmeSlug,
		ID:   siteapi.NewOptEntityId(siteapi.EntityId("800")),
	})
	require.NoError(t, err)
	doc := readSiteExport(t, res)

	var events []map[string]any
	require.NoError(t, json.Unmarshal(doc["events"], &events))
	assert.Len(t, events, 3)
	for _, key := range []string{"contact", "tags", "visitors", "unsubscribes", "suppressions", "confirmations", "outbound_messages", "broadcast_recipients"} {
		assert.Contains(t, doc, key)
	}
}

func TestSiteContactsExportByEmail(t *testing.T) {
	env := testhelper.Setup(t)
	owner := env.SiteActor(t, fixtures.OwnerJohnEmail)

	res, err := owner.SiteContactsExport(context.Background(), siteapi.SiteContactsExportParams{
		Slug:  fixtures.AcmeSlug,
		Email: siteapi.NewOptEmailAddress(fixtures.ContactExportSubjectEmail),
	})
	require.NoError(t, err)
	doc := readSiteExport(t, res)

	var contact map[string]any
	require.NoError(t, json.Unmarshal(doc["contact"], &contact))
	assert.EqualValues(t, fixtures.ContactExportSubjectID, contact["id"])
}

func TestSiteContactsExportStaysInsideTheWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// A foreign Contact is not found through the caller's own Workspace.
	owner := env.SiteActor(t, fixtures.OwnerJohnEmail)
	res, err := owner.SiteContactsExport(ctx, siteapi.SiteContactsExportParams{
		Slug: fixtures.AcmeSlug,
		ID:   siteapi.NewOptEntityId(siteapi.EntityId("900")),
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteContactsExportNotFound{}, res)

	// A non-member cannot address the Workspace at all.
	outsider := env.SiteActor(t, fixtures.OutsiderOscarEmail)
	res, err = outsider.SiteContactsExport(ctx, siteapi.SiteContactsExportParams{
		Slug: fixtures.AcmeSlug,
		ID:   siteapi.NewOptEntityId(siteapi.EntityId("800")),
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteContactsExportNotFound{}, res)
}
