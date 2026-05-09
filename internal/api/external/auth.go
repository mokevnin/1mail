package external

import (
	"context"
	"strconv"
	"time"

	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
	apiauth "github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/api/problems"
	"github.com/mokevnin/1mail/internal/api/resources"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

func (h *Handlers) AuthMeGet(ctx context.Context, req externalapi.AuthMeGetRequestObject) (externalapi.AuthMeGetResponseObject, error) {
	auth := apiauth.GetTokenAuth(ctx)
	if auth == nil {
		return externalapi.AuthMeGet401ApplicationProblemPlusJSONResponse(problems.Unauthorized("missing token").External()), nil
	}
	token, err := h.ent.ApiToken.Get(ctx, auth.TokenID)
	if ent.IsNotFound(err) {
		return externalapi.AuthMeGet404ApplicationProblemPlusJSONResponse(problems.NotFound("token not found").External()), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.AuthMeGet200JSONResponse(resources.ExternalTokenInfo(token)), nil
}

func (h *Handlers) AuthTokensList(ctx context.Context, req externalapi.AuthTokensListRequestObject) (externalapi.AuthTokensListResponseObject, error) {
	if !apiauth.HasScope(apiauth.GetTokenAuth(ctx), "tokens:read") {
		return externalapi.AuthTokensList403ApplicationProblemPlusJSONResponse(problems.Forbidden("insufficient scope").External()), nil
	}

	tokens, err := h.ent.ApiToken.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	items := lo.Map(tokens, func(t *ent.ApiToken, _ int) externalapi.ApiTokenInfo {
		return resources.ExternalTokenInfo(t)
	})
	return externalapi.AuthTokensList200JSONResponse{Items: items}, nil
}

func (h *Handlers) AuthTokensCreate(ctx context.Context, req externalapi.AuthTokensCreateRequestObject) (externalapi.AuthTokensCreateResponseObject, error) {
	if !apiauth.HasScope(apiauth.GetTokenAuth(ctx), "tokens:write") {
		return externalapi.AuthTokensCreate403ApplicationProblemPlusJSONResponse(problems.Forbidden("insufficient scope").External()), nil
	}

	resp, err := createToken(ctx, h.ent, req.Body.Name, req.Body.Scopes, req.Body.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return externalapi.AuthTokensCreate201JSONResponse(*resp), nil
}

func (h *Handlers) AuthTokensBootstrap(ctx context.Context, req externalapi.AuthTokensBootstrapRequestObject) (externalapi.AuthTokensBootstrapResponseObject, error) {
	if h.bootstrapToken == "" || req.Params.XBootstrapToken != h.bootstrapToken {
		return externalapi.AuthTokensBootstrap401ApplicationProblemPlusJSONResponse(problems.Unauthorized("invalid bootstrap token").External()), nil
	}

	resp, err := createToken(ctx, h.ent, req.Body.Name, req.Body.Scopes, req.Body.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return externalapi.AuthTokensBootstrap201JSONResponse(*resp), nil
}

func (h *Handlers) AuthTokensDelete(ctx context.Context, req externalapi.AuthTokensDeleteRequestObject) (externalapi.AuthTokensDeleteResponseObject, error) {
	if !apiauth.HasScope(apiauth.GetTokenAuth(ctx), "tokens:write") {
		return externalapi.AuthTokensDelete403ApplicationProblemPlusJSONResponse(problems.Forbidden("insufficient scope").External()), nil
	}

	id, err := strconv.ParseInt(string(req.Id), 10, 64)
	if err != nil {
		return externalapi.AuthTokensDelete400ApplicationProblemPlusJSONResponse(problems.BadRequest("invalid id").External()), nil
	}

	now := time.Now()
	_, err = h.ent.ApiToken.UpdateOneID(id).SetRevokedAt(now).Save(ctx)
	if ent.IsNotFound(err) {
		return externalapi.AuthTokensDelete404ApplicationProblemPlusJSONResponse(problems.NotFound("token not found").External()), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.AuthTokensDelete204Response{}, nil
}

func createToken(ctx context.Context, client *ent.Client, name string, scopes []externalapi.ApiTokenScope, expiresAt *externalapi.Timestamp) (*externalapi.CreateApiTokenResponse, error) {
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

	scopeStrings := lo.Map(scopes, func(s externalapi.ApiTokenScope, _ int) string {
		return string(s)
	})

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
		return nil, oops.In("external-auth").Public("could not create token").Wrap(err)
	}

	return &externalapi.CreateApiTokenResponse{
		Token:     service.TokenValue(prefix, secret),
		TokenInfo: resources.ExternalTokenInfo(token),
	}, nil
}
