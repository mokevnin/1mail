package site

import (
	"context"
	"net/http"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/clientip"
	"github.com/mokevnin/1mail/internal/consent"
	"github.com/mokevnin/1mail/internal/logging"
)

// SitePublicConfirmationsPerform is the double opt-in confirmation page's button
// (ADR 0013): the one place a confirmation is performed.
func (h *Handlers) SitePublicConfirmationsPerform(ctx context.Context, params siteapi.SitePublicConfirmationsPerformParams) (siteapi.SitePublicConfirmationsPerformRes, error) {
	target, err := h.tracker.DecodeConfirm(params.Token)
	if err != nil {
		v := problem(http.StatusBadRequest, "invalid token")
		return &v, nil
	}
	if err := consent.RecordConfirmation(ctx, h.ent, h.bus, target, clientip.FromContext(ctx)); err != nil {
		logging.FromContext(ctx).Error("site: confirmation failed", "destination", target.Destination, "err", err)
	}
	return &siteapi.SitePublicConfirmationsPerformNoContent{}, nil
}

// SitePublicUnsubscribesPerform is the unsubscribe page's button (ADR 0012): the
// same effect as POST /e/u/{token}, which stays for the mailbox one-click POST.
func (h *Handlers) SitePublicUnsubscribesPerform(ctx context.Context, params siteapi.SitePublicUnsubscribesPerformParams) (siteapi.SitePublicUnsubscribesPerformRes, error) {
	target, err := h.tracker.DecodeUnsub(params.Token)
	if err != nil {
		v := problem(http.StatusBadRequest, "invalid token")
		return &v, nil
	}
	if err := consent.RecordUnsubscribe(ctx, h.ent, h.bus, target); err != nil {
		logging.FromContext(ctx).Error("site: unsubscribe failed", "destination", target.Destination, "source", target.Source, "err", err)
	}
	return &siteapi.SitePublicUnsubscribesPerformNoContent{}, nil
}
