package server

import (
	"context"
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/automationrun"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/confirmation"
	"github.com/mokevnin/1mail/ent/unsubscribe"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/logging"
	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/samber/lo"
)

// pixelGIF is a 1x1 transparent GIF served by the open-tracking endpoint.
var pixelGIF, _ = base64.StdEncoding.DecodeString(
	"R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7",
)

// trackingHandler serves the public email engagement endpoints:
//
//	GET  /e/o/{token}       — open pixel:       record open, return a 1x1 gif
//	GET  /e/c/{token}?u=URL — click:            record click, 302 to URL
//	GET  /e/u/{token}       — unsubscribe confirm: render the SPA page, record nothing
//	POST /e/u/{token}       — unsubscribe perform: opt the destination out of the scope
//	GET  /e/confirm/{token} — double opt-in confirm: render the SPA page, record nothing
//	POST /e/confirm/{token} — double opt-in perform: record the confirmation
//
// Unsubscribe and confirm are split by method (ADR 0012 / RFC 8058, ADR 0013): GET
// is safe and only renders the confirmation page, so link scanners and security
// proxies that GET every URL cannot unsubscribe or confirm anyone; the state change
// happens only on POST — the target of both the mailbox one-click POST and the
// page's button. Confirmation tokens additionally expire (~7 days).
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
		tokenStr := r.PathValue("token")
		target, err := tracker.DecodeUnsub(tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}
		// Safe method: record nothing, render the SPA confirmation page. The page's
		// button POSTs back to this same URL to perform the opt-out. For a per-source
		// opt-out we pass the deliberate "unsubscribe from everything" escalation link
		// so the page can offer it (a click, never an auto-fired request).
		http.Redirect(w, r, confirmRedirect(tracker, tokenStr, target), http.StatusSeeOther)
	})

	mux.HandleFunc("POST /e/u/{token}", func(w http.ResponseWriter, r *http.Request) {
		target, err := tracker.DecodeUnsub(r.PathValue("token"))
		if err != nil {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}
		// Performs the opt-out — the target of the mailbox provider's one-click POST
		// (List-Unsubscribe=One-Click body) and the confirm page's button. No page is
		// returned; the SPA transitions its UI client-side on 204.
		recordUnsubscribe(r.Context(), client, bus, target)
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /e/confirm/{token}", func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.PathValue("token")
		if _, err := tracker.DecodeConfirm(tokenStr); err != nil {
			// Invalid or expired: send to the page with no token so it offers the
			// "request a new confirmation" path rather than a dead Confirm button.
			http.Redirect(w, r, "/confirm?expired=1", http.StatusSeeOther)
			return
		}
		// Safe method: record nothing, render the SPA confirmation page. The page's
		// button POSTs back to this same URL to perform the confirmation.
		http.Redirect(w, r, "/confirm?token="+url.QueryEscape(tokenStr), http.StatusSeeOther)
	})

	mux.HandleFunc("POST /e/confirm/{token}", func(w http.ResponseWriter, r *http.Request) {
		target, err := tracker.DecodeConfirm(r.PathValue("token"))
		if err != nil {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}
		// Performs the confirmation — the deliberate human act required for legal
		// validity. No page is returned; the SPA transitions its UI on 204.
		recordConfirmation(r.Context(), client, bus, target, clientIP(r))
		w.WriteHeader(http.StatusNoContent)
	})

	return mux
}

// clientIP is the best-effort confirming client address recorded as GDPR proof on
// a double-opt-in confirmation. It prefers the first hop of X-Forwarded-For (the
// binary runs behind Caddy/ingress) and falls back to the connection's remote host.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(xff)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// confirmRedirect is the SPA confirmation route the GET handler redirects to. It
// carries the token so the page's button can POST it back, and for a per-source
// opt-out the everything-scoped escalation URL (built from the same destination)
// so the page can offer it after confirming.
func confirmRedirect(tracker *tracking.Tracker, tokenStr string, target tracking.UnsubTarget) string {
	base := "/unsubscribe?token=" + url.QueryEscape(tokenStr)
	if target.Source == eligibility.SourceEverything {
		return base
	}
	allURL, err := tracker.UnsubscribeURL(tracking.UnsubTarget{
		Source:      eligibility.SourceEverything,
		Destination: target.Destination,
		WorkspaceID: target.WorkspaceID,
		ContactID:   target.ContactID,
	})
	if err != nil {
		return base
	}
	return base + "&all=" + url.QueryEscape(allURL)
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
		logging.FromContext(ctx).Error("tracking: record open failed", "recipient_id", recipientID, "err", err)
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
		logging.FromContext(ctx).Error("tracking: record click failed", "recipient_id", recipientID, "err", err)
	}
}

// recordUnsubscribe writes a per-(channel, destination, sending source) opt-out
// (ADR 0001) from a signed token — no contact-row lookup, so the opt-out records
// even if the contact was deleted between send and click (the point of
// destination-keying). The default in-email link is scoped to its sending source;
// "everything" is the deliberate escalation. All effects (the row, the broadcast
// counter, the automation enrollment exit, and the engagement event) live in one
// transaction gated on the existence check, so a repeated POST (mailbox retry or a
// double click) is a complete no-op and concurrent POSTs are counted exactly once.
func recordUnsubscribe(ctx context.Context, client *ent.Client, bus *events.Bus, target tracking.UnsubTarget) {
	dest := eligibility.NormalizeDestination(target.Destination)
	if dest == "" || target.WorkspaceID == 0 || target.Source == "" {
		return
	}
	automationID, isAutomation := eligibility.ParseAutomationSource(target.Source)

	err := bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		exists, err := tx.Unsubscribe.Query().Where(
			unsubscribe.WorkspaceID(target.WorkspaceID),
			unsubscribe.ChannelEQ(unsubscribe.ChannelEmail),
			unsubscribe.DestinationEQ(dest),
			unsubscribe.SendingSourceEQ(target.Source),
		).Exist(ctx)
		if err != nil || exists {
			return err
		}
		create := tx.Unsubscribe.Create().
			SetWorkspaceID(target.WorkspaceID).
			SetChannel(unsubscribe.ChannelEmail).
			SetDestination(dest).
			SetSendingSource(target.Source)
		if target.ContactID != 0 {
			create.SetContactID(target.ContactID)
		}
		if _, err := create.Save(ctx); err != nil {
			return err
		}

		// Everything-opt-out invalidates any confirmation (ADR 0013): the deliberate
		// "leave entirely" deletes the derived Confirmation row so returning requires
		// re-confirmation (stale consent never silently reactivates). The immutable
		// marketing.confirmed Event is preserved as proof ("confirmed at T1, left at
		// T2"). A narrower per-source opt-out does NOT touch confirmation.
		if target.Source == eligibility.SourceEverything {
			if _, err := tx.Confirmation.Delete().Where(
				confirmation.WorkspaceID(target.WorkspaceID),
				confirmation.ChannelEQ(confirmation.ChannelEmail),
				confirmation.DestinationEQ(dest),
			).Exec(ctx); err != nil {
				return err
			}
		}

		// Broadcast attribution: bump the triggering broadcast's counter.
		if target.BroadcastID != 0 {
			if _, err := tx.Broadcast.UpdateOneID(target.BroadcastID).AddUnsubscribedCount(1).Save(ctx); err != nil {
				return err
			}
		}
		// Automation: unsubscribing from an automation also exits its active
		// enrollment (ADR: two effects from one action).
		if isAutomation && target.ContactID != 0 {
			if _, err := tx.AutomationRun.Update().
				Where(
					automationrun.AutomationID(automationID),
					automationrun.ContactID(target.ContactID),
					automationrun.StatusEQ(automationrun.StatusActive),
				).
				SetStatus(automationrun.StatusExited).
				ClearResumeAt().
				Save(ctx); err != nil {
				return err
			}
		}

		return pub.Publish(ctx, &events.EmailEngagement{
			Action: events.NameEmailUnsubscribed, WorkspaceID: target.WorkspaceID, ContactID: target.ContactID,
			Email: dest, BroadcastID: target.BroadcastID,
		})
	})
	if err != nil {
		logging.FromContext(ctx).Error("tracking: unsubscribe failed", "destination", dest, "source", target.Source, "err", err)
	}
}

// recordConfirmation writes the derived Confirmation read-model row (provenance
// double_opt_in) and publishes the immutable marketing.confirmed Event in one
// transaction (ADR 0013) — the positive mirror of recordUnsubscribe. It is keyed
// by destination, so it records even if the contact was deleted between send and
// click. The existence check makes a repeated POST (mailbox retry, double click)
// a complete no-op: the confirmation stands and no second Event is logged.
func recordConfirmation(ctx context.Context, client *ent.Client, bus *events.Bus, target tracking.ConfirmTarget, ip string) {
	dest := eligibility.NormalizeDestination(target.Destination)
	if dest == "" || target.WorkspaceID == 0 {
		return
	}

	err := bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		exists, err := tx.Confirmation.Query().Where(
			confirmation.WorkspaceID(target.WorkspaceID),
			confirmation.ChannelEQ(confirmation.ChannelEmail),
			confirmation.DestinationEQ(dest),
		).Exist(ctx)
		if err != nil || exists {
			return err
		}
		create := tx.Confirmation.Create().
			SetWorkspaceID(target.WorkspaceID).
			SetChannel(confirmation.ChannelEmail).
			SetDestination(dest).
			SetProvenance(confirmation.ProvenanceDoubleOptIn)
		if target.ContactID != 0 {
			create.SetContactID(target.ContactID)
		}
		if _, err := create.Save(ctx); err != nil {
			return err
		}
		return pub.Publish(ctx, &events.MarketingConfirmed{
			WorkspaceID: target.WorkspaceID,
			ContactID:   target.ContactID,
			Email:       dest,
			Provenance:  string(confirmation.ProvenanceDoubleOptIn),
			IP:          ip,
		})
	})
	if err != nil {
		logging.FromContext(ctx).Error("tracking: confirmation failed", "destination", dest, "err", err)
	}
}
