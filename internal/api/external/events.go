package external

import (
	"context"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/event"
	externalapi "github.com/mokevnin/1mail/gen/external"
	apiauth "github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/api/problems"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/samber/lo"
)

func (h *Handlers) EventsCreate(ctx context.Context, req externalapi.EventsCreateRequestObject) (externalapi.EventsCreateResponseObject, error) {
	if !apiauth.HasScope(apiauth.GetTokenAuth(ctx), "events:write") {
		return externalapi.EventsCreate401ApplicationProblemPlusJSONResponse(problems.Unauthorized("insufficient scope").External()), nil
	}

	builders := lo.Map(req.Body.Events, func(e externalapi.EventInput, _ int) *ent.EventCreate {
		b := h.ent.Event.Create().
			SetSubjectID(e.SubjectId).
			SetAction(e.Action).
			SetNillableEmail((*string)(e.Email)).
			SetNillablePhone(e.Phone).
			SetNillableProspect(e.Prospect)

		if e.OccurredAt != nil {
			t := time.Time(*e.OccurredAt)
			b = b.SetOccurredAt(t)
		}
		if e.Properties != nil {
			b = b.SetProperties(*e.Properties)
		}
		return b
	})

	_, err := h.ent.Event.CreateBulk(builders...).Save(ctx)
	if err != nil {
		return nil, err
	}
	return externalapi.EventsCreate204Response{}, nil
}

func (h *Handlers) EventActionsList(ctx context.Context, req externalapi.EventActionsListRequestObject) (externalapi.EventActionsListResponseObject, error) {
	if !apiauth.HasScope(apiauth.GetTokenAuth(ctx), "events:read") {
		return externalapi.EventActionsList401ApplicationProblemPlusJSONResponse(problems.Unauthorized("insufficient scope").External()), nil
	}

	page, pageSize := pagination.Normalize(req.Params.Page, req.Params.PageSize)

	actions := h.ent.Event.Query().
		Order(ent.Asc(event.FieldAction)).
		GroupBy(event.FieldAction).
		StringsX(ctx)

	total := len(actions)
	start := pagination.Offset(page, pageSize)
	end := start + pageSize
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}

	slice := actions[start:end]
	items := lo.Map(slice, func(a string, _ int) externalapi.EventActionResource {
		return externalapi.EventActionResource{Action: a}
	})

	return externalapi.EventActionsList200JSONResponse{
		Items:      items,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}
