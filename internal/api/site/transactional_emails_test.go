package site_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/transactionalemail"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedTransactional(t *testing.T, db *ent.Client, ws int64, dest string, status transactionalemail.Status) {
	t.Helper()
	_, err := db.TransactionalEmail.Create().
		SetWorkspaceID(ws).
		SetChannel(transactionalemail.ChannelEmail).
		SetDestination(dest).
		SetTemplateID(1).
		SetStatus(status).
		Save(context.Background())
	require.NoError(t, err)
}

// The site list returns the workspace's transactional sends, scoped to the
// workspace and never leaking another tenant's rows.
func TestSiteTransactionalEmailsList(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// acme's own transactional sends come from the fixtures. A second workspace's
	// row is seeded only to prove it is excluded (the cross-tenant negative case).
	ws2, err := env.DB.Workspace.Create().
		SetName("Globex").SetSlug("globex-tx").
		SetCollectKey("globex-tx-ck").SetIngestKey("globex-tx-ik").Save(ctx)
	require.NoError(t, err)
	seedTransactional(t, env.DB, ws2.ID, "leak@example.com", transactionalemail.StatusSent)

	c := siteClient(t, env, "info@1mail.com")
	res, err := c.SiteTransactionalEmailsList(ctx, siteapi.SiteTransactionalEmailsListParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)

	page, ok := res.(*siteapi.SiteTransactionalEmailsListOK)
	require.Truef(t, ok, "got %T", res)
	require.NotEmpty(t, page.Items, "acme has seeded transactional sends")
	for _, it := range page.Items {
		assert.NotEqual(t, "leak@example.com", it.Destination, "cross-workspace row leaked")
	}
}

// The list requires a valid JWT like the rest of the site API.
func TestSiteTransactionalEmailsRequireAuth(t *testing.T) {
	env := testhelper.Setup(t)
	c, err := siteapi.NewClient("http://local/site", noJWT{}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)

	_, err = c.SiteTransactionalEmailsList(context.Background(), siteapi.SiteTransactionalEmailsListParams{WorkspaceSlug: "acme"})
	require.Error(t, err)
}
