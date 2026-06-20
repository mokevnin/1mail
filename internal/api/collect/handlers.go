package collect

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-faster/jx"
	"github.com/mokevnin/1mail/ent"
	collectapi "github.com/mokevnin/1mail/gen/collect"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/lo"
)

type Handlers struct {
	ent *ent.Client
}

func NewHandlers(client *ent.Client) *Handlers {
	return &Handlers{ent: client}
}

// rawMap decodes an ogen map[string]jx.Raw into map[string]any using
// encoding/json so that value types (float64 for numbers, nested
// map[string]any for objects, etc.) match what the service layer expects.
func rawMap(m map[string]jx.Raw) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		var decoded any
		if err := json.Unmarshal(v, &decoded); err != nil {
			return nil
		}
		out[k] = decoded
	}
	return out
}

func (h *Handlers) CollectEventsCreate(ctx context.Context, req *collectapi.CollectEventsInput) (collectapi.CollectEventsCreateRes, error) {
	events := lo.Map(req.Events, func(e collectapi.CollectEventInput, _ int) service.CollectEventInput {
		evt := service.CollectEventInput{
			VisitorID: e.VisitorId,
			Action:    e.Action,
		}
		if props, ok := e.Properties.Get(); ok {
			evt.Properties = rawMap(props)
		}
		if ts, ok := e.OccurredAt.Get(); ok {
			t := time.Time(ts)
			evt.OccurredAt = &t
		}
		return evt
	})

	if err := service.CollectEvents(ctx, h.ent, events); err != nil {
		return nil, err
	}
	return &collectapi.CollectEventsCreateNoContent{}, nil
}

func (h *Handlers) CollectIdentifyCreate(ctx context.Context, req *collectapi.CollectIdentifyInput) (collectapi.CollectIdentifyCreateRes, error) {
	input := service.IdentifyInput{
		VisitorID: req.VisitorId,
	}
	if v, ok := req.Email.Get(); ok {
		s := string(v)
		input.Email = &s
	}
	if v, ok := req.Phone.Get(); ok {
		s := v
		input.Phone = &s
	}
	if v, ok := req.SubjectId.Get(); ok {
		s := v
		input.SubjectID = &s
	}
	if traits, ok := req.Traits.Get(); ok {
		input.Traits = rawMap(traits)
	}

	if err := service.IdentifyVisitor(ctx, h.ent, input); err != nil {
		return nil, err
	}
	return &collectapi.CollectOkResponse{Ok: collectapi.CollectOkResponseOkTrue}, nil
}

var _ collectapi.Handler = (*Handlers)(nil)
