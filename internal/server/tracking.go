package server

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	apisite "github.com/mokevnin/1mail/internal/api/site"
	"github.com/mokevnin/1mail/internal/tracking"
)

// pixelGIF is a 1x1 transparent GIF served by the open-tracking endpoint.
var pixelGIF, _ = base64.StdEncoding.DecodeString(
	"R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7",
)

// trackingHandler serves the public email engagement endpoints:
//
//	GET /e/o/{token}        — open pixel:       record open, return a 1x1 gif
//	GET /e/c/{token}?u=URL  — click:            record click, 302 to URL
//	GET /e/u/{token}        — unsubscribe:      mark contact unsubscribed
//
// The token is a signed per-recipient JWT. Opens always return the pixel (even
// on a bad token) so we never leak token validity through the image.
func trackingHandler(client *ent.Client, tracker *tracking.Tracker, trigger apisite.AutomationTrigger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /e/o/{token}", func(w http.ResponseWriter, r *http.Request) {
		if rid, err := tracker.Decode(r.PathValue("token")); err == nil {
			recordOpen(r.Context(), client, trigger, rid)
		}
		w.Header().Set("Content-Type", "image/gif")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		_, _ = w.Write(pixelGIF)
	})

	mux.HandleFunc("GET /e/c/{token}", func(w http.ResponseWriter, r *http.Request) {
		dest := r.URL.Query().Get("u")
		if !strings.HasPrefix(dest, "http://") && !strings.HasPrefix(dest, "https://") {
			http.Error(w, "invalid destination", http.StatusBadRequest)
			return
		}
		rid, err := tracker.Decode(r.PathValue("token"))
		if err != nil {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}
		recordClick(r.Context(), client, trigger, rid, dest)
		http.Redirect(w, r, dest, http.StatusFound)
	})

	mux.HandleFunc("GET /e/u/{token}", func(w http.ResponseWriter, r *http.Request) {
		rid, err := tracker.Decode(r.PathValue("token"))
		if err != nil {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}
		recordUnsubscribe(r.Context(), client, trigger, rid)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><body style="font-family:sans-serif;text-align:center;padding:48px">` +
			`<h1>Unsubscribed</h1><p>You will no longer receive these emails.</p></body></html>`))
	})

	return mux
}

func recordOpen(ctx context.Context, client *ent.Client, trigger apisite.AutomationTrigger, recipientID int64) {
	rec, err := client.BroadcastRecipient.Get(ctx, recipientID)
	if err != nil {
		return
	}
	if rec.OpenedAt != nil {
		return // already counted
	}
	if _, err := rec.Update().SetOpenedAt(time.Now()).Save(ctx); err != nil {
		log.Printf("tracking: record open for recipient %d: %v", recipientID, err)
		return
	}
	if _, err := client.Broadcast.UpdateOneID(rec.BroadcastID).AddOpenedCount(1).Save(ctx); err != nil {
		log.Printf("tracking: increment opened_count for broadcast %d: %v", rec.BroadcastID, err)
	}
	recordEvent(ctx, client, trigger, rec, "email.opened", nil)
}

func recordClick(ctx context.Context, client *ent.Client, trigger apisite.AutomationTrigger, recipientID int64, dest string) {
	rec, err := client.BroadcastRecipient.Get(ctx, recipientID)
	if err != nil {
		return
	}
	if rec.ClickedAt != nil {
		return
	}
	if _, err := rec.Update().SetClickedAt(time.Now()).Save(ctx); err != nil {
		log.Printf("tracking: record click for recipient %d: %v", recipientID, err)
		return
	}
	if _, err := client.Broadcast.UpdateOneID(rec.BroadcastID).AddClickedCount(1).Save(ctx); err != nil {
		log.Printf("tracking: increment clicked_count for broadcast %d: %v", rec.BroadcastID, err)
	}
	recordEvent(ctx, client, trigger, rec, "email.clicked", map[string]any{"url": dest})
}

func recordUnsubscribe(ctx context.Context, client *ent.Client, trigger apisite.AutomationTrigger, recipientID int64) {
	rec, err := client.BroadcastRecipient.Get(ctx, recipientID)
	if err != nil {
		return
	}
	c, err := client.Contact.Get(ctx, rec.ContactID)
	if err != nil || c.Status == contact.StatusUnsubscribed {
		return // already unsubscribed: don't double-count
	}
	if _, err := c.Update().SetStatus(contact.StatusUnsubscribed).Save(ctx); err != nil {
		log.Printf("tracking: unsubscribe contact %d: %v", rec.ContactID, err)
		return
	}
	if _, err := client.Broadcast.UpdateOneID(rec.BroadcastID).AddUnsubscribedCount(1).Save(ctx); err != nil {
		log.Printf("tracking: increment unsubscribed_count for broadcast %d: %v", rec.BroadcastID, err)
	}
	recordEvent(ctx, client, trigger, rec, "email.unsubscribed", nil)
}

// recordEvent appends a first-class engagement event and enrolls the contact into
// automations triggered by that action. Automation emails are sent untracked, so
// they emit no open/click events — no trigger loop.
func recordEvent(ctx context.Context, client *ent.Client, trigger apisite.AutomationTrigger, rec *ent.BroadcastRecipient, action string, extra map[string]any) {
	c, err := client.Contact.Get(ctx, rec.ContactID)
	if err != nil {
		return
	}
	props := map[string]any{"broadcast_id": rec.BroadcastID}
	for k, v := range extra {
		props[k] = v
	}
	if _, err := client.Event.Create().
		SetWorkspaceID(rec.WorkspaceID).
		SetSubjectID(c.Email).
		SetEmail(c.Email).
		SetAction(action).
		SetProperties(props).
		Save(ctx); err != nil {
		log.Printf("tracking: record %s event for contact %d: %v", action, rec.ContactID, err)
		return
	}
	_ = trigger.OnEvent(ctx, rec.WorkspaceID, rec.ContactID, action)
}
