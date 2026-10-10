package auth

import (
	"context"
	"log/slog"
	"strconv"

	gptoken "github.com/go-pkgz/auth/v2/token"
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
