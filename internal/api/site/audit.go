package site

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-faster/jx"
	"github.com/mokevnin/1mail/ent"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/events"
)

const (
	defaultAuditLimit = 25
	maxAuditLimit     = 100
	// operatorLabel is how a platform Operator appears to the customer (ADR 0022):
	// the staff identity is never exposed.
	operatorLabel = "1mail staff"
)

// SiteAuditList shows the Workspace's Audit log, newest first. Owner and admin only;
// an instance without an Enterprise license answers 402 (ADR 0014, ADR 0022).
func (h *Handlers) SiteAuditList(ctx context.Context, params siteapi.SiteAuditListParams) (siteapi.SiteAuditListRes, error) {
	s, role, err := h.scopedWithRoleFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAuditListNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	if !canManageMembers(role) {
		v := siteapi.SiteAuditListForbidden(problem(http.StatusForbidden, "insufficient role"))
		return &v, nil
	}
	if h.audit == nil || !h.audit.Licensed() {
		v := siteapi.SiteAuditListPaymentRequired(problem(http.StatusPaymentRequired, "the audit log needs an Enterprise license"))
		return &v, nil
	}

	var cursor int64
	if c, ok := params.Cursor.Get(); ok && c != "" {
		if cursor, err = strconv.ParseInt(c, 10, 64); err != nil || cursor < 1 {
			v := siteapi.SiteAuditListBadRequest(problem(http.StatusBadRequest, "invalid cursor"))
			return &v, nil
		}
	}
	limit := int(params.Limit.Or(defaultAuditLimit))
	if limit < 1 || limit > maxAuditLimit {
		limit = defaultAuditLimit
	}

	rows, next, err := h.audit.Entries(ctx, s, cursor, limit)
	if err != nil {
		return nil, err
	}
	out := &siteapi.SiteAuditEntryList{Items: make([]siteapi.SiteAuditEntryResource, len(rows))}
	for i, e := range rows {
		out.Items[i] = auditEntryResource(e)
	}
	if next > 0 {
		out.NextCursor = siteapi.NewOptNilString(strconv.FormatInt(next, 10))
	}
	return out, nil
}

func auditEntryResource(e *ent.AuditEntry) siteapi.SiteAuditEntryResource {
	actor := siteapi.SiteAuditActor{
		Kind: siteapi.SiteAuditActorKind(e.ActorKind),
		ID:   nilableString(e.ActorID),
		Name: nilableString(e.ActorName),
	}
	if e.ActorKind == events.ActorOperator {
		actor.ID = siteapi.OptNilString{}
		actor.Name = siteapi.NewOptNilString(operatorLabel)
	}
	res := siteapi.SiteAuditEntryResource{
		ID:         siteapi.EntityId(strconv.FormatInt(e.ID, 10)),
		OccurredAt: siteapi.Timestamp(e.OccurredAt),
		Actor:      actor,
		Action:     e.Action,
		Target: siteapi.SiteAuditTarget{
			Type: e.TargetType,
			ID:   nilableString(e.TargetID),
			Name: nilableString(e.TargetName),
		},
		RequestId: nilableString(e.RequestID),
		IP:        nilableString(e.IP),
		UserAgent: nilableString(e.UserAgent),
	}
	if len(e.Diff) > 0 {
		diff := make(siteapi.SiteAuditEntryResourceDiff, len(e.Diff))
		for k, v := range e.Diff {
			if raw, err := json.Marshal(v); err == nil {
				diff[k] = jx.Raw(raw)
			}
		}
		res.Diff = siteapi.NewOptNilSiteAuditEntryResourceDiff(diff)
	}
	return res
}

func nilableString(v *string) siteapi.OptNilString {
	if v == nil {
		return siteapi.OptNilString{}
	}
	return siteapi.NewOptNilString(*v)
}
