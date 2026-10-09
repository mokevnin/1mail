package external_test

import (
	"context"
	"strconv"
	"testing"

	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func entityIDString(id int64) externalapi.EntityId {
	return externalapi.EntityId(strconv.FormatInt(id, 10))
}

func TestExternalTemplatesCRUD(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"templates:read", "templates:write"}))
	ctx := context.Background()

	list, err := c.TemplatesList(ctx, externalapi.TemplatesListParams{})
	require.NoError(t, err)
	page, ok := list.(*externalapi.TemplatesListOK)
	require.Truef(t, ok, "got %T", list)
	names := map[string]bool{}
	for _, it := range page.Items {
		names[it.Name] = true
	}
	assert.True(t, names["Welcome"], "fixture template 1 is listed")

	got, err := c.TemplatesGet(ctx, externalapi.TemplatesGetParams{ID: "1"})
	require.NoError(t, err)
	tpl, ok := got.(*externalapi.TemplateResource)
	require.Truef(t, ok, "got %T", got)
	assert.Equal(t, "Welcome", tpl.Name)
	assert.Contains(t, tpl.Body, "<mjml>")

	created, err := c.TemplatesCreate(ctx, &externalapi.CreateTemplateInput{
		Name:    "Win-back",
		Subject: externalapi.NewOptString("We miss you"),
	})
	require.NoError(t, err)
	res, ok := created.(*externalapi.TemplateResource)
	require.Truef(t, ok, "got %T", created)
	assert.Equal(t, "We miss you", res.Subject)

	upd, err := c.TemplatesUpdate(ctx, &externalapi.UpdateTemplateInput{Name: externalapi.NewOptString("Win-back v2")},
		externalapi.TemplatesUpdateParams{ID: res.ID})
	require.NoError(t, err)
	updated, ok := upd.(*externalapi.TemplateResource)
	require.Truef(t, ok, "got %T", upd)
	assert.Equal(t, "Win-back v2", updated.Name)
	assert.Equal(t, "We miss you", updated.Subject, "unset fields are kept")

	del, err := c.TemplatesDelete(ctx, externalapi.TemplatesDeleteParams{ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesDeleteNoContent{}, del)

	missing, err := c.TemplatesGet(ctx, externalapi.TemplatesGetParams{ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesGetNotFound{}, missing)
}

func TestExternalTemplatesScopes(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	res, err := client(t, env, "").TemplatesList(ctx, externalapi.TemplatesListParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesListUnauthorized{}, res)

	// A contacts-only token reaches no template operation.
	other := client(t, env, seedToken(t, env.DB, []string{"contacts:read", "contacts:write"}))
	l, err := other.TemplatesList(ctx, externalapi.TemplatesListParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesListUnauthorized{}, l)

	// Read-only token cannot write.
	ro := client(t, env, seedToken(t, env.DB, []string{"templates:read"}))
	g, err := ro.TemplatesGet(ctx, externalapi.TemplatesGetParams{ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplateResource{}, g)
	cr, err := ro.TemplatesCreate(ctx, &externalapi.CreateTemplateInput{Name: "x"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesCreateUnauthorized{}, cr)
	up, err := ro.TemplatesUpdate(ctx, &externalapi.UpdateTemplateInput{}, externalapi.TemplatesUpdateParams{ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesUpdateUnauthorized{}, up)
	de, err := ro.TemplatesDelete(ctx, externalapi.TemplatesDeleteParams{ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesDeleteUnauthorized{}, de)
}

func TestExternalTemplatesAreWorkspaceScoped(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// Incidental second tenant, only to prove isolation.
	ws2, err := env.DB.Workspace.Create().SetName("Globex").SetSlug("globex-tpl").
		SetCollectKey("globex-tpl-ck").SetIngestKey("globex-tpl-ik").Save(ctx)
	require.NoError(t, err)
	foreign, err := env.DB.EmailTemplate.Create().SetWorkspaceID(ws2.ID).SetName("Foreign").Save(ctx)
	require.NoError(t, err)
	id := entityIDString(foreign.ID)

	c := client(t, env, seedToken(t, env.DB, []string{"templates:read", "templates:write"}))

	list, err := c.TemplatesList(ctx, externalapi.TemplatesListParams{})
	require.NoError(t, err)
	for _, it := range list.(*externalapi.TemplatesListOK).Items {
		assert.NotEqual(t, "Foreign", it.Name)
	}
	g, err := c.TemplatesGet(ctx, externalapi.TemplatesGetParams{ID: id})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesGetNotFound{}, g)
	u, err := c.TemplatesUpdate(ctx, &externalapi.UpdateTemplateInput{Name: externalapi.NewOptString("hijack")}, externalapi.TemplatesUpdateParams{ID: id})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesUpdateNotFound{}, u)
	d, err := c.TemplatesDelete(ctx, externalapi.TemplatesDeleteParams{ID: id})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesDeleteNotFound{}, d)
}
