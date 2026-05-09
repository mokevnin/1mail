package auth

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/apitoken"
	entuser "github.com/mokevnin/1mail/ent/user"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/lo"
	"golang.org/x/crypto/bcrypt"
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

func HasScope(auth *TokenAuth, scope string) bool {
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

		go func() {
			now := time.Now()
			_ = client.ApiToken.UpdateOneID(token.ID).SetLastUsedAt(now).Exec(context.Background())
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

type CredChecker struct {
	ent *ent.Client
}

func NewCredChecker(client *ent.Client) *CredChecker {
	return &CredChecker{ent: client}
}

func (c *CredChecker) Check(user, password string) (bool, error) {
	u, err := c.ent.User.Query().Where(entuser.Email(user)).Only(context.Background())
	if ent.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if u.PasswordHash == "" {
		return false, nil
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil, nil
}
