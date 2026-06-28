// Package tracking mints and verifies the per-recipient tokens used by the email
// open/click/unsubscribe endpoints, and rewrites a broadcast's HTML to route
// links through the click tracker and embed the open pixel + unsubscribe footer.
//
// Tokens are signed JWTs (HS256) — we reuse the existing JWT secret rather than
// hand-rolling a MAC. Link rewriting uses the official x/net/html tokenizer, not
// regexes, so arbitrary author markup round-trips safely.
package tracking

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/net/html"
)

// Tracker mints recipient tokens and builds tracking URLs against baseURL.
type Tracker struct {
	secret  []byte
	baseURL string
}

// New builds a Tracker. baseURL is the public origin (e.g. https://1mail.localhost);
// a trailing slash is trimmed.
func New(secret, baseURL string) *Tracker {
	return &Tracker{secret: []byte(secret), baseURL: strings.TrimRight(baseURL, "/")}
}

// Token mints a signed token identifying a broadcast recipient.
func (t *Tracker) Token(recipientID int64) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"rid": strconv.FormatInt(recipientID, 10),
	})
	return tok.SignedString(t.secret)
}

// Decode validates a token and returns the recipient id it carries.
func (t *Tracker) Decode(token string) (int64, error) {
	parsed, err := jwt.Parse(token, func(*jwt.Token) (any, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return 0, err
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("tracking: unexpected claims type")
	}
	rid, ok := claims["rid"].(string)
	if !ok {
		return 0, fmt.Errorf("tracking: missing rid claim")
	}
	return strconv.ParseInt(rid, 10, 64)
}

// OpenURL is the 1x1 pixel URL that records an open.
func (t *Tracker) OpenURL(token string) string {
	return t.baseURL + "/e/o/" + token
}

// ClickURL wraps a destination link so a click is recorded before redirecting.
func (t *Tracker) ClickURL(token, dest string) string {
	return t.baseURL + "/e/c/" + token + "?u=" + url.QueryEscape(dest)
}

// UnsubscribeURL records an unsubscribe.
func (t *Tracker) UnsubscribeURL(token string) string {
	return t.baseURL + "/e/u/" + token
}

// Rewrite prepares a broadcast body for a recipient: it routes http(s) links
// through the click tracker, appends the open pixel, and adds an unsubscribe
// footer. On a tokenizer error it falls back to the original body plus pixel and
// footer so a send is never blocked by odd markup.
func (t *Tracker) Rewrite(body string, recipientID int64) (string, error) {
	token, err := t.Token(recipientID)
	if err != nil {
		return body, err
	}

	rewritten, err := t.rewriteLinks(body, token)
	if err != nil {
		rewritten = body
	}

	pixel := fmt.Sprintf(`<img src="%s" width="1" height="1" alt="" style="display:none">`, t.OpenURL(token))
	footer := fmt.Sprintf(
		`<p style="font-size:12px;color:#888888;margin-top:24px">`+
			`If you no longer wish to receive these emails, `+
			`<a href="%s">unsubscribe</a>.</p>`,
		t.UnsubscribeURL(token),
	)
	return rewritten + pixel + footer, nil
}

func (t *Tracker) rewriteLinks(body, token string) (string, error) {
	z := html.NewTokenizer(strings.NewReader(body))
	var sb strings.Builder
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				return sb.String(), nil
			}
			return "", z.Err()
		}
		tok := z.Token()
		if tt == html.StartTagToken && tok.Data == "a" {
			for i, a := range tok.Attr {
				if a.Key == "href" && trackable(a.Val) {
					tok.Attr[i].Val = t.ClickURL(token, a.Val)
				}
			}
		}
		sb.WriteString(tok.String())
	}
}

// trackable reports whether a link should be routed through the click tracker.
// Only absolute http(s) links are wrapped; mailto/tel/anchors are left alone.
func trackable(href string) bool {
	return strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")
}
