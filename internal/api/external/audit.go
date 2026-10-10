package external

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-faster/jx"
	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/events"
)

const (
	defaultAuditLimit = 25
	maxAuditLimit     = 100
	// auditReadScope gates the Audit log read. It is deliberately absent from the
	// OAuth grantable scopes, and the operation is hidden from MCP (ADR 0016, 0022).
	auditReadScope = "audit:read"
)

// AuditEntriesList reads the Workspace's Audit log, newest first, for a token with
// the audit:read scope (a SIEM feed). An instance without an Enterprise license
// answers 402 (ADR 0014, ADR 0022).
func (h *Handlers) AuditEntriesList(ctx context.Context, params externalapi.AuditEntriesListParams) (externalapi.AuditEntriesListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), auditReadScope) {
		res := externalapi.AuditEntriesListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	if h.audit == nil || !h.audit.Licensed() {
		res := externalapi.AuditEntriesListPaymentRequired(problem(http.StatusPaymentRequired, "the audit log needs an Enterprise license"))
		return &res, nil
	}

	var cursor int64
	if c, ok := params.Cursor.Get(); ok && c != "" {
		var err error
		if cursor, err = strconv.ParseInt(c, 10, 64); err != nil || cursor < 1 {
			res := externalapi.AuditEntriesListBadRequest(problem(http.StatusBadRequest, "invalid cursor"))
			return &res, nil
		}
	}
	limit := int(params.Limit.Or(defaultAuditLimit))
	if limit < 1 || limit > maxAuditLimit {
		limit = defaultAuditLimit
	}

	rows, next, err := h.audit.Entries(ctx, auth.TokenScoped(ctx), cursor, limit)
	if err != nil {
		return nil, err
	}
	out := &externalapi.AuditEntryList{Items: make([]externalapi.AuditEntryResource, len(rows))}
	for i, e := range rows {
		out.Items[i] = auditEntryResource(e)
	}
	if next > 0 {
		out.NextCursor = externalapi.NewOptNilString(strconv.FormatInt(next, 10))
	}
	return out, nil
}

func auditEntryResource(e *ent.AuditEntry) externalapi.AuditEntryResource {
	actor := externalapi.AuditActor{
		Kind: externalapi.AuditActorKind(e.ActorKind),
		ID:   nilableString(e.ActorID),
		Name: nilableString(e.ActorName),
	}
	if e.ActorKind == events.ActorOperator {
		actor.ID = externalapi.OptNilString{}
		actor.Name = externalapi.NewOptNilString(events.OperatorLabel)
	}
	res := externalapi.AuditEntryResource{
		ID:         externalapi.EntityId(strconv.FormatInt(e.ID, 10)),
		OccurredAt: externalapi.Timestamp(e.OccurredAt),
		Actor:      actor,
		Action:     e.Action,
		Target: externalapi.AuditTarget{
			Type: e.TargetType,
			ID:   nilableString(e.TargetID),
			Name: nilableString(e.TargetName),
		},
		RequestId: nilableString(e.RequestID),
		IP:        nilableString(e.IP),
		UserAgent: nilableString(e.UserAgent),
	}
	if len(e.Diff) > 0 {
		diff := make(externalapi.AuditEntryResourceDiff, len(e.Diff))
		for k, v := range e.Diff {
			if raw, err := json.Marshal(v); err == nil {
				diff[k] = jx.Raw(raw)
			}
		}
		res.Diff = externalapi.NewOptNilAuditEntryResourceDiff(diff)
	}
	return res
}

func nilableString(v *string) externalapi.OptNilString {
	if v == nil {
		return externalapi.OptNilString{}
	}
	return externalapi.NewOptNilString(*v)
}
