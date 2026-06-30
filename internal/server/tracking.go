package server

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/unsubscribe"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/samber/lo"
)

// pixelGIF is a 1x1 transparent GIF served by the open-tracking endpoint.
var pixelGIF, _ = base64.StdEncoding.DecodeString(
	"R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7",
)

// trackingHandler serves the public email engagement endpoints:
//
//	GET /e/o/{token}        — open pixel:       record open, return a 1x1 gif
//	GET /e/c/{token}?u=URL  — click:            record click, 302 to URL
//	GET /e/u/{token}        — unsubscribe:      opt the destination out of "broadcasts"
//
// The token is a signed per-recipient JWT. Opens always return the pixel (even
// on a bad token) so we never leak token validity through the image.
func trackingHandler(client *ent.Client, bus *events.Bus, tracker *tracking.Tracker) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /e/o/{token}", func(w http.ResponseWriter, r *http.Request) {
		if rid, err := tracker.Decode(r.PathValue("token")); err == nil {
			recordOpen(r.Context(), client, bus, rid)
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
		recordClick(r.Context(), client, bus, rid, dest)
		http.Redirect(w, r, dest, http.StatusFound)
	})

	mux.HandleFunc("GET /e/u/{token}", func(w http.ResponseWriter, r *http.Request) {
		rid, err := tracker.Decode(r.PathValue("token"))
		if err != nil {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}
		recordUnsubscribe(r.Context(), client, bus, rid)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><body style="font-family:sans-serif;text-align:center;padding:48px">` +
			`<h1>Unsubscribed</h1><p>You will no longer receive these emails.</p></body></html>`))
	})

	return mux
}

// The record* helpers commit the state change and publish the engagement event in
// one transaction (transactional outbox). The state write is a conditional UPDATE
// (… WHERE opened_at IS NULL, etc.) and the counter increment + publish are gated
// on its rows-affected, so concurrent or redelivered hits are counted exactly once
// (no double-count race). The persist subscriber writes the Event log row and the
// automations subscriber enrolls the contact — both off the bus. Automation emails
// are sent untracked, so they emit no open/click events (no loop).

func recordOpen(ctx context.Context, client *ent.Client, bus *events.Bus, recipientID int64) {
	rec, err := client.BroadcastRecipient.Get(ctx, recipientID)
	if err != nil {
		return
	}
	c, err := client.Contact.Get(ctx, rec.ContactID)
	if err != nil {
		return
	}
	err = bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		n, err := tx.BroadcastRecipient.Update().
			Where(broadcastrecipient.ID(rec.ID), broadcastrecipient.OpenedAtIsNil()).
			SetOpenedAt(time.Now()).Save(ctx)
		if err != nil || n == 0 {
			return err // n == 0: already counted, nothing to publish
		}
		if _, err := tx.Broadcast.UpdateOneID(rec.BroadcastID).AddOpenedCount(1).Save(ctx); err != nil {
			return err
		}
		return pub.Publish(ctx, &events.EmailEngagement{
			Action: events.NameEmailOpened, WorkspaceID: rec.WorkspaceID, ContactID: rec.ContactID,
			Email: lo.FromPtr(c.Email), BroadcastID: rec.BroadcastID,
		})
	})
	if err != nil {
		log.Printf("tracking: record open for recipient %d: %v", recipientID, err)
	}
}

func recordClick(ctx context.Context, client *ent.Client, bus *events.Bus, recipientID int64, dest string) {
	rec, err := client.BroadcastRecipient.Get(ctx, recipientID)
	if err != nil {
		return
	}
	c, err := client.Contact.Get(ctx, rec.ContactID)
	if err != nil {
		return
	}
	err = bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		n, err := tx.BroadcastRecipient.Update().
			Where(broadcastrecipient.ID(rec.ID), broadcastrecipient.ClickedAtIsNil()).
			SetClickedAt(time.Now()).Save(ctx)
		if err != nil || n == 0 {
			return err
		}
		if _, err := tx.Broadcast.UpdateOneID(rec.BroadcastID).AddClickedCount(1).Save(ctx); err != nil {
			return err
		}
		return pub.Publish(ctx, &events.EmailEngagement{
			Action: events.NameEmailClicked, WorkspaceID: rec.WorkspaceID, ContactID: rec.ContactID,
			Email: lo.FromPtr(c.Email), BroadcastID: rec.BroadcastID, URL: dest,
		})
	})
	if err != nil {
		log.Printf("tracking: record click for recipient %d: %v", recipientID, err)
	}
}

// recordUnsubscribe writes a per-destination opt-out from the "broadcasts"
// sending source (ADR 0001) — never touching any contact status, which no longer
// exists. The default in-email link is scoped to the source, not "everything".
// Scope (broadcasts) and attribution (which broadcast) are two separate facts:
// the scope lives in the Unsubscribe row, the attribution in the counter + event.
func recordUnsubscribe(ctx context.Context, client *ent.Client, bus *events.Bus, recipientID int64) {
	rec, err := client.BroadcastRecipient.Get(ctx, recipientID)
	if err != nil {
		return
	}
	c, err := client.Contact.Get(ctx, rec.ContactID)
	if err != nil {
		return
	}
	dest := eligibility.NormalizeDestination(lo.FromPtr(c.Email))
	if dest == "" {
		return
	}
	err = bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		// Exactly-once: skip the counter + event if the destination already opted
		// out of broadcasts. Concurrent duplicate clicks are caught by the unique
		// (workspace, channel, destination, sending_source) index — the loser's tx
		// rolls back, so the counter is incremented exactly once.
		exists, err := tx.Unsubscribe.Query().Where(
			unsubscribe.WorkspaceID(rec.WorkspaceID),
			unsubscribe.ChannelEQ(unsubscribe.ChannelEmail),
			unsubscribe.DestinationEQ(dest),
			unsubscribe.SendingSourceEQ(eligibility.SourceBroadcasts),
		).Exist(ctx)
		if err != nil || exists {
			return err
		}
		create := tx.Unsubscribe.Create().
			SetWorkspaceID(rec.WorkspaceID).
			SetChannel(unsubscribe.ChannelEmail).
			SetDestination(dest).
			SetSendingSource(eligibility.SourceBroadcasts).
			SetContactID(rec.ContactID)
		if _, err := create.Save(ctx); err != nil {
			return err
		}
		if _, err := tx.Broadcast.UpdateOneID(rec.BroadcastID).AddUnsubscribedCount(1).Save(ctx); err != nil {
			return err
		}
		return pub.Publish(ctx, &events.EmailEngagement{
			Action: events.NameEmailUnsubscribed, WorkspaceID: rec.WorkspaceID, ContactID: rec.ContactID,
			Email: lo.FromPtr(c.Email), BroadcastID: rec.BroadcastID,
		})
	})
	if err != nil {
		log.Printf("tracking: unsubscribe contact %d: %v", rec.ContactID, err)
	}
}
