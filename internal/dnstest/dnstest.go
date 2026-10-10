// Package dnstest runs an in-process DNS server for tests, so code that resolves
// names goes through a real *net.Resolver (NXDOMAIN, SERVFAIL, multi-string TXT)
// instead of a hand-built stub.
package dnstest

import (
	"net"
	"testing"

	"github.com/foxcpp/go-mockdns"
)

// Resolver serves zones (keys are fully qualified, with the trailing dot) and
// returns a *net.Resolver wired to the server; the server stops with the test.
// A name outside every zone answers NXDOMAIN; a zone with Err set answers SERVFAIL.
func Resolver(t testing.TB, zones map[string]mockdns.Zone) *net.Resolver {
	t.Helper()
	srv, err := mockdns.NewServerWithLogger(zones, logger{t}, false)
	if err != nil {
		t.Fatalf("start mock DNS server: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	return srv.NewResolver()
}

// logger routes the server's own log lines to the test log.
type logger struct{ t testing.TB }

func (l logger) Printf(format string, args ...any) { l.t.Logf(format, args...) }
