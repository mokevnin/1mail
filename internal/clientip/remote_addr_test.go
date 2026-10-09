package clientip_test

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mokevnin/1mail/internal/clientip"
)

func TestFromRequestUsesTheRawRemoteAddrWhenItHasNoPort(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), "GET", "/", nil)
	r.RemoteAddr = "203.0.113.9"
	assert.Equal(t, "203.0.113.9", clientip.FromRequest(r))
}

func TestFromRequestTakesTheFirstForwardedHop(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), "GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:4000"
	r.Header.Set("X-Forwarded-For", " 198.51.100.7 , 10.0.0.2")
	assert.Equal(t, "198.51.100.7", clientip.FromRequest(r))
}
