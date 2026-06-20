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
	"github.com/mokevnin/1mail/internal/pagination"
)

func (h *Handlers) EventsCreate(ctx context.Context, req *externalapi.RecordEventsInput) (externalapi.EventsCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "events:write") {
		res := externalapi.EventsCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	builders := make([]*ent.EventCreate, len(req.Events))
	for i := range req.Events {
		e := req.Events[i]
		b := h.ent.Event.Create().
			SetSubjectID(e.SubjectId).
			SetAction(e.Action)

		if v, ok := e.Email.Get(); ok {
			s := string(v)
			b = b.SetNillableEmail(&s)
		}
		if v, ok := e.Phone.Get(); ok {
			b = b.SetNillablePhone(&v)
		}
		if v, ok := e.Prospect.Get(); ok {
			b = b.SetNillableProspect(&v)
		}
		if v, ok := e.OccurredAt.Get(); ok {
			b = b.SetOccurredAt(time.Time(v))
		}
		if props, ok := e.Properties.Get(); ok {
			converted := make(map[string]interface{}, len(props))
			for key, raw := range props {
				var value interface{}
				if err := json.Unmarshal([]byte(raw), &value); err != nil {
					return nil, err
				}
				converted[key] = value
			}
			b = b.SetProperties(converted)
		}

		builders[i] = b
	}

	_, err := h.ent.Event.CreateBulk(builders...).Save(ctx)
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

	page, pageSize := pagination.Normalize(optInt32Ptr(params.Page), optInt32Ptr(params.PageSize))

	actions := h.ent.Event.Query().
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
