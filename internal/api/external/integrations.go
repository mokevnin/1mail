package external

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/integration"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/sendlimit"
)

// IntegrationsList returns the Workspace's sending-provider Integrations with the Send
// rate limit as enforced and the last 24 hours of usage. Credentials and provider
// config are never part of the resource.
func (h *Handlers) IntegrationsList(ctx context.Context, params externalapi.IntegrationsListParams) (externalapi.IntegrationsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "integrations:read") {
		res := externalapi.IntegrationsListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))

	s := auth.TokenScoped(ctx)
	q := s.Integration().Query()
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.Order(ent.Asc(integration.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	usage, err := sendlimit.Usage(ctx, s, time.Now())
	if err != nil {
		return nil, err
	}
	items := make([]externalapi.IntegrationResource, len(rows))
	for i, row := range rows {
		items[i] = integrationResource(row, usage)
	}
	return &externalapi.IntegrationsListOK{
		Items:      items,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

// integrationResource is hand-built (not goverter): the Send rate limit is computed
// from the Integration and its usage, not copied from a field.
func integrationResource(row *ent.Integration, usage map[int64]int) externalapi.IntegrationResource {
	return externalapi.IntegrationResource{
		ID:        externalapi.EntityId(strconv.FormatInt(row.ID, 10)),
		Name:      row.Name,
		Channel:   externalapi.IntegrationChannel(row.Channel.String()),
		Provider:  externalapi.IntegrationProvider(row.Provider.String()),
		Enabled:   row.Enabled,
		IsDefault: row.IsDefault,
		SendLimit: sendLimitStatus(row, usage),
		CreatedAt: externalapi.Timestamp(row.CreatedAt),
	}
}

func sendLimitStatus(row *ent.Integration, usage map[int64]int) externalapi.SendLimitStatus {
	eff := sendlimit.EffectiveOf(row)
	out := externalapi.SendLimitStatus{
		PerSecond:   sendLimitValue(eff.PerSecond),
		PerDay:      sendLimitValue(eff.PerDay),
		SentLast24h: int32(usage[row.ID]),
		Warnings:    []externalapi.SendLimitWarning{},
	}
	if eff.Unlimited() {
		out.Warnings = append(out.Warnings, externalapi.SendLimitWarningUnlimited)
	}
	return out
}

func sendLimitValue(v sendlimit.Value) externalapi.SendLimitValue {
	out := externalapi.SendLimitValue{}
	if v.Limit == nil {
		out.Limit = externalapi.NilInt32{Null: true}
		out.Source = externalapi.NilSendLimitSource{Null: true}
		return out
	}
	out.Limit = externalapi.NewNilInt32(int32(*v.Limit))
	out.Source = externalapi.NewNilSendLimitSource(externalapi.SendLimitSource(v.Source))
	return out
}
