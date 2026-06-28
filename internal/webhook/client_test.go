package webhook_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mokevnin/1mail/internal/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignMatchesReceiverRecomputation(t *testing.T) {
	secret := "whsec_abc"
	body := []byte(`{"hello":"world"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	assert.Equal(t, want, webhook.Sign(secret, body))
}

// The hardened client refuses to dial loopback (and by extension private/
// link-local) addresses — the core SSRF guard. httptest binds to 127.0.0.1.
func TestNewClientBlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := webhook.NewClient(2 * time.Second).Get(srv.URL)
	require.Error(t, err, "must refuse to dial a loopback address")
}

// Send signs the body with the endpoint secret and treats 2xx as success.
func TestSendSignsBodyAndSucceedsOn2xx(t *testing.T) {
	secret := "whsec_xyz"
	body := []byte(`{"id":"evt_1","type":"contact.created"}`)

	var gotSig, gotEvent, gotDelivery string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get(webhook.SignatureHeader)
		gotEvent = r.Header.Get(webhook.EventHeader)
		gotDelivery = r.Header.Get(webhook.DeliveryHeader)
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	// Plain client: httptest is loopback, which the hardened client would block.
	err := webhook.Send(context.Background(), srv.Client(), srv.URL, secret, "contact.created", "evt_1", body)
	require.NoError(t, err)

	assert.Equal(t, "contact.created", gotEvent)
	assert.Equal(t, "evt_1", gotDelivery)
	assert.Equal(t, body, gotBody)
	// Receiver recomputes the signature over the received body and it matches.
	assert.Equal(t, webhook.Sign(secret, gotBody), gotSig)
	assert.True(t, hmac.Equal([]byte(gotSig), []byte(webhook.Sign(secret, body))))
}

func TestSendErrorsOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := webhook.Send(context.Background(), srv.Client(), srv.URL, "s", "x", "d", []byte(`{}`))
	require.Error(t, err, "non-2xx must error so river retries")
}
