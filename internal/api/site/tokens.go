package site

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/apitoken"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/i18n"
	"github.com/mokevnin/1mail/internal/service"
)

// SiteTokensList returns the workspace's active (non-revoked) API tokens.
func (h *Handlers) SiteTokensList(ctx context.Context, params siteapi.SiteTokensListParams) (siteapi.SiteTokensListRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := problem(http.StatusNotFound, "workspace not found")
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	tokens, err := h.ent.ApiToken.Query().
		Where(apitoken.WorkspaceID(ws), apitoken.RevokedAtIsNil()).
		Order(ent.Asc(apitoken.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	items := make(siteapi.SiteTokensListOKApplicationJSON, len(tokens))
	for i, t := range tokens {
		items[i] = mapper.TokenToResource(t)
	}
	return &items, nil
}

// SiteTokensCreate mints a workspace API token. The full secret is returned once;
// only its bcrypt hash and public prefix are stored.
func (h *Handlers) SiteTokensCreate(ctx context.Context, req *siteapi.SiteCreateTokenInput, params siteapi.SiteTokensCreateParams) (siteapi.SiteTokensCreateRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTokensCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		v := siteapi.SiteTokensCreateUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity,
			i18n.T("errors.name_empty", nil),
			map[string][]string{"name": {i18n.T("errors.name_empty", nil)}},
		))
		return &v, nil
	}

	prefix, err := service.GenerateTokenPrefix()
	if err != nil {
		return nil, err
	}
	secret, err := service.GenerateTokenSecret()
	if err != nil {
		return nil, err
	}
	hash, err := service.HashTokenSecret(secret)
	if err != nil {
		return nil, err
	}

	create := h.ent.ApiToken.Create().
		SetName(name).
		SetPrefix(prefix).
		SetSecretHash(hash).
		SetScopes(req.Scopes).
		SetWorkspaceID(ws)
	if v, ok := req.ExpiresAt.Get(); ok {
		create = create.SetExpiresAt(time.Time(v))
	}
	token, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	return &siteapi.SiteCreateTokenResponse{
		Token:    service.TokenValue(prefix, secret),
		Resource: mapper.TokenToResource(token),
	}, nil
}

// SiteTokensDelete revokes (soft-deletes) a workspace API token.
func (h *Handlers) SiteTokensDelete(ctx context.Context, params siteapi.SiteTokensDeleteParams) (siteapi.SiteTokensDeleteRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTokensDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteTokensDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}

	// Scope the revoke to the workspace; an unknown / non-owned / already-revoked
	// id affects zero rows and maps to 404.
	n, err := h.ent.ApiToken.Update().
		Where(apitoken.ID(id), apitoken.WorkspaceID(ws), apitoken.RevokedAtIsNil()).
		SetRevokedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		v := siteapi.SiteTokensDeleteNotFound(problem(http.StatusNotFound, "token not found"))
		return &v, nil
	}
	return &siteapi.SiteTokensDeleteNoContent{}, nil
}
