package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent/automation"
	"github.com/mokevnin/1mail/ent/automationrun"
	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/unsubscribe"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unsubPath turns a tracker UnsubscribeURL into the local "/e/u/<token>" path the
// in-memory test server dispatches on.
func unsubPath(t *testing.T, tr *tracking.Tracker, target tracking.UnsubTarget) string {
	t.Helper()
	full, err := tr.UnsubscribeURL(target)
	require.NoError(t, err)
	_, token, ok := strings.Cut(full, "/e/u/")
	require.True(t, ok, "url %q lacks /e/u/", full)
	return "/e/u/" + token
}

// The public /e/* endpoints record opens, clicks and unsubscribes against a
// per-recipient token and feed first-class engagement events.
func TestTrackingEndpoints(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	cfg, err := config.Load("test")
	require.NoError(t, err)
	tr := tracking.New(cfg.JWTSecret, cfg.AppURL)

	// A recipient in fixture workspace acme (id 1). Alice (id 1) is not already
	// unsubscribed from broadcasts (bob is, in the fixtures).
	c := env.DB.Contact.GetX(ctx, 1)

	b, err := env.DB.Broadcast.Create().
		SetWorkspaceID(1).SetName("Track").SetSubject("Hi").
		SetStatus(broadcast.StatusSent).Save(ctx)
	require.NoError(t, err)

	rec, err := env.DB.BroadcastRecipient.Create().
		SetBroadcastID(b.ID).SetWorkspaceID(1).SetContactID(c.ID).
		SetStatus(broadcastrecipient.StatusSent).Save(ctx)
	require.NoError(t, err)

	token, err := tr.Token(rec.ID)
	require.NoError(t, err)

	get := func(path string) *http.Response {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		env.Server.ServeHTTP(w, req)
		return w.Result()
	}

	// Open: returns the pixel and records the open.
	resp := get("/e/o/" + token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/gif", resp.Header.Get("Content-Type"))

	gotRec := env.DB.BroadcastRecipient.GetX(ctx, rec.ID)
	assert.NotNil(t, gotRec.OpenedAt)
	assert.Equal(t, 1, env.DB.Broadcast.GetX(ctx, b.ID).OpenedCount)

	// Click: records the click and 302-redirects to the destination.
	resp = get("/e/c/" + token + "?u=https%3A%2F%2Fdest.test%2Fx")
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "https://dest.test/x", resp.Header.Get("Location"))
	assert.NotNil(t, env.DB.BroadcastRecipient.GetX(ctx, rec.ID).ClickedAt)
	assert.Equal(t, 1, env.DB.Broadcast.GetX(ctx, b.ID).ClickedCount)

	// Unsubscribe: the broadcast footer link is a "broadcasts"-scoped token; the
	// endpoint records the opt-out, bumps the counter once, and 303-redirects to
	// the SPA confirmation page (no server-rendered HTML).
	uPath := unsubPath(t, tr, tracking.UnsubTarget{
		Source: eligibility.SourceBroadcasts, Destination: *c.Email,
		WorkspaceID: 1, ContactID: c.ID, BroadcastID: b.ID,
	})
	resp = get(uPath)
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.True(t, strings.HasPrefix(resp.Header.Get("Location"), "/unsubscribed"))
	optedOut, err := env.DB.Unsubscribe.Query().Where(
		unsubscribe.WorkspaceID(1),
		unsubscribe.ChannelEQ(unsubscribe.ChannelEmail),
		unsubscribe.DestinationEQ(*c.Email),
		unsubscribe.SendingSourceEQ(eligibility.SourceBroadcasts),
	).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, optedOut, "unsubscribe writes a broadcasts-scoped opt-out")
	assert.Equal(t, 1, env.DB.Broadcast.GetX(ctx, b.ID).UnsubscribedCount)

	// Redelivery of the same unsubscribe must not double-count.
	get(uPath)
	assert.Equal(t, 1, env.DB.Broadcast.GetX(ctx, b.ID).UnsubscribedCount)

	// Each endpoint published an engagement event onto the transactional outbox.
	// The persist + automations subscribers consume these (the router isn't run
	// under txdb; delivery and projection are covered by the events package
	// tests). Assert the outbox carries all three events for this recipient.
	rows, err := env.SQLDB.Query(
		`SELECT payload->>'name' FROM watermill_domain_events WHERE payload->'data'->>'email' = $1`, c.Email)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var names []string
	for rows.Next() {
		var n string
		require.NoError(t, rows.Scan(&n))
		names = append(names, n)
	}
	require.NoError(t, rows.Err())
	assert.Contains(t, names, "email.opened")
	assert.Contains(t, names, "email.clicked")
	assert.Contains(t, names, "email.unsubscribed")
}

// Unsubscribing from an automation records the opt-out AND exits the in-flight
// enrollment — two effects from one action (ADR 0001).
func TestUnsubscribeAutomationExitsEnrollment(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	cfg, err := config.Load("test")
	require.NoError(t, err)
	tr := tracking.New(cfg.JWTSecret, cfg.AppURL)

	get := func(path string) *http.Response {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		env.Server.ServeHTTP(w, req)
		return w.Result()
	}

	c := env.DB.Contact.GetX(ctx, 1)
	a, err := env.DB.Automation.Create().SetWorkspaceID(1).
		SetName("Welcome").SetTriggerEvent("contact.created").SetStatus(automation.StatusActive).
		SetDefinition("[]").Save(ctx)
	require.NoError(t, err)
	run, err := env.DB.AutomationRun.Create().
		SetWorkspaceID(1).SetAutomationID(a.ID).SetContactID(c.ID).
		SetStatus(automationrun.StatusActive).Save(ctx)
	require.NoError(t, err)

	resp := get(unsubPath(t, tr, tracking.UnsubTarget{
		Source: eligibility.AutomationSource(a.ID), Destination: *c.Email,
		WorkspaceID: 1, ContactID: c.ID,
	}))
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)

	optedOut, err := env.DB.Unsubscribe.Query().Where(
		unsubscribe.WorkspaceID(1),
		unsubscribe.DestinationEQ(*c.Email),
		unsubscribe.SendingSourceEQ(eligibility.AutomationSource(a.ID)),
	).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, optedOut, "records the automation-scoped opt-out")
	assert.Equal(t, automationrun.StatusExited, env.DB.AutomationRun.GetX(ctx, run.ID).Status,
		"the active enrollment is exited")
}

// The confirmation redirect carries an "unsubscribe from everything" escalation;
// following it records an everything-scoped opt-out while the source opt-out
// stays — two scopes coexist.
func TestUnsubscribeEverythingEscalation(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	cfg, err := config.Load("test")
	require.NoError(t, err)
	tr := tracking.New(cfg.JWTSecret, cfg.AppURL)

	get := func(path string) *http.Response {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		env.Server.ServeHTTP(w, req)
		return w.Result()
	}

	c := env.DB.Contact.GetX(ctx, 1)
	b, err := env.DB.Broadcast.Create().SetWorkspaceID(1).SetName("B").SetSubject("Hi").
		SetStatus(broadcast.StatusSent).Save(ctx)
	require.NoError(t, err)

	// Source unsubscribe → redirect carries the escalation URL in ?all=.
	resp := get(unsubPath(t, tr, tracking.UnsubTarget{
		Source: eligibility.SourceBroadcasts, Destination: *c.Email,
		WorkspaceID: 1, ContactID: c.ID, BroadcastID: b.ID,
	}))
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	loc, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	allURL := loc.Query().Get("all")
	require.NotEmpty(t, allURL, "redirect offers an unsubscribe-from-everything link")

	// Follow the escalation link.
	_, token, ok := strings.Cut(allURL, "/e/u/")
	require.True(t, ok)
	resp = get("/e/u/" + token)
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	// The everything escalation page itself offers no further escalation.
	loc2, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	assert.Empty(t, loc2.Query().Get("all"))

	// Both scopes now exist for the destination.
	for _, src := range []string{eligibility.SourceBroadcasts, eligibility.SourceEverything} {
		exists, err := env.DB.Unsubscribe.Query().Where(
			unsubscribe.WorkspaceID(1),
			unsubscribe.DestinationEQ(*c.Email),
			unsubscribe.SendingSourceEQ(src),
		).Exist(ctx)
		require.NoError(t, err)
		assert.Truef(t, exists, "scope %q opt-out recorded", src)
	}
}
