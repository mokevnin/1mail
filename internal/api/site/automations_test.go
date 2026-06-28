package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSiteAutomationsCRUDAndActivation(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()
	slug := "acme"

	created, err := c.SiteAutomationsCreate(ctx, &siteapi.SiteCreateAutomationInput{
		Name:         "Welcome series",
		TriggerEvent: "contact.created",
		Definition:   siteapi.NewOptString(`[{"type":"email","subject":"Hi","body":"<mjml></mjml>"}]`),
	}, siteapi.SiteAutomationsCreateParams{WorkspaceSlug: slug})
	require.NoError(t, err)
	res, ok := created.(*siteapi.SiteAutomationResource)
	require.Truef(t, ok, "got %T", created)
	assert.Equal(t, "Welcome series", res.Name)
	assert.Equal(t, siteapi.SiteAutomationStatusDraft, res.Status)
	assert.Equal(t, "contact.created", res.TriggerEvent)

	// Activate / deactivate flip the status.
	act, err := c.SiteAutomationsActivate(ctx, siteapi.SiteAutomationsActivateParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	actRes, ok := act.(*siteapi.SiteAutomationResource)
	require.Truef(t, ok, "got %T", act)
	assert.Equal(t, siteapi.SiteAutomationStatusActive, actRes.Status)

	deact, err := c.SiteAutomationsDeactivate(ctx, siteapi.SiteAutomationsDeactivateParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	deactRes, ok := deact.(*siteapi.SiteAutomationResource)
	require.Truef(t, ok, "got %T", deact)
	assert.Equal(t, siteapi.SiteAutomationStatusDraft, deactRes.Status)

	// List shows it.
	list, err := c.SiteAutomationsList(ctx, siteapi.SiteAutomationsListParams{WorkspaceSlug: slug})
	require.NoError(t, err)
	listed, ok := list.(*siteapi.SiteAutomationsListOK)
	require.Truef(t, ok, "got %T", list)
	assert.Equal(t, int32(1), listed.TotalItems)

	// Update renames.
	upd, err := c.SiteAutomationsUpdate(ctx, &siteapi.SiteUpdateAutomationInput{
		Name: siteapi.NewOptString("Onboarding"),
	}, siteapi.SiteAutomationsUpdateParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	updRes, ok := upd.(*siteapi.SiteAutomationResource)
	require.Truef(t, ok, "got %T", upd)
	assert.Equal(t, "Onboarding", updRes.Name)

	// Delete.
	del, err := c.SiteAutomationsDelete(ctx, siteapi.SiteAutomationsDeleteParams{WorkspaceSlug: slug, ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteAutomationsDeleteNoContent{}, del)
}
