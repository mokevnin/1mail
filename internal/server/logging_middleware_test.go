package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRecovererLogsPanicWithRequestID verifies the middleware chain order:
// requestID must wrap recoverer so a recovered panic is logged with the
// correlation id, and the client still gets a 500 (not a hung/blank response).
func TestRecovererLogsPanicWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	handler := chain(boom, requestID, recoverer)

	req := httptest.NewRequest(http.MethodGet, "/explode", nil)
	req.Header.Set("X-Request-Id", "req-abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := rec.Header().Get("X-Request-Id"); got != "req-abc" {
		t.Fatalf("X-Request-Id header = %q, want req-abc", got)
	}

	var line map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil {
		t.Fatalf("parse panic log line %q: %v", buf.String(), err)
	}
	if line["msg"] != "panic recovered" {
		t.Fatalf("msg = %v, want 'panic recovered'", line["msg"])
	}
	if line["request_id"] != "req-abc" {
		t.Fatalf("request_id = %v, want req-abc", line["request_id"])
	}
	if line["path"] != "/explode" {
		t.Fatalf("path = %v, want /explode", line["path"])
	}
	if stack, _ := line["stack"].(string); !strings.Contains(stack, "logging_middleware_test.go") {
		t.Fatalf("stack does not point at the panic site: %v", line["stack"])
	}
}
