package telemetry

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/config"
)

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

func TestMetricsServerServesExpositionOnItsOwnListener(t *testing.T) {
	isolate(t)
	stop, err := Setup(context.Background(), &config.Config{OtelServiceName: "1mail-test"}, "test", BuildInfo{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = stop(context.Background()) })

	srv := NewMetricsServer("127.0.0.1:0")
	require.NoError(t, srv.Listen())
	go func() { _ = srv.Serve() }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })
	base := "http://" + srv.Addr().String()

	code, body := get(t, base+"/metrics")
	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "go_goroutine_count")

	// Tenant-label policy: see TestMetricsExpositionCarriesNoTenantLabels.
	code, _ = get(t, base+"/healthz")
	assert.Equal(t, http.StatusNotFound, code, "only /metrics is served")
}

func TestMetricsServerListenFailsOnOccupiedAddress(t *testing.T) {
	occupied, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = occupied.Close() })

	srv := NewMetricsServer(occupied.Addr().String())
	require.Error(t, srv.Listen())
}

func TestMetricsServerShutdownStopsServing(t *testing.T) {
	srv := NewMetricsServer("127.0.0.1:0")
	require.NoError(t, srv.Listen())
	done := make(chan error, 1)
	go func() { done <- srv.Serve() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, srv.Shutdown(ctx))
	require.NoError(t, <-done, "Serve returns nil after a graceful shutdown")
}
