package external

import (
	"context"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/apitoken"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/lo"
)

type TokenAuth struct {
	TokenID int64
	Name    string
	Scopes  []string
}

type contextKey struct{}

var tokenAuthKey = contextKey{}

func WithTokenAuth(ctx context.Context, auth *TokenAuth) context.Context {
	return context.WithValue(ctx, tokenAuthKey, auth)
}

func GetTokenAuth(ctx context.Context) *TokenAuth {
	v, _ := ctx.Value(tokenAuthKey).(*TokenAuth)
	return v
}

func hasScope(auth *TokenAuth, scope string) bool {
	if auth == nil {
		return false
	}
	return lo.Contains(auth.Scopes, scope)
}

func MakeTokenValidator(client *ent.Client) middleware.KeyAuthValidator {
	return func(key string, c echo.Context) (bool, error) {
		parsed := service.ParseToken(key)
		if parsed == nil {
			return false, nil
		}

		token, err := client.ApiToken.Query().
			Where(apitoken.Prefix(parsed.Prefix)).
			First(c.Request().Context())
		if err != nil {
			return false, nil
		}
		if token.RevokedAt != nil {
			return false, nil
		}
		if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
			return false, nil
		}
		if !service.VerifyTokenSecret(parsed.Secret, token.SecretHash) {
			return false, nil
		}

		// fire-and-forget last_used_at update
		go func() {
			now := time.Now()
			client.ApiToken.UpdateOneID(token.ID).SetLastUsedAt(now).Exec(context.Background())
		}()

		auth := &TokenAuth{
			TokenID: token.ID,
			Name:    token.Name,
			Scopes:  token.Scopes,
		}
		c.SetRequest(c.Request().WithContext(
			WithTokenAuth(c.Request().Context(), auth),
		))
		return true, nil
	}
}

func MakeCollectKeyValidator(configuredKey string) middleware.KeyAuthValidator {
	return func(key string, c echo.Context) (bool, error) {
		if configuredKey == "" {
			return false, echo.NewHTTPError(503, "collect key not configured")
		}
		return key == configuredKey, nil
	}
}

// Handlers is the combined struct implementing the external StrictServerInterface.
type Handlers struct {
	ent            *ent.Client
	bootstrapToken string
}

func NewHandlers(client *ent.Client, bootstrapToken string) *Handlers {
	return &Handlers{ent: client, bootstrapToken: bootstrapToken}
}

func toContactResource(c *ent.Contact) externalapi.ContactResource {
	r := externalapi.ContactResource{
		Id:        itoa(c.ID),
		Email:     externalapi.EmailAddress(c.Email),
		Status:    externalapi.ContactStatus(c.Status),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		FirstName: c.FirstName,
		LastName:  c.LastName,
		TimeZone:  c.TimeZone,
	}
	if c.CustomFields != nil {
		cf := map[string]string(c.CustomFields)
		r.CustomFields = &cf
	}
	return r
}

func toTokenInfo(t *ent.ApiToken) externalapi.ApiTokenInfo {
	scopes := lo.Map(t.Scopes, func(s string, _ int) externalapi.ApiTokenScope {
		return externalapi.ApiTokenScope(s)
	})
	return externalapi.ApiTokenInfo{
		Id:         itoa(t.ID),
		Name:       t.Name,
		Scopes:     scopes,
		ExpiresAt:  t.ExpiresAt,
		RevokedAt:  t.RevokedAt,
		LastUsedAt: t.LastUsedAt,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func itoa(id int64) string {
	return externalapi.EntityId(strconv.FormatInt(id, 10))
}

type fieldErrors map[string][]string

func conflict(detail string) externalapi.ProblemDetails { return problem(409, "Conflict", detail) }
func conflictWithErrors(detail string, errors fieldErrors) externalapi.ProblemDetails {
	return problemWithErrors(409, "Conflict", detail, errors)
}
func notFound(detail string) externalapi.ProblemDetails   { return problem(404, "Not Found", detail) }
func badRequest(detail string) externalapi.ProblemDetails { return problem(400, "Bad Request", detail) }
func forbidden(detail string) externalapi.ProblemDetails  { return problem(403, "Forbidden", detail) }
func notImpl(detail string) externalapi.ProblemDetails {
	return problem(501, "Not Implemented", detail)
}

func problem(status int32, title, detail string) externalapi.ProblemDetails {
	return externalapi.ProblemDetails{
		Status: &status,
		Title:  strptr(title),
		Detail: strptr(detail),
	}
}

func problemWithErrors(status int32, title, detail string, errors fieldErrors) externalapi.ProblemDetails {
	p := problem(status, title, detail)
	p.Errors = ptr(map[string][]string(errors))
	return p
}

func strptr(s string) *string { return &s }

func ptr[T any](v T) *T { return &v }

var _ externalapi.StrictServerInterface = (*Handlers)(nil)
