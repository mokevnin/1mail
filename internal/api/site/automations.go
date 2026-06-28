package site

import (
	"context"
	"net/http"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/automation"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
)

func (h *Handlers) SiteAutomationsList(ctx context.Context, params siteapi.SiteAutomationsListParams) (siteapi.SiteAutomationsListRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAutomationsListNotFound(problem(http.StatusNotFound, "workspace not found"))
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

	q := h.ent.Automation.Query().Where(automation.WorkspaceID(ws))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := q.Order(ent.Desc(automation.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteAutomationResource, len(items))
	for i, a := range items {
		resources[i] = mapper.AutomationToResource(a)
	}
	return &siteapi.SiteAutomationsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteAutomationsCreate(ctx context.Context, req *siteapi.SiteCreateAutomationInput, params siteapi.SiteAutomationsCreateParams) (siteapi.SiteAutomationsCreateRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAutomationsCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	q := h.ent.Automation.Create().
		SetWorkspaceID(ws).
		SetName(req.Name).
		SetTriggerEvent(req.TriggerEvent)
	if v, ok := req.Definition.Get(); ok {
		q = q.SetDefinition(v)
	}
	a, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}
	res := mapper.AutomationToResource(a)
	return &res, nil
}

func (h *Handlers) SiteAutomationsGet(ctx context.Context, params siteapi.SiteAutomationsGetParams) (siteapi.SiteAutomationsGetRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAutomationsGetNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	a, err := h.automationByID(ctx, ws, params.ID)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAutomationsGetNotFound(problem(http.StatusNotFound, "automation not found"))
		return &v, nil
	}
	if err != nil {
		v := siteapi.SiteAutomationsGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	res := mapper.AutomationToResource(a)
	return &res, nil
}

func (h *Handlers) SiteAutomationsUpdate(ctx context.Context, req *siteapi.SiteUpdateAutomationInput, params siteapi.SiteAutomationsUpdateParams) (siteapi.SiteAutomationsUpdateRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAutomationsUpdateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteAutomationsUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	q := h.ent.Automation.UpdateOneID(id).
		Where(automation.WorkspaceID(ws)).
		SetNillableName(convert.StringPtr(req.Name)).
		SetNillableTriggerEvent(convert.StringPtr(req.TriggerEvent)).
		SetNillableDefinition(convert.StringPtr(req.Definition))
	a, err := q.Save(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAutomationsUpdateNotFound(problem(http.StatusNotFound, "automation not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.AutomationToResource(a)
	return &res, nil
}

func (h *Handlers) SiteAutomationsDelete(ctx context.Context, params siteapi.SiteAutomationsDeleteParams) (siteapi.SiteAutomationsDeleteRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAutomationsDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteAutomationsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	err = h.ent.Automation.DeleteOneID(id).Where(automation.WorkspaceID(ws)).Exec(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAutomationsDeleteNotFound(problem(http.StatusNotFound, "automation not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteAutomationsDeleteNoContent{}, nil
}

func (h *Handlers) SiteAutomationsActivate(ctx context.Context, params siteapi.SiteAutomationsActivateParams) (siteapi.SiteAutomationsActivateRes, error) {
	a, err := h.setAutomationStatus(ctx, params.WorkspaceSlug, params.ID, automation.StatusActive)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAutomationsActivateNotFound(problem(http.StatusNotFound, "automation not found"))
		return &v, nil
	}
	if err != nil {
		v := siteapi.SiteAutomationsActivateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	res := mapper.AutomationToResource(a)
	return &res, nil
}

func (h *Handlers) SiteAutomationsDeactivate(ctx context.Context, params siteapi.SiteAutomationsDeactivateParams) (siteapi.SiteAutomationsDeactivateRes, error) {
	a, err := h.setAutomationStatus(ctx, params.WorkspaceSlug, params.ID, automation.StatusDraft)
	if ent.IsNotFound(err) {
		v := siteapi.SiteAutomationsDeactivateNotFound(problem(http.StatusNotFound, "automation not found"))
		return &v, nil
	}
	if err != nil {
		v := siteapi.SiteAutomationsDeactivateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	res := mapper.AutomationToResource(a)
	return &res, nil
}

func (h *Handlers) automationByID(ctx context.Context, ws int64, id siteapi.EntityId) (*ent.Automation, error) {
	parsed, err := strconv.ParseInt(string(id), 10, 64)
	if err != nil {
		return nil, err
	}
	return h.ent.Automation.Query().
		Where(automation.IDEQ(parsed), automation.WorkspaceID(ws)).
		Only(ctx)
}

func (h *Handlers) setAutomationStatus(ctx context.Context, slug string, id siteapi.EntityId, status automation.Status) (*ent.Automation, error) {
	ws, err := h.workspaceID(ctx, slug)
	if err != nil {
		return nil, err
	}
	parsed, err := strconv.ParseInt(string(id), 10, 64)
	if err != nil {
		return nil, err
	}
	return h.ent.Automation.UpdateOneID(parsed).
		Where(automation.WorkspaceID(ws)).
		SetStatus(status).
		Save(ctx)
}
