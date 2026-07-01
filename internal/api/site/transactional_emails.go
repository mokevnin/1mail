package site

import (
	"context"
	"net/http"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/transactionalemail"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/pagination"
)

// SiteTransactionalEmailsList returns the workspace's transactional send history
// (the durable trace written by the /api/emails surface), most recent first.
func (h *Handlers) SiteTransactionalEmailsList(ctx context.Context, params siteapi.SiteTransactionalEmailsListParams) (siteapi.SiteTransactionalEmailsListRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTransactionalEmailsListNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	var pagePtr, pageSizePtr *int32
	if v, ok := params.Page.Get(); ok {
		pagePtr = &v
	}
	if v, ok := params.PageSize.Get(); ok {
		pageSizePtr = &v
	}
	page, pageSize := pagination.Normalize(pagePtr, pageSizePtr)

	q := h.ent.TransactionalEmail.Query().Where(transactionalemail.WorkspaceID(ws))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := q.Order(ent.Desc(transactionalemail.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]siteapi.SiteTransactionalEmailResource, len(items))
	for i, t := range items {
		res[i] = mapper.TransactionalEmailToResource(t)
	}
	return &siteapi.SiteTransactionalEmailsListOK{
		Items:      res,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}
