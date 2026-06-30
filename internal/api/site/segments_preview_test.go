package site_test

import (
	"context"
	"testing"
	"time"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSiteSegmentsPreviewCountsMatchingActiveContacts(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	// An active contact with a distinctive custom field in workspace acme (id 1).
	_, err := env.DB.Contact.Create().
		SetWorkspaceID(1).
		SetEmail("preview-target@test.dev").
		SetCustomFields(map[string]any{"plan": "preview-pro"}).
		Save(ctx)
	require.NoError(t, err)

	def := `{"combinator":"and","rules":[{"field":"custom:plan","operator":"=","value":"preview-pro"}]}`
	out, err := c.SiteSegmentsPreview(ctx, &siteapi.SitePreviewSegmentInput{
		Definition: siteapi.NewOptNilString(def),
	}, siteapi.SiteSegmentsPreviewParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	res, ok := out.(*siteapi.SitePreviewSegmentResult)
	require.Truef(t, ok, "got %T", out)
	assert.Equal(t, int32(1), res.Count)
}

func TestSiteSegmentsPreviewEventCondition(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	ct, err := env.DB.Contact.Create().SetWorkspaceID(1).SetEmail("ev-prev@test.dev").Save(ctx)
	require.NoError(t, err)
	// Events attach by the stable contact_id (ADR 0002), not the email string.
	_, err = env.DB.Event.Create().
		SetWorkspaceID(1).SetContactID(ct.ID).
		SetAction("signed_up").SetOccurredAt(time.Now()).Save(ctx)
	require.NoError(t, err)

	def := `{"combinator":"and","rules":[{"field":"event:signed_up","operator":"performed","value":"30"}]}`
	out, err := c.SiteSegmentsPreview(ctx, &siteapi.SitePreviewSegmentInput{
		Definition: siteapi.NewOptNilString(def),
	}, siteapi.SiteSegmentsPreviewParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	res, ok := out.(*siteapi.SitePreviewSegmentResult)
	require.Truef(t, ok, "got %T", out)
	assert.Equal(t, int32(1), res.Count)
}

func TestSiteSegmentsPreviewRejectsInvalidDefinition(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	out, err := c.SiteSegmentsPreview(ctx, &siteapi.SitePreviewSegmentInput{
		Definition: siteapi.NewOptNilString(`{"rules":[{"field":"nope","operator":"=","value":"x"}]}`),
	}, siteapi.SiteSegmentsPreviewParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSegmentsPreviewUnprocessableEntity{}, out)
}

func TestSiteSegmentsCreateRejectsInvalidRuleDefinition(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	out, err := c.SiteSegmentsCreate(ctx, &siteapi.SiteCreateSegmentInput{
		Name:       "Bad rule",
		Type:       siteapi.SiteSegmentTypeRule,
		Definition: siteapi.NewOptNilString(`{"rules":[{"field":"email","operator":"weird","value":"x"}]}`),
	}, siteapi.SiteSegmentsCreateParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSegmentsCreateUnprocessableEntity{}, out)
}
