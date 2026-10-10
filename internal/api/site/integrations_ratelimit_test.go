package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSiteIntegrationsSendRateLimitRoundTrips(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()

	// Unlimited by default: both limits read back as null.
	got, err := c.SiteIntegrationsGet(ctx, siteapi.SiteIntegrationsGetParams{Slug: fixtures.AcmeSlug, ID: "1"})
	require.NoError(t, err)
	def := got.(*siteapi.SiteIntegrationResource)
	assert.True(t, def.MaxPerSecond.Null)
	assert.True(t, def.MaxPerDay.Null)

	upd := siteapi.SiteIntegrationsUpdateParams{Slug: fixtures.AcmeSlug, ID: def.ID}
	updated, err := c.SiteIntegrationsUpdate(ctx, &siteapi.SiteUpdateIntegrationInput{
		MaxPerSecond: siteapi.NewOptNilInt32(14),
		MaxPerDay:    siteapi.NewOptNilInt32(50000),
	}, upd)
	require.NoError(t, err)
	res := updated.(*siteapi.SiteIntegrationResource)
	assert.Equal(t, int32(14), res.MaxPerSecond.Or(0))
	assert.Equal(t, int32(50000), res.MaxPerDay.Or(0))

	// Omitting a limit keeps it; null clears it.
	partial, err := c.SiteIntegrationsUpdate(ctx, &siteapi.SiteUpdateIntegrationInput{
		MaxPerDay: siteapi.OptNilInt32{Set: true, Null: true},
	}, upd)
	require.NoError(t, err)
	partialRes := partial.(*siteapi.SiteIntegrationResource)
	assert.Equal(t, int32(14), partialRes.MaxPerSecond.Or(0), "omitted: unchanged")
	assert.True(t, partialRes.MaxPerDay.Null, "null: cleared")

	// A limit must be positive.
	bad, err := c.SiteIntegrationsUpdate(ctx, &siteapi.SiteUpdateIntegrationInput{
		MaxPerSecond: siteapi.NewOptNilInt32(0),
	}, upd)
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteIntegrationsUpdateUnprocessableEntity{}, bad)

	// Create accepts the limits too.
	created, err := c.SiteIntegrationsCreate(ctx, &siteapi.SiteCreateIntegrationInput{
		Name: "Limited SMTP", Config: smtpInput("smtp.example.com", "pw"),
		MaxPerSecond: siteapi.NewOptNilInt32(2),
	}, siteapi.SiteIntegrationsCreateParams{Slug: fixtures.AcmeSlug})
	require.NoError(t, err)
	assert.Equal(t, int32(2), created.(*siteapi.SiteIntegrationResource).MaxPerSecond.Or(0))
}
