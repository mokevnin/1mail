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
	"github.com/mokevnin/1mail/ent/confirmation"
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

// confirmPath turns a tracker ConfirmURL into the local "/e/confirm/<token>" path
// the in-memory test server dispatches on.
func confirmPath(t *testing.T, tr *tracking.Tracker, target tracking.ConfirmTarget) string {
	t.Helper()
	full, err := tr.ConfirmURL(target)
	require.NoError(t, err)
	_, token, ok := strings.Cut(full, "/e/confirm/")
	require.True(t, ok, "url %q lacks /e/confirm/", full)
	return "/e/confirm/" + token
}

// Double opt-in (ADR 0013): GET /e/confirm/{token} renders the SPA page and records
// nothing (scanner-safe); only POST records the confirmation, and it is idempotent
// and publishes exactly one marketing.confirmed event.
func TestConfirmEndpoint(t *testing.T) {
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
	post := func(path string) *http.Response {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		w := httptest.NewRecorder()
		env.Server.ServeHTTP(w, req)
		return w.Result()
	}

	c := env.DB.Contact.GetX(ctx, 1)
	cPath := confirmPath(t, tr, tracking.ConfirmTarget{
		Destination: *c.Email, WorkspaceID: 1, ContactID: c.ID,
	})
	confirmed := func() bool {
		ok, err := env.DB.Confirmation.Query().Where(
			confirmation.WorkspaceID(1),
			confirmation.ChannelEQ(confirmation.ChannelEmail),
			confirmation.DestinationEQ(*c.Email),
		).Exist(ctx)
		require.NoError(t, err)
		return ok
	}

	// GET is safe: 303 to the SPA confirm page, records nothing.
	resp := get(cPath)
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.True(t, strings.HasPrefix(resp.Header.Get("Location"), "/confirm?token="))
	assert.False(t, confirmed(), "GET records nothing")

	// POST records the confirmation with provenance double_opt_in.
	resp = post(cPath)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.True(t, confirmed(), "POST records the confirmation")
	row := env.DB.Confirmation.Query().Where(confirmation.DestinationEQ(*c.Email)).OnlyX(ctx)
	assert.Equal(t, confirmation.ProvenanceDoubleOptIn, row.Provenance)

	// A repeated POST is a complete no-op: still one row.
	post(cPath)
	n, err := env.DB.Confirmation.Query().Where(confirmation.DestinationEQ(*c.Email)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "repeated POST does not duplicate the confirmation")

	// Exactly one marketing.confirmed event was published onto the outbox.
	var count int
	require.NoError(t, env.SQLDB.QueryRow(
		`SELECT count(*) FROM watermill_domain_events
		 WHERE payload->>'name' = 'marketing.confirmed' AND payload->'data'->>'email' = $1`,
		*c.Email).Scan(&count))
	assert.Equal(t, 1, count, "one confirmation event, published once")
}

// The deliberate "unsubscribe from everything" invalidates a confirmation (ADR
// 0013): it deletes the derived Confirmation row so returning requires
// re-confirmation. A narrower per-source opt-out leaves the confirmation intact.
func TestUnsubscribeEverythingInvalidatesConfirmation(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	cfg, err := config.Load("test")
	require.NoError(t, err)
	tr := tracking.New(cfg.JWTSecret, cfg.AppURL)

	post := func(path string) *http.Response {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		w := httptest.NewRecorder()
		env.Server.ServeHTTP(w, req)
		return w.Result()
	}

	c := env.DB.Contact.GetX(ctx, 1)
	_, err = env.DB.Confirmation.Create().SetWorkspaceID(1).
		SetChannel(confirmation.ChannelEmail).SetDestination(*c.Email).
		SetProvenance(confirmation.ProvenanceDoubleOptIn).SetContactID(c.ID).Save(ctx)
	require.NoError(t, err)

	confExists := func() bool {
		ok, err := env.DB.Confirmation.Query().Where(
			confirmation.WorkspaceID(1),
			confirmation.DestinationEQ(*c.Email),
		).Exist(ctx)
		require.NoError(t, err)
		return ok
	}

	// A per-source (broadcasts) opt-out must NOT invalidate the confirmation.
	post(unsubPath(t, tr, tracking.UnsubTarget{
		Source: eligibility.SourceBroadcasts, Destination: *c.Email, WorkspaceID: 1, ContactID: c.ID,
	}))
	assert.True(t, confExists(), "per-source opt-out leaves the confirmation")

	// The everything opt-out deletes the confirmation row.
	post(unsubPath(t, tr, tracking.UnsubTarget{
		Source: eligibility.SourceEverything, Destination: *c.Email, WorkspaceID: 1, ContactID: c.ID,
	}))
	assert.False(t, confExists(), "everything opt-out invalidates the confirmation")
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
	post := func(path string) *http.Response {
		req := httptest.NewRequest(http.MethodPost, path, nil)
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

	// Unsubscribe (ADR 0012 / RFC 8058): the broadcast footer link is a
	// "broadcasts"-scoped token. GET is safe — it 303-redirects to the SPA confirm
	// page and records nothing (a link scanner's GET must not opt anyone out); only
	// POST performs the opt-out.
	uPath := unsubPath(t, tr, tracking.UnsubTarget{
		Source: eligibility.SourceBroadcasts, Destination: *c.Email,
		WorkspaceID: 1, ContactID: c.ID, BroadcastID: b.ID,
	})
	optedOut := func() bool {
		ok, err := env.DB.Unsubscribe.Query().Where(
			unsubscribe.WorkspaceID(1),
			unsubscribe.ChannelEQ(unsubscribe.ChannelEmail),
			unsubscribe.DestinationEQ(*c.Email),
			unsubscribe.SendingSourceEQ(eligibility.SourceBroadcasts),
		).Exist(ctx)
		require.NoError(t, err)
		return ok
	}

	resp = get(uPath)
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.True(t, strings.HasPrefix(resp.Header.Get("Location"), "/unsubscribe?"))
	assert.False(t, optedOut(), "GET records nothing")
	assert.Equal(t, 0, env.DB.Broadcast.GetX(ctx, b.ID).UnsubscribedCount)

	// POST performs the opt-out and bumps the counter once.
	resp = post(uPath)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.True(t, optedOut(), "POST writes a broadcasts-scoped opt-out")
	assert.Equal(t, 1, env.DB.Broadcast.GetX(ctx, b.ID).UnsubscribedCount)

	// A repeated POST (mailbox retry / double click) must not double-count.
	post(uPath)
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

	post := func(path string) *http.Response {
		req := httptest.NewRequest(http.MethodPost, path, nil)
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

	// POST performs the opt-out (GET only renders the confirm page — ADR 0012).
	resp := post(unsubPath(t, tr, tracking.UnsubTarget{
		Source: eligibility.AutomationSource(a.ID), Destination: *c.Email,
		WorkspaceID: 1, ContactID: c.ID,
	}))
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

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

// The confirm-page redirect carries an "unsubscribe from everything" escalation;
// following it lands on a confirm page with no further escalation, and POSTing
// both tokens records an everything-scoped opt-out alongside the source one — two
// scopes coexist.
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
	post := func(path string) *http.Response {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		w := httptest.NewRecorder()
		env.Server.ServeHTTP(w, req)
		return w.Result()
	}

	c := env.DB.Contact.GetX(ctx, 1)
	b, err := env.DB.Broadcast.Create().SetWorkspaceID(1).SetName("B").SetSubject("Hi").
		SetStatus(broadcast.StatusSent).Save(ctx)
	require.NoError(t, err)

	// GET the source unsubscribe → confirm-page redirect carrying the escalation
	// URL in ?all=.
	srcPath := unsubPath(t, tr, tracking.UnsubTarget{
		Source: eligibility.SourceBroadcasts, Destination: *c.Email,
		WorkspaceID: 1, ContactID: c.ID, BroadcastID: b.ID,
	})
	resp := get(srcPath)
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	loc, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	allURL := loc.Query().Get("all")
	require.NotEmpty(t, allURL, "confirm page offers an unsubscribe-from-everything link")

	// The escalation link is itself a GET /e/u/ confirm link; following it lands on
	// a confirm page that offers no further escalation.
	_, evToken, ok := strings.Cut(allURL, "/e/u/")
	require.True(t, ok)
	evPath := "/e/u/" + evToken
	resp = get(evPath)
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	loc2, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	assert.Empty(t, loc2.Query().Get("all"))

	// POST both tokens to perform both opt-outs.
	assert.Equal(t, http.StatusNoContent, post(srcPath).StatusCode)
	assert.Equal(t, http.StatusNoContent, post(evPath).StatusCode)

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
