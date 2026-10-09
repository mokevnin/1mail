// Package clientip carries the best-effort client address of a request through the
// context, so ogen handlers (which never see the *http.Request) can record it as
// proof, e.g. on a double-opt-in confirmation.
package clientip

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type ctxKey struct{}

// FromRequest prefers the first hop of X-Forwarded-For (the binary runs behind
// Caddy/ingress) and falls back to the connection's remote host.
func FromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(first)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// Middleware stores FromRequest in the request context for FromContext.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, FromRequest(r))))
	})
}

// FromContext returns the address stored by Middleware, or "" outside a request.
func FromContext(ctx context.Context) string {
	ip, _ := ctx.Value(ctxKey{}).(string)
	return ip
}
