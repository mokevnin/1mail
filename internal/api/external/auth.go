package external

import (
	"context"
	"strconv"
	"time"

	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/service"
)

func (h *Handlers) AuthMeGet(ctx context.Context, req externalapi.AuthMeGetRequestObject) (externalapi.AuthMeGetResponseObject, error) {
	auth := GetTokenAuth(ctx)
	if auth == nil {
		return externalapi.AuthMeGet401ApplicationProblemPlusJSONResponse(unauthorized("missing token")), nil
	}
	token, err := h.ent.ApiToken.Get(ctx, auth.TokenID)
	if ent.IsNotFound(err) {
		return externalapi.AuthMeGet404ApplicationProblemPlusJSONResponse(notFound("token not found")), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.AuthMeGet200JSONResponse(toTokenInfo(token)), nil
}

func (h *Handlers) AuthTokensList(ctx context.Context, req externalapi.AuthTokensListRequestObject) (externalapi.AuthTokensListResponseObject, error) {
	if !hasScope(GetTokenAuth(ctx), "tokens:read") {
		return externalapi.AuthTokensList403ApplicationProblemPlusJSONResponse(forbidden("insufficient scope")), nil
	}

	tokens, err := h.ent.ApiToken.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]externalapi.ApiTokenInfo, len(tokens))
	for i, t := range tokens {
		items[i] = toTokenInfo(t)
	}
	return externalapi.AuthTokensList200JSONResponse{Items: items}, nil
}

func (h *Handlers) AuthTokensCreate(ctx context.Context, req externalapi.AuthTokensCreateRequestObject) (externalapi.AuthTokensCreateResponseObject, error) {
	if !hasScope(GetTokenAuth(ctx), "tokens:write") {
		return externalapi.AuthTokensCreate403ApplicationProblemPlusJSONResponse(forbidden("insufficient scope")), nil
	}

	resp, err := createToken(ctx, h.ent, req.Body.Name, req.Body.Scopes, req.Body.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return externalapi.AuthTokensCreate201JSONResponse(*resp), nil
}

func (h *Handlers) AuthTokensBootstrap(ctx context.Context, req externalapi.AuthTokensBootstrapRequestObject) (externalapi.AuthTokensBootstrapResponseObject, error) {
	if h.bootstrapToken == "" || req.Params.XBootstrapToken != h.bootstrapToken {
		return externalapi.AuthTokensBootstrap401ApplicationProblemPlusJSONResponse(unauthorized("invalid bootstrap token")), nil
	}

	scopes := make([]externalapi.ApiTokenScope, len(req.Body.Scopes))
	copy(scopes, req.Body.Scopes)

	resp, err := createToken(ctx, h.ent, req.Body.Name, scopes, req.Body.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return externalapi.AuthTokensBootstrap201JSONResponse(*resp), nil
}

func (h *Handlers) AuthTokensDelete(ctx context.Context, req externalapi.AuthTokensDeleteRequestObject) (externalapi.AuthTokensDeleteResponseObject, error) {
	if !hasScope(GetTokenAuth(ctx), "tokens:write") {
		return externalapi.AuthTokensDelete403ApplicationProblemPlusJSONResponse(forbidden("insufficient scope")), nil
	}

	id, err := strconv.ParseInt(string(req.Id), 10, 64)
	if err != nil {
		return externalapi.AuthTokensDelete400ApplicationProblemPlusJSONResponse(badRequest("invalid id")), nil
	}

	now := time.Now()
	_, err = h.ent.ApiToken.UpdateOneID(id).SetRevokedAt(now).Save(ctx)
	if ent.IsNotFound(err) {
		return externalapi.AuthTokensDelete404ApplicationProblemPlusJSONResponse(notFound("token not found")), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.AuthTokensDelete204Response{}, nil
}

func createToken(ctx context.Context, client *ent.Client, name string, scopes []externalapi.ApiTokenScope, expiresAt *externalapi.Timestamp) (*externalapi.CreateApiTokenResponse, error) {
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

	scopeStrings := make([]string, len(scopes))
	for i, s := range scopes {
		scopeStrings[i] = string(s)
	}

	q := client.ApiToken.Create().
		SetName(name).
		SetPrefix(prefix).
		SetSecretHash(hash).
		SetScopes(scopeStrings)

	if expiresAt != nil {
		t := time.Time(*expiresAt)
		q = q.SetExpiresAt(t)
	}

	token, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}

	return &externalapi.CreateApiTokenResponse{
		Token:     service.TokenValue(prefix, secret),
		TokenInfo: toTokenInfo(token),
	}, nil
}

func unauthorized(detail string) externalapi.ProblemDetails {
	return problem(401, "Unauthorized", detail)
}
