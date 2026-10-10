package smtp_test

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/messaging/smtp"
)

// capture is one message accepted by the fake server.
type capture struct {
	from, rcpt, data, auth string
}

// fakeServer is a minimal in-process SMTP server (no SMTP server library is in
// go.mod). It speaks just enough of RFC 5321 for go-mail: EHLO, AUTH PLAIN, MAIL,
// RCPT, DATA, QUIT. It never offers STARTTLS, like a local relay.
type fakeServer struct {
	port int

	mu       sync.Mutex
	messages []capture
	// rejectData makes the server refuse DATA with a 554.
	rejectData bool
	// dataReply, when set, is the full reply line the server gives to DATA.
	dataReply string
}

func newFakeServer(ctx context.Context, t *testing.T, rejectData bool) *fakeServer {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &fakeServer{port: ln.Addr().(*net.TCPAddr).Port, rejectData: rejectData}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go srv.serve(conn)
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		<-done
	})
	return srv
}

func (s *fakeServer) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	r := bufio.NewReader(conn)
	say := func(format string, args ...any) { _, _ = fmt.Fprintf(conn, format+"\r\n", args...) }

	var cur capture
	say("220 fake ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		cmd := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELLO"):
			say("250-fake")
			say("250 AUTH PLAIN")
		case strings.HasPrefix(cmd, "AUTH PLAIN"):
			cur.auth = strings.TrimSpace(line[len("AUTH PLAIN"):])
			say("235 ok")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			cur.from = line[len("MAIL FROM:"):]
			say("250 ok")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			cur.rcpt = line[len("RCPT TO:"):]
			say("250 ok")
		case cmd == "DATA" && s.dataReply != "":
			say("%s", s.dataReply)
		case cmd == "DATA":
			if s.rejectData {
				say("554 rejected")
				continue
			}
			say("354 go ahead")
			var body strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				body.WriteString(l)
			}
			cur.data = body.String()
			s.mu.Lock()
			s.messages = append(s.messages, cur)
			s.mu.Unlock()
			cur = capture{}
			say("250 queued")
		case cmd == "RSET", cmd == "NOOP":
			say("250 ok")
		case cmd == "QUIT":
			say("221 bye")
			return
		default:
			say("502 unsupported")
		}
	}
}

func (s *fakeServer) captured() []capture {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]capture(nil), s.messages...)
}

func build(t *testing.T, cfg smtp.Config) messaging.EmailSender {
	t.Helper()
	raw := fmt.Sprintf(`{"host":%q,"port":%d,"username":%q,"password":%q,"from":%q,"fromName":%q}`,
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.From, cfg.FromName)
	built, err := smtp.Descriptor().Build([]byte(raw), nil)
	require.NoError(t, err)
	sender, ok := built.(messaging.EmailSender)
	require.True(t, ok)
	return sender
}

func TestSendDeliversWithConfiguredDefaults(t *testing.T) {
	srv := newFakeServer(t.Context(), t, false)
	sender := build(t, smtp.Config{Host: "127.0.0.1", Port: srv.port, From: "noreply@acme.com", FromName: "Acme"})

	receipt, err := sender.Send(context.Background(), messaging.EmailMessage{
		To: "rcpt@example.com", Subject: "Hello", Text: "plain body",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, receipt.MessageID)

	got := srv.captured()
	require.Len(t, got, 1)
	assert.Contains(t, got[0].from, "noreply@acme.com", "envelope sender falls back to the integration From")
	assert.Contains(t, got[0].rcpt, "rcpt@example.com")
	assert.Contains(t, got[0].data, "From: \"Acme\" <noreply@acme.com>")
	assert.Contains(t, got[0].data, "Subject: Hello")
	assert.Contains(t, got[0].data, "plain body")
	assert.Contains(t, got[0].data, strings.Trim(receipt.MessageID, "<>"))
	assert.Empty(t, got[0].auth, "no credentials configured means no AUTH")
}

func TestSendMessageFromOverridesDefault(t *testing.T) {
	srv := newFakeServer(t.Context(), t, false)
	sender := build(t, smtp.Config{Host: "127.0.0.1", Port: srv.port, From: "noreply@acme.com", FromName: "Acme"})

	_, err := sender.Send(context.Background(), messaging.EmailMessage{
		From: "news@acme.com", FromName: "News", To: "rcpt@example.com", Subject: "s", HTML: "<p>x</p>",
	})
	require.NoError(t, err)

	got := srv.captured()
	require.Len(t, got, 1)
	assert.Contains(t, got[0].from, "news@acme.com")
	assert.Contains(t, got[0].data, "News")
}

func TestSendAuthenticatesWhenCredentialsSet(t *testing.T) {
	srv := newFakeServer(t.Context(), t, false)
	sender := build(t, smtp.Config{
		Host: "127.0.0.1", Port: srv.port, Username: "user", Password: "secret", From: "noreply@acme.com",
	})

	_, err := sender.Send(context.Background(), messaging.EmailMessage{To: "rcpt@example.com", Subject: "s", Text: "x"})
	require.NoError(t, err)

	got := srv.captured()
	require.Len(t, got, 1)
	decoded, err := base64.StdEncoding.DecodeString(got[0].auth)
	require.NoError(t, err)
	assert.Equal(t, "\x00user\x00secret", string(decoded))
}

func TestSendErrors(t *testing.T) {
	ctx := t.Context()
	msg := messaging.EmailMessage{To: "rcpt@example.com", Subject: "s", Text: "x"}

	t.Run("invalid message", func(t *testing.T) {
		srv := newFakeServer(ctx, t, false)
		sender := build(t, smtp.Config{Host: "127.0.0.1", Port: srv.port, From: "noreply@acme.com"})
		_, err := sender.Send(ctx, messaging.EmailMessage{To: "not-an-address", Text: "x"})
		assert.ErrorContains(t, err, "invalid to address")
		assert.Empty(t, srv.captured())
	})

	t.Run("server rejects DATA", func(t *testing.T) {
		srv := newFakeServer(ctx, t, true)
		sender := build(t, smtp.Config{Host: "127.0.0.1", Port: srv.port, From: "noreply@acme.com"})
		_, err := sender.Send(ctx, msg)
		assert.Error(t, err)
		assert.Empty(t, srv.captured())
	})

	t.Run("transient too-fast reply is busy", func(t *testing.T) {
		srv := newFakeServer(ctx, t, false)
		srv.dataReply = "454 4.7.0 Throttling failure: Maximum sending rate exceeded."
		sender := build(t, smtp.Config{Host: "127.0.0.1", Port: srv.port, From: "noreply@acme.com"})
		_, err := sender.Send(ctx, msg)
		assert.ErrorIs(t, err, messaging.ErrBusy)
		assert.NotErrorIs(t, err, messaging.ErrQuotaExceeded)
	})

	t.Run("transient daily quota reply is quota exceeded", func(t *testing.T) {
		srv := newFakeServer(ctx, t, false)
		srv.dataReply = "454 4.7.0 Throttling failure: Daily message quota exceeded."
		sender := build(t, smtp.Config{Host: "127.0.0.1", Port: srv.port, From: "noreply@acme.com"})
		_, err := sender.Send(ctx, msg)
		assert.ErrorIs(t, err, messaging.ErrQuotaExceeded)
	})

	t.Run("permanent reply is never busy", func(t *testing.T) {
		srv := newFakeServer(ctx, t, false)
		srv.dataReply = "554 5.7.1 rate limit policy violation, rejected"
		sender := build(t, smtp.Config{Host: "127.0.0.1", Port: srv.port, From: "noreply@acme.com"})
		_, err := sender.Send(ctx, msg)
		require.Error(t, err)
		assert.NotErrorIs(t, err, messaging.ErrBusy)
	})

	t.Run("unreachable host", func(t *testing.T) {
		ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port := ln.Addr().(*net.TCPAddr).Port
		require.NoError(t, ln.Close())
		sender := build(t, smtp.Config{Host: "127.0.0.1", Port: port, From: "noreply@acme.com"})
		_, err = sender.Send(ctx, msg)
		assert.Error(t, err)
	})

	t.Run("invalid client options", func(t *testing.T) {
		// A port outside 1..65535 is rejected by go-mail when building the client.
		sender := build(t, smtp.Config{Host: "127.0.0.1", Port: 70000, From: "noreply@acme.com"})
		_, err := sender.Send(ctx, msg)
		assert.ErrorContains(t, err, "smtp: client")
	})
}

func TestDefaultFrom(t *testing.T) {
	sender := build(t, smtp.Config{Host: "h", Port: 25, From: "noreply@acme.com", FromName: "Acme"})
	df, ok := sender.(messaging.DefaultFromer)
	require.True(t, ok)
	addr, name := df.DefaultFrom()
	assert.Equal(t, "noreply@acme.com", addr)
	assert.Equal(t, "Acme", name)
}

func TestDescriptorValidate(t *testing.T) {
	d := smtp.Descriptor()
	assert.Equal(t, messaging.ChannelEmail, d.Channel)
	assert.Equal(t, messaging.ProviderSMTP, d.Provider)

	cases := map[string]struct {
		cfg     string
		wantErr string
	}{
		"valid":        {`{"host":"h","port":25,"from":"a@b.com"}`, ""},
		"malformed":    {`{`, "invalid config"},
		"missing host": {`{"port":25,"from":"a@b.com"}`, "host is required"},
		"bad port":     {`{"host":"h","port":0,"from":"a@b.com"}`, "port must be positive"},
		"missing from": {`{"host":"h","port":25}`, "from is required"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := d.Validate([]byte(tc.cfg))
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestBuildRejectsMalformedConfig(t *testing.T) {
	_, err := smtp.Descriptor().Build([]byte(`{`), nil)
	assert.ErrorContains(t, err, "invalid config")
}
