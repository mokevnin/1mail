package external

import (
	"context"
	"net/http"
	"time"

	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/oops"
)

func (h *Handlers) AuthMeGet(ctx context.Context) (externalapi.AuthMeGetRes, error) {
	a := auth.GetTokenAuth(ctx)
	if a == nil {
		res := externalapi.AuthMeGetUnauthorized(problem(http.StatusUnauthorized, "missing token"))
		return &res, nil
	}

	token, err := h.ent.ApiToken.Get(ctx, a.TokenID)
	if ent.IsNotFound(err) {
		res := externalapi.AuthMeGetNotFound(problem(http.StatusNotFound, "token not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}

	info := mapper.ApiTokenToInfo(token)
	return &info, nil
}

func (h *Handlers) AuthTokensList(ctx context.Context) (externalapi.AuthTokensListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "tokens:read") {
		res := externalapi.AuthTokensListForbidden(problem(http.StatusForbidden, "insufficient scope"))
		return &res, nil
	}

	tokens, err := h.ent.ApiToken.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]externalapi.ApiTokenInfo, len(tokens))
	for i, t := range tokens {
		items[i] = mapper.ApiTokenToInfo(t)
	}
	return &externalapi.ApiTokenListResponse{Items: items}, nil
}

func (h *Handlers) AuthTokensCreate(ctx context.Context, req *externalapi.CreateApiTokenInput) (externalapi.AuthTokensCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "tokens:write") {
		res := externalapi.AuthTokensCreateForbidden(problem(http.StatusForbidden, "insufficient scope"))
		return &res, nil
	}

	resp, err := createToken(ctx, h.ent, req.Name, req.Scopes, req.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *Handlers) AuthTokensBootstrap(ctx context.Context, req *externalapi.CreateApiTokenInput, params externalapi.AuthTokensBootstrapParams) (externalapi.AuthTokensBootstrapRes, error) {
	if h.bootstrapToken == "" || params.XBootstrapToken != h.bootstrapToken {
		res := externalapi.AuthTokensBootstrapUnauthorized(problem(http.StatusUnauthorized, "invalid bootstrap token"))
		return &res, nil
	}

	resp, err := createToken(ctx, h.ent, req.Name, req.Scopes, req.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *Handlers) AuthTokensDelete(ctx context.Context, params externalapi.AuthTokensDeleteParams) (externalapi.AuthTokensDeleteRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "tokens:write") {
		res := externalapi.AuthTokensDeleteForbidden(problem(http.StatusForbidden, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.AuthTokensDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	_, err = h.ent.ApiToken.UpdateOneID(id).SetRevokedAt(time.Now()).Save(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.AuthTokensDeleteNotFound(problem(http.StatusNotFound, "token not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	return &externalapi.AuthTokensDeleteNoContent{}, nil
}

func createToken(ctx context.Context, client *ent.Client, name string, scopes []externalapi.ApiTokenScope, expiresAt externalapi.OptNilTimestamp) (*externalapi.CreateApiTokenResponse, error) {
	prefix, err := service.GenerateTokenPrefix()
	if err != nil {
		return nil, oops.In("external-auth").Public("could not create token").Wrap(err)
	}
	secret, err := service.GenerateTokenSecret()
	if err != nil {
		return nil, oops.In("external-auth").Public("could not create token").Wrap(err)
	}
	hash, err := service.HashTokenSecret(secret)
	if err != nil {
		return nil, oops.In("external-auth").Public("could not create token").Wrap(err)
	}

	scopeStrings := make([]string, len(scopes))
	for i, s := range scopes {
		scopeStrings[i] = string(s)
	}

	q := client.ApiToken.Create().
		SetName(name).
		SetPrefix(prefix).
		SetSecretHash(hash).
		SetScopes(scopeStrings)

	if v, ok := expiresAt.Get(); ok {
		q = q.SetExpiresAt(time.Time(v))
	}

	token, err := q.Save(ctx)
	if err != nil {
		return nil, oops.In("external-auth").Public("could not create token").Wrap(err)
	}

	return &externalapi.CreateApiTokenResponse{
		Token:     service.TokenValue(prefix, secret),
		TokenInfo: mapper.ApiTokenToInfo(token),
	}, nil
}
