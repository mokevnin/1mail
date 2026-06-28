package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSiteTemplatesCRUD(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()
	slug := "acme"

	created, err := c.SiteTemplatesCreate(ctx, &siteapi.SiteCreateEmailTemplateInput{
		Name:       "Welcome",
		Subject:    siteapi.NewOptString("Welcome {{ first_name }}"),
		BodyFormat: siteapi.NewOptSiteEmailBodyFormat(siteapi.SiteEmailBodyFormatMjml),
		BodyHtml:   siteapi.NewOptString("<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>"),
	}, siteapi.SiteTemplatesCreateParams{WorkspaceSlug: slug})
	require.NoError(t, err)
	res, ok := created.(*siteapi.SiteEmailTemplateResource)
	require.Truef(t, ok, "got %T", created)
	assert.Equal(t, "Welcome", res.Name)
	assert.Equal(t, siteapi.SiteEmailBodyFormatMjml, res.BodyFormat)

	list, err := c.SiteTemplatesList(ctx, siteapi.SiteTemplatesListParams{WorkspaceSlug: slug})
	require.NoError(t, err)
	listed, ok := list.(*siteapi.SiteTemplatesListOK)
	require.Truef(t, ok, "got %T", list)
	assert.Equal(t, int32(1), listed.TotalItems)

	updated, err := c.SiteTemplatesUpdate(ctx, &siteapi.SiteUpdateEmailTemplateInput{
		Name: siteapi.NewOptString("Welcome v2"),
	}, siteapi.SiteTemplatesUpdateParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	updRes, ok := updated.(*siteapi.SiteEmailTemplateResource)
	require.Truef(t, ok, "got %T", updated)
	assert.Equal(t, "Welcome v2", updRes.Name)

	del, err := c.SiteTemplatesDelete(ctx, siteapi.SiteTemplatesDeleteParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteTemplatesDeleteNoContent{}, del)
}

// Test-send without a configured sending integration is rejected with 422.
func TestSiteBroadcastsTestSendWithoutIntegration(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	created, err := c.SiteBroadcastsCreate(ctx, &siteapi.SiteCreateBroadcastInput{
		Name: "Preview me", Subject: siteapi.NewOptString("Hi"),
	}, siteapi.SiteBroadcastsCreateParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	b := created.(*siteapi.SiteBroadcastResource)

	out, err := c.SiteBroadcastsTestSend(ctx, &siteapi.SiteTestSendBroadcastInput{
		Email: "qa@test.dev",
	}, siteapi.SiteBroadcastsTestSendParams{WorkspaceSlug: "acme", ID: b.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteBroadcastsTestSendUnprocessableEntity{}, out)
}
