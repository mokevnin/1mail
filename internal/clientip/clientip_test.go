package clientip_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mokevnin/1mail/internal/clientip"
	"github.com/stretchr/testify/assert"
)

func TestFromRequest(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.9:5555"
	assert.Equal(t, "10.0.0.9", clientip.FromRequest(r), "remote host without a proxy header")

	r.Header.Set("X-Forwarded-For", "203.0.113.7")
	assert.Equal(t, "203.0.113.7", clientip.FromRequest(r), "the hop our proxy appended wins")
}

func TestMiddlewareExposesAddressToContext(t *testing.T) {
	var got string
	h := clientip.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = clientip.FromContext(r.Context())
	}))
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.9:5555"
	h.ServeHTTP(httptest.NewRecorder(), r)
	assert.Equal(t, "10.0.0.9", got)
	assert.Empty(t, clientip.FromContext(t.Context()))
}
