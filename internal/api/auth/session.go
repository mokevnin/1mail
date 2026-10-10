package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	gptoken "github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mokevnin/1mail/ent"
	entuser "github.com/mokevnin/1mail/ent/user"
)

// The session token's claims beyond go-pkgz's own (ADR 0020). Both ride the token's
// user attributes as decimal strings: ClaimUserID is the User's id, the lookup key of
// every request; ClaimEpoch is the User's session_epoch at issuance.
const (
	ClaimUserID = "uid"
	ClaimEpoch  = "epoch"
)

// SessionClaims is the go-pkgz claims updater (Opts.ClaimsUpd): at issuance it
// resolves the login the provider put into the token's user name and writes that
// User's id and current session epoch into the token. A login it cannot resolve
// leaves the claims as they are, and the site security handler rejects the token.
type SessionClaims struct {
	ent *ent.Client
}

func NewSessionClaims(client *ent.Client) *SessionClaims {
	return &SessionClaims{ent: client}
}

var _ gptoken.ClaimsUpdater = (*SessionClaims)(nil)

// Update is the go-pkgz hook; it carries no context, so it stamps under a fresh one.
func (s *SessionClaims) Update(claims gptoken.Claims) gptoken.Claims {
	return s.Stamp(context.Background(), claims)
}

// Stamp writes the id and current session epoch of the User the claims' login names.
func (s *SessionClaims) Stamp(ctx context.Context, claims gptoken.Claims) gptoken.Claims {
	if claims.User == nil || claims.User.Name == "" {
		return claims
	}
	u, err := s.ent.User.Query().Where(entuser.Email(claims.User.Name)).Only(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "session: user of a new token not resolved", "error", err)
		return claims
	}
	claims.User.SetStrAttr(ClaimUserID, strconv.FormatInt(u.ID, 10))
	claims.User.SetStrAttr(ClaimEpoch, strconv.FormatInt(u.SessionEpoch, 10))
	return claims
}

// sessionUser reads the User id and epoch a session token carries. ok is false
// when either is missing or malformed (a token issued before the epoch existed).
func sessionUser(claims gptoken.Claims) (id, epoch int64, ok bool) {
	if claims.User == nil {
		return 0, 0, false
	}
	id, err := strconv.ParseInt(claims.User.StrAttr(ClaimUserID), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	epoch, err = strconv.ParseInt(claims.User.StrAttr(ClaimEpoch), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return id, epoch, true
}

// Sessions writes the site session cookie on the response of the request being
// served (ADR 0020): a revocation point that keeps the acting User signed in
// reissues it under the new epoch, and "sign out everywhere" clears it. ogen
// handlers see no ResponseWriter, so Bind puts the request's writer in its
// context; cookies go through the go-pkgz token service, so they carry the
// login's exact attributes and the claims updater stamps the current epoch.
type Sessions struct {
	tokens *gptoken.Service
	check  *SiteSecurityHandler
}

// NewSessions builds the cookie writer over the go-pkgz service that issues
// logins and the handler that verifies site sessions.
func NewSessions(tokens *gptoken.Service, check *SiteSecurityHandler) *Sessions {
	return &Sessions{tokens: tokens, check: check}
}

type exchange struct {
	w http.ResponseWriter
	r *http.Request
}

var exchangeKey = struct{ name string }{"sessionExchange"}

// Bind makes the request's writer reachable from the handlers next serves.
func (s *Sessions) Bind(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), exchangeKey, &exchange{w: w, r: r})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func exchangeFrom(ctx context.Context) (*exchange, error) {
	ex, ok := ctx.Value(exchangeKey).(*exchange)
	if !ok {
		return nil, errors.New("session: request not bound (Sessions.Bind)")
	}
	return ex, nil
}

// Issue sets a fresh session cookie for the User with the given login email,
// stamped with their current epoch. Call it after the epoch bump has committed:
// a cookie stamped before carries the old epoch and is rejected.
func (s *Sessions) Issue(ctx context.Context, email string) error {
	ex, err := exchangeFrom(ctx)
	if err != nil {
		return err
	}
	_, err = s.tokens.Set(ex.w, gptoken.Claims{
		// Name is what the claims updater resolves; ID is go-pkgz's display id
		// (the session check reads the stamped User id instead).
		User: &gptoken.User{Name: email, ID: "direct_" + gptoken.HashID(sha256.New(), email)},
		RegisteredClaims: jwt.RegisteredClaims{
			ID:       rand.Text(),
			Issuer:   s.tokens.Issuer,
			Audience: jwt.ClaimStrings{s.tokens.Issuer},
		},
		AuthProvider: &gptoken.AuthProvider{Name: "direct"},
	})
	return err
}

// End clears the session cookie of the request being served.
func (s *Sessions) End(ctx context.Context) error {
	ex, err := exchangeFrom(ctx)
	if err != nil {
		return err
	}
	s.tokens.Reset(ex.w)
	return nil
}

// Holder returns the id of the User whose valid session the request carries, for
// public operations that act on a session when one is present. ok is false when
// the request carries none or one that does not verify.
func (s *Sessions) Holder(ctx context.Context) (id int64, ok bool) {
	ex, err := exchangeFrom(ctx)
	if err != nil {
		return 0, false
	}
	c, err := ex.r.Cookie(s.tokens.JWTCookieName)
	if err != nil {
		return 0, false
	}
	u, err := s.check.Verify(ctx, c.Value)
	if err != nil {
		return 0, false
	}
	return u.ID, true
}
