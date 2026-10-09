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
	s := auth.TokenScoped(ctx)

	dest := eligibility.NormalizeDestination(string(req.Destination))
	if dest == "" {
		res := externalapi.SuppressionsCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, "destination must not be empty"))
		return &res, nil
	}

	// Idempotent per (channel, destination): an existing entry keeps its reason.
	if err := s.Suppression().Create().
		SetChannel(suppression.ChannelEmail).
		SetDestination(dest).
		SetReason(suppression.ReasonManual).
		OnConflictColumns(suppression.FieldWorkspaceID, suppression.FieldChannel, suppression.FieldDestination).
		Ignore().
		Exec(ctx); err != nil {
		return nil, err
	}
	created, err := s.Suppression().Query().
		Where(suppression.ChannelEQ(suppression.ChannelEmail), suppression.DestinationEQ(dest)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return &externalapi.SuppressionResource{
		ID:          entityID(created.ID),
		Destination: created.Destination,
		Reason:      externalapi.SuppressionReason(created.Reason),
		CreatedAt:   externalapi.Timestamp(created.CreatedAt),
	}, nil
}

func (h *Handlers) UnsubscribesCreate(ctx context.Context, req *externalapi.CreateUnsubscribeInput) (externalapi.UnsubscribesCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:write") {
		res := externalapi.UnsubscribesCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	s := auth.TokenScoped(ctx)
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
		exists, err := s.Automation().Query().Where(automation.ID(automationID)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !exists {
			return unprocessable("unknown automation in sendingSource")
		}
	default:
		return unprocessable("sendingSource must be broadcasts, everything or automation:<id>")
	}

	target := tracking.UnsubTarget{Source: source, Destination: dest, WorkspaceID: s.WorkspaceID()}
	if c, err := s.Contact().Query().Where(contact.EmailEqualFold(dest)).First(ctx); err == nil {
		target.ContactID = c.ID
	} else if !ent.IsNotFound(err) {
		return nil, err
	}
	// Same effects as the unsubscribe link: row, confirmation reset on "everything",
	// automation exit and the engagement event, atomically and idempotently.
	if err := consent.RecordUnsubscribe(ctx, h.bus, target); err != nil {
		return nil, err
	}

	u, err := s.Unsubscribe().Query().Where(
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
