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
	stdhtml "html"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/net/html"
)

// confirmTTL is how long a double-opt-in confirmation link stays valid. Unlike
// the eternal unsubscribe/tracking tokens, a confirmation link expires (ADR 0013):
// stale consent should not be confirmable weeks later, and never-confirmed tokens
// become purgeable. An expired link routes the recipient to "request a new one".
const confirmTTL = 7 * 24 * time.Hour

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

// UnsubTarget is the self-describing payload an unsubscribe link carries. The
// opt-out is destination-keyed (ADR 0001), so the link records against the
// destination directly — no contact-row lookup, and it survives the contact being
// deleted between send and click. BroadcastID is set only for broadcast sends
// (for per-broadcast attribution); Source is the sending-source string the opt-out
// is scoped to ("broadcasts" / "automation:<id>" / "everything").
type UnsubTarget struct {
	Source      string
	Destination string
	WorkspaceID int64
	ContactID   int64
	BroadcastID int64
}

// Token mints a signed token identifying a broadcast recipient (open/click).
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

// unsubToken mints a signed scoped-unsubscribe token. int64 fields are stored as
// strings: jwt.MapClaims decodes JSON numbers to float64, which loses precision
// above 2^53.
func (t *Tracker) unsubToken(target UnsubTarget) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"src":  target.Source,
		"dest": target.Destination,
		"ws":   strconv.FormatInt(target.WorkspaceID, 10),
		"cid":  strconv.FormatInt(target.ContactID, 10),
		"bid":  strconv.FormatInt(target.BroadcastID, 10),
	})
	return tok.SignedString(t.secret)
}

// DecodeUnsub validates a scoped-unsubscribe token and returns its target.
func (t *Tracker) DecodeUnsub(token string) (UnsubTarget, error) {
	parsed, err := jwt.Parse(token, func(*jwt.Token) (any, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return UnsubTarget{}, err
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return UnsubTarget{}, fmt.Errorf("tracking: unexpected claims type")
	}
	src, _ := claims["src"].(string)
	dest, _ := claims["dest"].(string)
	if src == "" {
		return UnsubTarget{}, fmt.Errorf("tracking: missing src claim")
	}
	ws, err := claimInt64(claims, "ws")
	if err != nil {
		return UnsubTarget{}, err
	}
	cid, err := claimInt64(claims, "cid")
	if err != nil {
		return UnsubTarget{}, err
	}
	bid, err := claimInt64(claims, "bid")
	if err != nil {
		return UnsubTarget{}, err
	}
	return UnsubTarget{Source: src, Destination: dest, WorkspaceID: ws, ContactID: cid, BroadcastID: bid}, nil
}

// ConfirmTarget is the payload a double-opt-in confirmation link carries (ADR
// 0013). Like the opt-out, the confirmation is destination-keyed and not
// source-scoped: confirming an address makes it mailable across sources, so the
// link records against the destination directly (no contact-row lookup) and
// survives the contact being deleted between send and click.
type ConfirmTarget struct {
	Destination string
	WorkspaceID int64
	ContactID   int64
}

// confirmToken mints a signed, expiring confirmation token. int64 fields are
// stored as strings for the same precision reason as unsubToken; the exp claim is
// a real JWT numeric date so DecodeConfirm can reject a stale link.
func (t *Tracker) confirmToken(target ConfirmTarget) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"dest": target.Destination,
		"ws":   strconv.FormatInt(target.WorkspaceID, 10),
		"cid":  strconv.FormatInt(target.ContactID, 10),
		"exp":  time.Now().Add(confirmTTL).Unix(),
	})
	return tok.SignedString(t.secret)
}

// DecodeConfirm validates a confirmation token and returns its target. It
// rejects a token whose exp has passed (and requires exp to be present), so an
// expired link cannot confirm — the endpoint surfaces that as "request a new one".
func (t *Tracker) DecodeConfirm(token string) (ConfirmTarget, error) {
	parsed, err := jwt.Parse(token, func(*jwt.Token) (any, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return ConfirmTarget{}, err
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return ConfirmTarget{}, fmt.Errorf("tracking: unexpected claims type")
	}
	dest, _ := claims["dest"].(string)
	if dest == "" {
		return ConfirmTarget{}, fmt.Errorf("tracking: missing dest claim")
	}
	ws, err := claimInt64(claims, "ws")
	if err != nil {
		return ConfirmTarget{}, err
	}
	cid, err := claimInt64(claims, "cid")
	if err != nil {
		return ConfirmTarget{}, err
	}
	return ConfirmTarget{Destination: dest, WorkspaceID: ws, ContactID: cid}, nil
}

// claimInt64 reads a string-encoded int64 claim.
func claimInt64(claims jwt.MapClaims, key string) (int64, error) {
	s, ok := claims[key].(string)
	if !ok {
		return 0, fmt.Errorf("tracking: missing %s claim", key)
	}
	return strconv.ParseInt(s, 10, 64)
}

// OpenURL is the 1x1 pixel URL that records an open.
func (t *Tracker) OpenURL(token string) string {
	return t.baseURL + "/e/o/" + token
}

// ClickURL wraps a destination link so a click is recorded before redirecting.
func (t *Tracker) ClickURL(token, dest string) string {
	return t.baseURL + "/e/c/" + token + "?u=" + url.QueryEscape(dest)
}

// UnsubscribeURL is the public link that records the scoped opt-out in target.
func (t *Tracker) UnsubscribeURL(target UnsubTarget) (string, error) {
	token, err := t.unsubToken(target)
	if err != nil {
		return "", err
	}
	return t.baseURL + "/e/u/" + token, nil
}

// ConfirmURL is the public double-opt-in link (ADR 0013). Like UnsubscribeURL the
// GET is safe: it renders the SPA confirmation page, and the confirmation is
// recorded only when the recipient POSTs back (scanner-safe, deliberate consent).
func (t *Tracker) ConfirmURL(target ConfirmTarget) (string, error) {
	token, err := t.confirmToken(target)
	if err != nil {
		return "", err
	}
	return t.baseURL + "/e/confirm/" + token, nil
}

// UnsubscribeFooter is the email's unsubscribe footer, scoped to target. Shared by
// the broadcast Rewrite and the automation send step (which appends it directly).
// postalAddress is the sending workspace's physical address (CAN-SPAM); when
// non-empty it is escaped and appended below the unsubscribe line (its own
// newlines become <br>). An empty address renders the footer unchanged — the
// address is a non-gating nudge, not a send-blocker.
func (t *Tracker) UnsubscribeFooter(target UnsubTarget, postalAddress string) (string, error) {
	url, err := t.UnsubscribeURL(target)
	if err != nil {
		return "", err
	}
	var address string
	if addr := strings.TrimSpace(postalAddress); addr != "" {
		address = "<br>" + strings.ReplaceAll(stdhtml.EscapeString(addr), "\n", "<br>")
	}
	return fmt.Sprintf(
		`<p style="font-size:12px;color:#888888;margin-top:24px">`+
			`If you no longer wish to receive these emails, `+
			`<a href="%s">unsubscribe</a>.%s</p>`,
		url, address,
	), nil
}

// Rewrite prepares a broadcast body for a recipient: it routes http(s) links
// through the click tracker (rid token), appends the open pixel, and adds the
// unsubscribe footer (scoped to unsub, carrying the workspace's CAN-SPAM postal
// address). On a tokenizer error it falls back to the original body plus pixel
// and footer so a send is never blocked by odd markup.
func (t *Tracker) Rewrite(body string, recipientID int64, unsub UnsubTarget, postalAddress string) (string, error) {
	token, err := t.Token(recipientID)
	if err != nil {
		return body, err
	}

	rewritten, err := t.rewriteLinks(body, token)
	if err != nil {
		rewritten = body
	}

	footer, err := t.UnsubscribeFooter(unsub, postalAddress)
	if err != nil {
		return body, err
	}
	pixel := fmt.Sprintf(`<img src="%s" width="1" height="1" alt="" style="display:none">`, t.OpenURL(token))
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
