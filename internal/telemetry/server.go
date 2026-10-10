package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// MetricsServer is the opt-in Prometheus listener (ADR 0018): the only place
// /metrics is served. Listen binds the address up front so a bind failure is
// synchronous; Serve then accepts until Shutdown.
type MetricsServer struct {
	srv *http.Server
	ln  net.Listener
}

// NewMetricsServer builds a server for addr (host:port) serving only /metrics.
func NewMetricsServer(addr string) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", MetricsHandler())
	return &MetricsServer{srv: &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}}
}

// Listen binds the configured address.
func (m *MetricsServer) Listen() error {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", m.srv.Addr)
	if err != nil {
		return fmt.Errorf("bind metrics listener %s: %w", m.srv.Addr, err)
	}
	m.ln = ln
	return nil
}

// Addr is the bound address (after Listen), useful with an ephemeral port.
func (m *MetricsServer) Addr() net.Addr { return m.ln.Addr() }

// Serve serves on the bound listener and returns nil after a graceful Shutdown.
func (m *MetricsServer) Serve() error {
	if err := m.srv.Serve(m.ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server.
func (m *MetricsServer) Shutdown(ctx context.Context) error { return m.srv.Shutdown(ctx) }
