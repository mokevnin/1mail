package collect

import (
	"context"
	"time"

	"github.com/mokevnin/1mail/backend/ent"
	collectapi "github.com/mokevnin/1mail/backend/gen/collect"
	"github.com/mokevnin/1mail/backend/internal/service"
)

type Handlers struct {
	ent *ent.Client
}

func NewHandlers(client *ent.Client) *Handlers {
	return &Handlers{ent: client}
}

func (h *Handlers) CollectEventsCreate(ctx context.Context, req collectapi.CollectEventsCreateRequestObject) (collectapi.CollectEventsCreateResponseObject, error) {
	events := make([]service.CollectEventInput, len(req.Body.Events))
	for i, e := range req.Body.Events {
		evt := service.CollectEventInput{
			VisitorID: e.VisitorId,
			Action:    e.Action,
		}
		if e.OccurredAt != nil {
			t := time.Time(*e.OccurredAt)
			evt.OccurredAt = &t
		}
		if e.Properties != nil {
			evt.Properties = *e.Properties
		}
		events[i] = evt
	}

	if err := service.CollectEvents(ctx, h.ent, events); err != nil {
		return nil, err
	}
	return collectapi.CollectEventsCreate204Response{}, nil
}

func (h *Handlers) CollectIdentifyCreate(ctx context.Context, req collectapi.CollectIdentifyCreateRequestObject) (collectapi.CollectIdentifyCreateResponseObject, error) {
	input := service.IdentifyInput{
		VisitorID: req.Body.VisitorId,
		SubjectID: req.Body.SubjectId,
		Email:     (*string)(req.Body.Email),
		Phone:     req.Body.Phone,
	}
	if req.Body.Traits != nil {
		input.Traits = *req.Body.Traits
	}

	if err := service.IdentifyVisitor(ctx, h.ent, input); err != nil {
		return nil, err
	}
	ok := collectapi.CollectOkResponseOk(true)
	return collectapi.CollectIdentifyCreate200JSONResponse{Ok: ok}, nil
}

var _ collectapi.StrictServerInterface = (*Handlers)(nil)
