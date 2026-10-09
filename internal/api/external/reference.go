package external

import (
	"context"
	"net/http"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/customfield"
	"github.com/mokevnin/1mail/ent/sendingdomain"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/reputation"
)

const (
	defaultRateWindowDays = 7
	maxRateWindowDays     = 90
)

// CustomFieldsList returns the whole (small, unpaginated) Custom field catalogue.
func (h *Handlers) CustomFieldsList(ctx context.Context) (externalapi.CustomFieldsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "custom_fields:read") {
		res := problem(http.StatusUnauthorized, "insufficient scope")
		return &res, nil
	}
	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))

	items, err := h.ent.CustomField.Query().
		Where(customfield.WorkspaceID(ws)).
		Order(ent.Asc(customfield.FieldKey)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	resources := make([]externalapi.CustomFieldResource, len(items))
	for i, f := range items {
		resources[i] = mapper.CustomFieldToResource(f)
	}
	return &externalapi.CustomFieldsListOK{
		Items:      resources,
		Page:       1,
		PageSize:   int32(len(resources)),
		TotalItems: int32(len(resources)),
		TotalPages: 1,
	}, nil
}

func (h *Handlers) SendingDomainsList(ctx context.Context, params externalapi.SendingDomainsListParams) (externalapi.SendingDomainsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "sending_domains:read") {
		res := externalapi.SendingDomainsListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))

	q := h.ent.SendingDomain.Query().Where(sendingdomain.WorkspaceID(ws))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := q.Order(ent.Asc(sendingdomain.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	resources := make([]externalapi.SendingDomainResource, len(items))
	for i, d := range items {
		resources[i] = mapper.SendingDomainToResource(d)
	}
	return &externalapi.SendingDomainsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

// SendingDomainRatesList returns the (numerator, denominator, rate) triples of ADR 0011
// for every Sending domain, over the trailing window.
func (h *Handlers) SendingDomainRatesList(ctx context.Context, params externalapi.SendingDomainRatesListParams) (externalapi.SendingDomainRatesListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "sending_domains:read") {
		res := externalapi.SendingDomainRatesListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))

	days := params.WindowDays.Or(defaultRateWindowDays)
	if days < 1 || days > maxRateWindowDays {
		res := externalapi.SendingDomainRatesListBadRequest(problem(http.StatusBadRequest, "windowDays must be between 1 and 90"))
		return &res, nil
	}

	rates, err := h.reputation.Rates(ctx, ws, time.Duration(days)*24*time.Hour)
	if err != nil {
		return nil, err
	}
	items := make([]externalapi.SendingDomainRatesResource, len(rates))
	for i, r := range rates {
		items[i] = externalapi.SendingDomainRatesResource{
			SendingDomainId: externalapi.EntityId(mapper.SendingDomainToResource(r.Domain).ID),
			Domain:          r.Domain.Domain,
			WindowDays:      days,
			ComplaintRate:   rateTriple(r.Complaint),
			BounceRate:      rateTriple(r.Bounce),
		}
	}
	return &externalapi.SendingDomainRatesListOK{
		Items:      items,
		Page:       1,
		PageSize:   int32(len(items)),
		TotalItems: int32(len(items)),
		TotalPages: 1,
	}, nil
}

func rateTriple(r reputation.Rate) externalapi.RateTriple {
	out := externalapi.RateTriple{Numerator: int32(r.Numerator), Denominator: int32(r.Denominator)}
	if r.Rate == nil {
		out.Rate = externalapi.NilFloat64{Null: true}
	} else {
		out.Rate = externalapi.NewNilFloat64(*r.Rate)
	}
	return out
}
