package site_test

import (
	"context"
	"testing"
	"time"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSiteSegmentsPreviewCountsMatchingActiveContacts(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()

	// An active contact with a distinctive custom field in workspace acme (id 1).
	_, err := env.DB.Contact.Create().
		SetWorkspaceID(fixtures.AcmeID).
		SetEmail("preview-target@test.dev").
		SetCustomFields(map[string]any{"plan": "preview-pro"}).
		Save(ctx)
	require.NoError(t, err)

	def := `{"combinator":"and","rules":[{"field":"custom:plan","operator":"=","value":"preview-pro"}]}`
	out, err := c.SiteSegmentsPreview(ctx, &siteapi.SitePreviewSegmentInput{
		Definition: siteapi.NewOptNilString(def),
	}, siteapi.SiteSegmentsPreviewParams{Slug: fixtures.AcmeSlug})
	require.NoError(t, err)
	res, ok := out.(*siteapi.SitePreviewSegmentResult)
	require.Truef(t, ok, "got %T", out)
	assert.Equal(t, int32(1), res.Count)
}

func TestSiteSegmentsPreviewEventCondition(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()

	ct, err := env.DB.Contact.Create().SetWorkspaceID(fixtures.AcmeID).SetEmail("ev-prev@test.dev").Save(ctx)
	require.NoError(t, err)
	// Events attach by the stable contact_id (ADR 0002), not the email string.
	_, err = env.DB.Event.Create().
		SetWorkspaceID(fixtures.AcmeID).SetContactID(ct.ID).
		SetAction("signed_up").SetOccurredAt(time.Now()).Save(ctx)
	require.NoError(t, err)

	def := `{"combinator":"and","rules":[{"field":"event:signed_up","operator":"performed","value":"30"}]}`
	out, err := c.SiteSegmentsPreview(ctx, &siteapi.SitePreviewSegmentInput{
		Definition: siteapi.NewOptNilString(def),
	}, siteapi.SiteSegmentsPreviewParams{Slug: fixtures.AcmeSlug})
	require.NoError(t, err)
	res, ok := out.(*siteapi.SitePreviewSegmentResult)
	require.Truef(t, ok, "got %T", out)
	assert.Equal(t, int32(1), res.Count)
}

func TestSiteSegmentsPreviewRejectsInvalidDefinition(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()

	out, err := c.SiteSegmentsPreview(ctx, &siteapi.SitePreviewSegmentInput{
		Definition: siteapi.NewOptNilString(`{"rules":[{"field":"nope","operator":"=","value":"x"}]}`),
	}, siteapi.SiteSegmentsPreviewParams{Slug: fixtures.AcmeSlug})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSegmentsPreviewUnprocessableEntity{}, out)
}

func TestSiteSegmentsCreateRejectsInvalidRuleDefinition(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()

	out, err := c.SiteSegmentsCreate(ctx, &siteapi.SiteCreateSegmentInput{
		Name:       "Bad rule",
		Definition: siteapi.NewOptNilString(`{"rules":[{"field":"email","operator":"weird","value":"x"}]}`),
	}, siteapi.SiteSegmentsCreateParams{Slug: fixtures.AcmeSlug})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSegmentsCreateUnprocessableEntity{}, out)
}

func TestSiteSegmentsCreateAndUpdateShareValidation(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()
	bad := siteapi.NewOptNilString(`{"rules":[{"field":"email","operator":"weird","value":"x"}]}`)

	// Create rejects a bad definition.
	created, err := c.SiteSegmentsCreate(ctx, &siteapi.SiteCreateSegmentInput{
		Name: "Bad", Definition: bad,
	}, siteapi.SiteSegmentsCreateParams{Slug: fixtures.AcmeSlug})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSegmentsCreateUnprocessableEntity{}, created)

	updated, err := c.SiteSegmentsUpdate(ctx, &siteapi.SiteUpdateSegmentInput{Definition: bad},
		siteapi.SiteSegmentsUpdateParams{Slug: fixtures.AcmeSlug, ID: idStr(fixtures.SegmentActiveID)})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSegmentsUpdateUnprocessableEntity{}, updated)
}

// A Segment is always a rule: an explicit null definition is rejected on update
// (an absent key keeps the stored rule), and the stored rule is left untouched.
func TestSiteSegmentsUpdateRejectsNullDefinition(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()
	params := siteapi.SiteSegmentsUpdateParams{Slug: fixtures.AcmeSlug, ID: "1"}

	var cleared siteapi.OptNilString
	cleared.SetToNull()
	out, err := c.SiteSegmentsUpdate(ctx, &siteapi.SiteUpdateSegmentInput{Definition: cleared}, params)
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSegmentsUpdateUnprocessableEntity{}, out)

	// Absent definition still updates the name and keeps the rule.
	out, err = c.SiteSegmentsUpdate(ctx, &siteapi.SiteUpdateSegmentInput{Name: siteapi.NewOptString("Renamed")}, params)
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSegmentResource{}, out)
}
