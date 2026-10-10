package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/internal/telemetry"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestHealthz(t *testing.T) {
	env := testhelper.Setup(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	env.Server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}

func TestReadyz(t *testing.T) {
	env := testhelper.Setup(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	env.Server.ServeHTTP(w, req)

	// The txdb test connection is live, so readiness reports OK.
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}

// Metrics live on the opt-in internal listener (ADR 0018), never on the public
// port. Telemetry is set up so that a re-mounted route would serve the exposition.
// (The catch-all answers unknown paths with the SPA shell or the not-embedded hint.)
func TestPublicHandlerDoesNotServeMetrics(t *testing.T) {
	prevMP, prevTP, prevProp := otel.GetMeterProvider(), otel.GetTracerProvider(), otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetMeterProvider(prevMP)
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})
	stop, err := telemetry.Setup(t.Context(), &config.Config{OtelServiceName: "1mail-test"}, "test", telemetry.BuildInfo{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = stop(context.Background()) })

	env := testhelper.Setup(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	env.Server.ServeHTTP(w, req)

	assert.NotContains(t, w.Body.String(), "go_goroutine_count")
	assert.NotContains(t, w.Body.String(), "target_info")
}
