package external

import (
	"context"
	"errors"
	"net/http"

	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/api/external/resources"
	"github.com/mokevnin/1mail/internal/automations"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
)

const (
	scopeAutomationsRead     = "automations:read"
	scopeAutomationsWrite    = "automations:write"
	scopeAutomationsActivate = "automations:activate"
)

func (h *Handlers) AutomationsList(ctx context.Context, params externalapi.AutomationsListParams) (externalapi.AutomationsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), scopeAutomationsRead) {
		res := externalapi.AutomationsListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))
	items, total, err := h.automations.List(ctx, auth.TokenScoped(ctx), pageSize, pagination.Offset(page, pageSize))
	if err != nil {
		return nil, err
	}
	out := make([]externalapi.AutomationResource, len(items))
	for i, a := range items {
		out[i] = mapper.AutomationToResource(a)
	}
	return &externalapi.AutomationsListOK{
		Items:      out,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

// AutomationsCreate always stores a draft: activating is the send-class operation.
func (h *Handlers) AutomationsCreate(ctx context.Context, req *externalapi.CreateAutomationInput) (externalapi.AutomationsCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), scopeAutomationsWrite) {
		res := externalapi.AutomationsCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	in := automations.CreateInput{Name: req.Name, TriggerEvent: req.TriggerEvent}
	if req.Steps != nil {
		in.Steps = resources.AutomationSteps(req.Steps)
	}
	a, err := h.automations.Create(ctx, auth.TokenScoped(ctx), in)
	if errors.Is(err, automations.ErrInvalidStep) {
		res := externalapi.AutomationsCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, err.Error()))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.AutomationToResource(a)
	return &res, nil
}

func (h *Handlers) AutomationsGet(ctx context.Context, params externalapi.AutomationsGetParams) (externalapi.AutomationsGetRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), scopeAutomationsRead) {
		res := externalapi.AutomationsGetUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.AutomationsGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}
	a, err := h.automations.Get(ctx, auth.TokenScoped(ctx), id)
	if errors.Is(err, automations.ErrNotFound) {
		res := externalapi.AutomationsGetNotFound(problem(http.StatusNotFound, "automation not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.AutomationToResource(a)
	return &res, nil
}

func (h *Handlers) AutomationsUpdate(ctx context.Context, req *externalapi.UpdateAutomationInput, params externalapi.AutomationsUpdateParams) (externalapi.AutomationsUpdateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), scopeAutomationsWrite) {
		res := externalapi.AutomationsUpdateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.AutomationsUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	in := automations.UpdateInput{
		Name:         convert.StringPtr(req.Name),
		TriggerEvent: convert.StringPtr(req.TriggerEvent),
	}
	if req.Steps != nil {
		steps := resources.AutomationSteps(req.Steps)
		in.Steps = &steps
	}
	a, err := h.automations.Update(ctx, auth.TokenScoped(ctx), id, in)
	switch {
	case errors.Is(err, automations.ErrNotFound):
		res := externalapi.AutomationsUpdateNotFound(problem(http.StatusNotFound, "automation not found"))
		return &res, nil
	case errors.Is(err, automations.ErrInvalidStep):
		res := externalapi.AutomationsUpdateUnprocessableEntity(problem(http.StatusUnprocessableEntity, err.Error()))
		return &res, nil
	case err != nil:
		return nil, err
	}
	res := mapper.AutomationToResource(a)
	return &res, nil
}

func (h *Handlers) AutomationsDelete(ctx context.Context, params externalapi.AutomationsDeleteParams) (externalapi.AutomationsDeleteRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), scopeAutomationsWrite) {
		res := externalapi.AutomationsDeleteUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.AutomationsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}
	err = h.automations.Delete(ctx, auth.TokenScoped(ctx), id)
	if errors.Is(err, automations.ErrNotFound) {
		res := externalapi.AutomationsDeleteNotFound(problem(http.StatusNotFound, "automation not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	return &externalapi.AutomationsDeleteNoContent{}, nil
}

// AutomationsActivate and AutomationsDeactivate need automations:activate, which
// authoring scopes never imply (ADR 0016). Over MCP, activate additionally needs
// mcp:send (the x-mcp send flag).
func (h *Handlers) AutomationsActivate(ctx context.Context, params externalapi.AutomationsActivateParams) (externalapi.AutomationsActivateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), scopeAutomationsActivate) {
		res := externalapi.AutomationsActivateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.AutomationsActivateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}
	a, err := h.automations.Activate(ctx, auth.TokenScoped(ctx), id)
	if errors.Is(err, automations.ErrNotFound) {
		res := externalapi.AutomationsActivateNotFound(problem(http.StatusNotFound, "automation not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.AutomationToResource(a)
	return &res, nil
}

func (h *Handlers) AutomationsDeactivate(ctx context.Context, params externalapi.AutomationsDeactivateParams) (externalapi.AutomationsDeactivateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), scopeAutomationsActivate) {
		res := externalapi.AutomationsDeactivateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.AutomationsDeactivateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}
	a, err := h.automations.Deactivate(ctx, auth.TokenScoped(ctx), id)
	if errors.Is(err, automations.ErrNotFound) {
		res := externalapi.AutomationsDeactivateNotFound(problem(http.StatusNotFound, "automation not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.AutomationToResource(a)
	return &res, nil
}
