// Package webhook delivers domain events to customer-configured HTTP endpoints.
// It owns the SSRF-safe HTTP client and the HMAC signing used by the river
// delivery worker.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"
)

// maxResponseBytes caps how much of a receiver's response body we read.
const maxResponseBytes = 64 << 10

// SignatureHeader carries the hex HMAC-SHA256 of the request body, prefixed
// "sha256=". Receivers recompute it with their endpoint secret to verify.
const SignatureHeader = "X-1mail-Signature"

// EventHeader and DeliveryHeader let receivers route and dedupe deliveries.
const (
	EventHeader    = "X-1mail-Event"
	DeliveryHeader = "X-1mail-Delivery"
)

// Sign returns the signature for a request body under the endpoint secret.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Send POSTs the signed body to url and returns an error for transport failures
// or non-2xx responses (so the caller can retry). The client should be one from
// NewClient so the SSRF guard and redirect/timeout policy apply.
func Send(ctx context.Context, client *http.Client, url, secret, eventName, deliveryID string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(EventHeader, eventName)
	req.Header.Set(DeliveryHeader, deliveryID)
	req.Header.Set(SignatureHeader, Sign(secret, body))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook endpoint returned status %d", resp.StatusCode)
	}
	return nil
}

// NewClient builds an HTTP client hardened for outbound webhook delivery:
//   - SSRF guard: the dial Control hook runs on the *resolved* IP, so a public
//     hostname that resolves to a private/loopback/link-local address (incl. the
//     cloud metadata endpoint) is refused — this defeats DNS rebinding, which a
//     hostname/URL allow-list cannot.
//   - redirects are not followed (a 3xx to an internal URL would bypass the guard).
//   - an overall timeout bounds slow/hanging endpoints.
//
// The caller still must cap the response body it reads.
func NewClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || !isPublicIP(ip) {
				return fmt.Errorf("webhook: refusing to dial non-public address %s", address)
			}
			return nil
		},
	}
	return &http.Client{
		Timeout: timeout,
		// Custom transport with Proxy unset: do NOT switch this to
		// http.DefaultTransport — ProxyFromEnvironment would let an env proxy dial
		// the blocked address and defeat the SSRF guard.
		Transport: &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse // surface the 3xx; don't follow it
		},
	}
}

// nat64Prefix is RFC 6052 well-known NAT64 space, which can embed an IPv4
// internal address (e.g. the metadata endpoint) inside an IPv6 address.
var _, nat64Prefix, _ = net.ParseCIDR("64:ff9b::/96")

// isPublicIP reports whether ip is a routable public address. Deny-by-default:
// only global-unicast addresses pass, and IsGlobalUnicast already excludes
// loopback, link-local (incl. 169.254/16 metadata + fe80::/10), multicast, and
// unspecified. We additionally reject IsPrivate (RFC1918 + IPv6 ULA fc00::/7),
// RFC 6598 shared/CGNAT space (100.64.0.0/10, common in k8s/cloud), and NAT64.
func isPublicIP(ip net.IP) bool {
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return false // 100.64.0.0/10
	}
	if nat64Prefix != nil && nat64Prefix.Contains(ip) {
		return false
	}
	return true
}
