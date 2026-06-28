package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The public /e/* endpoints record opens, clicks and unsubscribes against a
// per-recipient token and feed first-class engagement events.
func TestTrackingEndpoints(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	cfg, err := config.Load("test")
	require.NoError(t, err)
	tr := tracking.New(cfg.JWTSecret, cfg.AppURL)

	// A recipient in fixture workspace acme (id 1).
	c, err := env.DB.Contact.Query().
		Where(contact.WorkspaceID(1), contact.StatusEQ(contact.StatusActive)).
		First(ctx)
	require.NoError(t, err)

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

	// Unsubscribe: marks the contact unsubscribed.
	resp = get("/e/u/" + token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, contact.StatusUnsubscribed, env.DB.Contact.GetX(ctx, c.ID).Status)
	assert.Equal(t, 1, env.DB.Broadcast.GetX(ctx, b.ID).UnsubscribedCount)

	// Engagement events were recorded.
	actions, err := env.DB.Event.Query().
		Where(event.WorkspaceID(1), event.SubjectID(c.Email)).
		Select(event.FieldAction).Strings(ctx)
	require.NoError(t, err)
	assert.Contains(t, actions, "email.opened")
	assert.Contains(t, actions, "email.clicked")
	assert.Contains(t, actions, "email.unsubscribed")
}
