package site_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// Saving an SES Integration reads the account's send quota: the provider values feed
// the effective ceiling (the lowest of manual and provider, manual on a tie).
func TestSiteIntegrationsSaveDiscoversTheSESQuota(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()
	env.SES.SetQuota(14, 50000)

	created, err := c.SiteIntegrationsCreate(ctx, &siteapi.SiteCreateIntegrationInput{
		Name: "SES", Config: sesInput("eu-west-1", "AKIA1234", "secret"),
		MaxPerDay: siteapi.NewOptNilSiteMaxPerDay(1000),
	}, siteapi.SiteIntegrationsCreateParams{Slug: fixtures.AcmeSlug})
	require.NoError(t, err)
	status := created.(*siteapi.SiteIntegrationResource).SendLimit

	assert.Equal(t, int32(14), status.PerSecond.Limit.Or(0))
	assert.Equal(t, siteapi.SiteSendLimitSourceProvider, status.PerSecond.Source.Or(""))
	assert.Equal(t, int32(1000), status.PerDay.Limit.Or(0), "a manual value below the provider's wins")
	assert.Equal(t, siteapi.SiteSendLimitSourceManual, status.PerDay.Source.Or(""))
	assert.Empty(t, status.Warnings)
}

// A failed lookup never blocks the save or sending: the manual value (or none) applies
// and a warning is reported until a later save, or the hourly job, succeeds.
func TestSiteIntegrationsFailedQuotaLookupWarnsAndLaterSuccessClearsIt(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)
	ctx := context.Background()
	env.SES.SetQuotaErr(errors.New("AccessDenied: ses:GetSendQuota"))

	created, err := c.SiteIntegrationsCreate(ctx, &siteapi.SiteCreateIntegrationInput{
		Name: "SES", Config: sesInput("eu-west-1", "AKIA1234", "secret"),
	}, siteapi.SiteIntegrationsCreateParams{Slug: fixtures.AcmeSlug})
	require.NoError(t, err)
	res, ok := created.(*siteapi.SiteIntegrationResource)
	require.Truef(t, ok, "the save succeeds although the lookup failed, got %T", created)
	assert.Equal(t, []siteapi.SiteSendLimitWarning{
		siteapi.SiteSendLimitWarningUnlimited, siteapi.SiteSendLimitWarningProviderQuotaUnavailable,
	}, res.SendLimit.Warnings)
	assert.True(t, res.SendLimit.PerSecond.Limit.Null)

	// A manual value takes over as the only ceiling, and the warning stays.
	upd := siteapi.SiteIntegrationsUpdateParams{Slug: fixtures.AcmeSlug, ID: res.ID}
	manual, err := c.SiteIntegrationsUpdate(ctx, &siteapi.SiteUpdateIntegrationInput{
		MaxPerSecond: siteapi.NewOptNilSiteMaxPerSecond(5),
	}, upd)
	require.NoError(t, err)
	status := manual.(*siteapi.SiteIntegrationResource).SendLimit
	assert.Equal(t, int32(5), status.PerSecond.Limit.Or(0))
	assert.Equal(t, []siteapi.SiteSendLimitWarning{siteapi.SiteSendLimitWarningProviderQuotaUnavailable},
		status.Warnings, "one ceiling is enough to stop the unlimited warning; the quota warning stays")

	// The permission is granted and the Integration is saved again: the warning clears.
	env.SES.SetQuota(14, 50000)
	fixed, err := c.SiteIntegrationsUpdate(ctx, &siteapi.SiteUpdateIntegrationInput{
		Config: siteapi.NewOptNilSiteIntegrationConfigInput(sesInput("eu-west-1", "AKIA1234", "")),
	}, upd)
	require.NoError(t, err)
	status = fixed.(*siteapi.SiteIntegrationResource).SendLimit
	assert.Empty(t, status.Warnings)
	assert.Equal(t, int32(5), status.PerSecond.Limit.Or(0), "manual 5 is below the provider's 14")
	assert.Equal(t, int32(50000), status.PerDay.Limit.Or(0))
	assert.Equal(t, siteapi.SiteSendLimitSourceProvider, status.PerDay.Source.Or(""))
}

// SMTP has no provider quota, so saving it asks nothing and never warns about one.
func TestSiteIntegrationsSMTPSaveDoesNotLookUpAQuota(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)
	env.SES.SetQuotaErr(errors.New("must not be asked"))

	created, err := c.SiteIntegrationsCreate(context.Background(), &siteapi.SiteCreateIntegrationInput{
		Name: "SMTP", Config: smtpInput("smtp.example.com", "pw"),
	}, siteapi.SiteIntegrationsCreateParams{Slug: fixtures.AcmeSlug})
	require.NoError(t, err)
	assert.Equal(t, []siteapi.SiteSendLimitWarning{siteapi.SiteSendLimitWarningUnlimited},
		created.(*siteapi.SiteIntegrationResource).SendLimit.Warnings)
	assert.Zero(t, env.SES.QuotaCalls())
}
