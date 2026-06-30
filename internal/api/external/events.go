package external

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/event"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/service"
)

func (h *Handlers) EventsCreate(ctx context.Context, req *externalapi.RecordEventsInput) (externalapi.EventsCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "events:write") {
		res := externalapi.EventsCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))

	// Publish the whole batch in one transaction (one commit, atomic outbox). The
	// persist subscriber writes the Event rows; the webhooks subscriber fans them
	// out. These are the customer's own events, stored as-is. Ingest is now
	// accept-then-process: the caller gets 204 and rows land asynchronously.
	err := h.bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		for i := range req.Events {
			e := req.Events[i]
			collected := &events.CollectedEvent{
				WorkspaceID: ws,
				SubjectID:   e.SubjectId,
				Action:      e.Action,
			}
			if v, ok := e.Email.Get(); ok {
				collected.Email = string(v)
			}
			if v, ok := e.Phone.Get(); ok {
				collected.Phone = v
			}
			// Attach to a Contact by stable identity (ADR 0002): resolve the supplied
			// alias keys to an existing Contact id; 0 ⇒ anonymous (no contact yet).
			contactID, err := service.ResolveContactID(ctx, tx, ws, e.SubjectId, convert.StringPtr(e.Email), convert.StringPtr(e.Phone))
			if err != nil {
				return err
			}
			collected.ContactID = contactID
			if v, ok := e.OccurredAt.Get(); ok {
				collected.OccurredAt = time.Time(v)
			}
			if props, ok := e.Properties.Get(); ok {
				converted := make(map[string]any, len(props))
				for key, raw := range props {
					var value any
					if err := json.Unmarshal([]byte(raw), &value); err != nil {
						return err
					}
					converted[key] = value
				}
				collected.Properties = converted
			}
			if err := pub.Publish(ctx, collected); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &externalapi.EventsCreateNoContent{}, nil
}

func (h *Handlers) EventActionsList(ctx context.Context, params externalapi.EventActionsListParams) (externalapi.EventActionsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "events:read") {
		res := externalapi.EventActionsListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))

	actions := h.ent.Event.Query().
		Where(event.WorkspaceID(ws)).
		Order(ent.Asc(event.FieldAction)).
		GroupBy(event.FieldAction).
		StringsX(ctx)

	total := len(actions)
	start := pagination.Offset(page, pageSize)
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	slice := actions[start:end]
	items := make([]externalapi.EventActionResource, len(slice))
	for i, a := range slice {
		items[i] = externalapi.EventActionResource{Action: a}
	}

	return &externalapi.EventActionsListOK{
		Items:      items,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}
