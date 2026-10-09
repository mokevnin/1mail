package external

import (
	"context"
	"net/http"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/automation"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/ent/unsubscribe"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/consent"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/tracking"
)

// Consent only narrows through /api (ADR 0016): these two operations add a
// Suppression or record an Unsubscribe. Resubscribing and lifting a Suppression are
// deliberately absent — they widen reach and assert consent, so a human (or the
// contact via link) does them.

func (h *Handlers) SuppressionsCreate(ctx context.Context, req *externalapi.CreateSuppressionInput) (externalapi.SuppressionsCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:write") {
		res := externalapi.SuppressionsCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))

	dest := eligibility.NormalizeDestination(string(req.Destination))
	if dest == "" {
		res := externalapi.SuppressionsCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, "destination must not be empty"))
		return &res, nil
	}

	// Idempotent per (channel, destination): an existing entry keeps its reason.
	if err := h.ent.Suppression.Create().
		SetWorkspaceID(ws).
		SetChannel(suppression.ChannelEmail).
		SetDestination(dest).
		SetReason(suppression.ReasonManual).
		OnConflictColumns(suppression.FieldWorkspaceID, suppression.FieldChannel, suppression.FieldDestination).
		Ignore().
		Exec(ctx); err != nil {
		return nil, err
	}
	s, err := h.ent.Suppression.Query().
		Where(suppression.WorkspaceID(ws), suppression.ChannelEQ(suppression.ChannelEmail), suppression.DestinationEQ(dest)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return &externalapi.SuppressionResource{
		ID:          entityID(s.ID),
		Destination: s.Destination,
		Reason:      externalapi.SuppressionReason(s.Reason),
		CreatedAt:   externalapi.Timestamp(s.CreatedAt),
	}, nil
}

func (h *Handlers) UnsubscribesCreate(ctx context.Context, req *externalapi.CreateUnsubscribeInput) (externalapi.UnsubscribesCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:write") {
		res := externalapi.UnsubscribesCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	unprocessable := func(detail string) (externalapi.UnsubscribesCreateRes, error) {
		res := externalapi.UnsubscribesCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, detail))
		return &res, nil
	}

	dest := eligibility.NormalizeDestination(string(req.Destination))
	if dest == "" {
		return unprocessable("destination must not be empty")
	}
	source := req.SendingSource.Or(eligibility.SourceBroadcasts)
	switch automationID, isAutomation := eligibility.ParseAutomationSource(source); {
	case source == eligibility.SourceBroadcasts || source == eligibility.SourceEverything:
	case isAutomation:
		exists, err := h.ent.Automation.Query().Where(automation.ID(automationID), automation.WorkspaceID(ws)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !exists {
			return unprocessable("unknown automation in sendingSource")
		}
	default:
		return unprocessable("sendingSource must be broadcasts, everything or automation:<id>")
	}

	target := tracking.UnsubTarget{Source: source, Destination: dest, WorkspaceID: ws}
	if c, err := h.ent.Contact.Query().Where(contact.WorkspaceID(ws), contact.EmailEqualFold(dest)).First(ctx); err == nil {
		target.ContactID = c.ID
	} else if !ent.IsNotFound(err) {
		return nil, err
	}
	// Same effects as the unsubscribe link: row, confirmation reset on "everything",
	// automation exit and the engagement event, atomically and idempotently.
	if err := consent.RecordUnsubscribe(ctx, h.ent, h.bus, target); err != nil {
		return nil, err
	}

	u, err := h.ent.Unsubscribe.Query().Where(
		unsubscribe.WorkspaceID(ws),
		unsubscribe.ChannelEQ(unsubscribe.ChannelEmail),
		unsubscribe.DestinationEQ(dest),
		unsubscribe.SendingSourceEQ(source),
	).Only(ctx)
	if err != nil {
		return nil, err
	}
	return &externalapi.UnsubscribeResource{
		ID:            entityID(u.ID),
		Destination:   u.Destination,
		SendingSource: u.SendingSource,
		CreatedAt:     externalapi.Timestamp(u.CreatedAt),
	}, nil
}
