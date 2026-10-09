package external

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/eventlog"
	"github.com/mokevnin/1mail/internal/pagination"
)

func (h *Handlers) EventsCreate(ctx context.Context, req *externalapi.RecordEventsInput) (externalapi.EventsCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "events:write") {
		res := externalapi.EventsCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	inputs, err := eventInputs(req.Events)
	if err != nil {
		return nil, err
	}
	// The module resolves identity and publishes the batch atomically; rows land
	// asynchronously (accept-then-process).
	if err := h.eventlog.Ingest(ctx, auth.TokenScoped(ctx), inputs); err != nil {
		return nil, err
	}
	return &externalapi.EventsCreateNoContent{}, nil
}

func eventInputs(events []externalapi.EventInput) ([]eventlog.Input, error) {
	inputs := make([]eventlog.Input, len(events))
	for i, e := range events {
		in := eventlog.Input{
			SubjectID: e.SubjectId,
			Email:     convert.StringPtr(e.Email),
			Phone:     convert.StringPtr(e.Phone),
			Action:    e.Action,
		}
		if v, ok := e.OccurredAt.Get(); ok {
			in.OccurredAt = time.Time(v)
		}
		if props, ok := e.Properties.Get(); ok {
			in.Properties = make(map[string]any, len(props))
			for key, raw := range props {
				var value any
				if err := json.Unmarshal([]byte(raw), &value); err != nil {
					return nil, err
				}
				in.Properties[key] = value
			}
		}
		inputs[i] = in
	}
	return inputs, nil
}

// EventsBatchSubmit records each Event independently and reports a per-item result.
func (h *Handlers) EventsBatchSubmit(ctx context.Context, req *externalapi.RecordEventsBatchInput) (externalapi.EventsBatchSubmitRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "events:write") {
		res := externalapi.EventsBatchSubmitUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	inputs, err := eventInputs(req.Events)
	if err != nil {
		return nil, err
	}
	errs := h.eventlog.IngestEach(ctx, auth.TokenScoped(ctx), inputs)

	results := make([]externalapi.EventBatchItemResult, len(errs))
	for i, e := range errs {
		results[i] = externalapi.EventBatchItemResult{Index: int32(i), Status: externalapi.EventBatchStatusAccepted}
		if e != nil {
			results[i].Status = externalapi.EventBatchStatusFailed
			results[i].Error = externalapi.NewOptString(itemError(e))
		}
	}
	return &externalapi.RecordEventsBatchResult{Results: results}, nil
}

func (h *Handlers) EventActionsList(ctx context.Context, params externalapi.EventActionsListParams) (externalapi.EventActionsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "events:read") {
		res := externalapi.EventActionsListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))

	actions, err := h.eventlog.Actions(ctx, auth.TokenScoped(ctx))
	if err != nil {
		return nil, err
	}

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
