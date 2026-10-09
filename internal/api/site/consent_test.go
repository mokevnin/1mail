package site_test

import (
	"strings"
	"testing"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent/confirmation"
	"github.com/mokevnin/1mail/ent/unsubscribe"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func consentTracker(t *testing.T) *tracking.Tracker {
	t.Helper()
	cfg, err := config.Load("test")
	require.NoError(t, err)
	return tracking.New(cfg.JWTSecret, cfg.AppURL)
}

// tokenAfter returns the token segment of a tracker URL that follows marker.
func tokenAfter(t *testing.T, url, marker string) string {
	t.Helper()
	_, token, ok := strings.Cut(url, marker)
	require.True(t, ok, "url %q lacks %s", url, marker)
	return token
}

// Double opt-in (ADR 0013): the confirmation page's button performs the
// confirmation through the site API. It is idempotent and a bad token is a 400.
func TestSitePublicConfirmationsPerform(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := t.Context()
	anon := env.SiteAnonymous(t)
	tr := consentTracker(t)

	c := env.DB.Contact.GetX(ctx, fixtures.ContactAliceID)
	url, err := tr.ConfirmURL(tracking.ConfirmTarget{
		Destination: *c.Email, WorkspaceID: fixtures.AcmeID, ContactID: c.ID,
	})
	require.NoError(t, err)
	token := tokenAfter(t, url, "/e/confirm/")
	count := func() int {
		n, err := env.DB.Confirmation.Query().Where(
			confirmation.WorkspaceID(fixtures.AcmeID),
			confirmation.DestinationEQ(*c.Email),
		).Count(ctx)
		require.NoError(t, err)
		return n
	}
	require.Zero(t, count())

	out, err := anon.SitePublicConfirmationsPerform(ctx, siteapi.SitePublicConfirmationsPerformParams{Token: token})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SitePublicConfirmationsPerformNoContent{}, out)
	assert.Equal(t, 1, count())

	_, err = anon.SitePublicConfirmationsPerform(ctx, siteapi.SitePublicConfirmationsPerformParams{Token: token})
	require.NoError(t, err)
	assert.Equal(t, 1, count(), "repeating is a no-op")

	out, err = anon.SitePublicConfirmationsPerform(ctx, siteapi.SitePublicConfirmationsPerformParams{Token: "garbage"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, out)
}

// Unsubscribe (ADR 0012): the page button performs the opt-out through the site
// API; the mailbox one-click POST keeps using /e/u/{token}.
func TestSitePublicUnsubscribesPerform(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := t.Context()
	anon := env.SiteAnonymous(t)
	tr := consentTracker(t)

	c := env.DB.Contact.GetX(ctx, fixtures.ContactAliceID)
	url, err := tr.UnsubscribeURL(tracking.UnsubTarget{
		Source: eligibility.SourceBroadcasts, Destination: *c.Email, WorkspaceID: fixtures.AcmeID, ContactID: c.ID,
	})
	require.NoError(t, err)
	token := tokenAfter(t, url, "/e/u/")
	optedOut := func() bool {
		ok, err := env.DB.Unsubscribe.Query().Where(
			unsubscribe.WorkspaceID(fixtures.AcmeID),
			unsubscribe.DestinationEQ(*c.Email),
			unsubscribe.SendingSourceEQ(eligibility.SourceBroadcasts),
		).Exist(ctx)
		require.NoError(t, err)
		return ok
	}
	require.False(t, optedOut())

	out, err := anon.SitePublicUnsubscribesPerform(ctx, siteapi.SitePublicUnsubscribesPerformParams{Token: token})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SitePublicUnsubscribesPerformNoContent{}, out)
	assert.True(t, optedOut())

	out, err = anon.SitePublicUnsubscribesPerform(ctx, siteapi.SitePublicUnsubscribesPerformParams{Token: "garbage"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, out)
}
