//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

// Address is a mailbox as Mailpit reports it.
type Address struct {
	Name    string
	Address string
}

// Summary is one inbox row, as listed.
type Summary struct {
	ID      string
	From    Address
	To      []Address
	Subject string
}

// Message is a delivered email with the fields the scenarios assert on.
type Message struct {
	ID      string
	From    Address
	To      []Address
	Subject string
	Text    string
	HTML    string
	// Headers are the raw header values keyed by canonical name, so a scenario can
	// read List-Unsubscribe and List-Unsubscribe-Post exactly as they went on the wire.
	Headers map[string][]string
}

// Header returns the first value of the named header ("" when absent).
func (m Message) Header(name string) string {
	return first(m.Headers[http.CanonicalHeaderKey(name)])
}

func first(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// Mailpit is a thin client over Mailpit's HTTP API (it publishes only Swagger 2.0,
// which the project's OpenAPI generator rejects, and three endpoints are enough).
type Mailpit struct {
	// SMTPHost and SMTPPort are where a Workspace's SMTP Integration should point.
	SMTPHost string
	SMTPPort int

	base string
	hc   *http.Client
	poll time.Duration
}

// NewMailpit builds a client for the Mailpit HTTP API at base ("http://127.0.0.1:8025").
func NewMailpit(base string) *Mailpit {
	return &Mailpit{base: strings.TrimRight(base, "/"), hc: &http.Client{Timeout: 10 * time.Second}, poll: 100 * time.Millisecond}
}

type apiAddress struct{ Name, Address string }

func (a apiAddress) toAddress() Address { return Address(a) }

func addresses(in []apiAddress) []Address {
	out := make([]Address, len(in))
	for i, a := range in {
		out[i] = a.toAddress()
	}
	return out
}

func (m *Mailpit) do(ctx context.Context, method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, m.base+path, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := m.hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("mailpit %s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(b)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// List returns the newest inbox rows (Mailpit pages at 50 by default; limit raises it).
func (m *Mailpit) List(ctx context.Context, limit int) ([]Summary, error) {
	return m.summaries(ctx, "/api/v1/messages", url.Values{"limit": {fmt.Sprint(limit)}})
}

// Search returns the inbox rows matching a Mailpit search query (e.g. `to:a@b.test`).
func (m *Mailpit) Search(ctx context.Context, query string) ([]Summary, error) {
	return m.summaries(ctx, "/api/v1/search", url.Values{"query": {query}, "limit": {"100"}})
}

func (m *Mailpit) summaries(ctx context.Context, path string, q url.Values) ([]Summary, error) {
	var resp struct {
		Messages []struct {
			ID      string `json:"ID"`
			From    apiAddress
			To      []apiAddress
			Subject string
		} `json:"messages"`
	}
	if err := m.do(ctx, http.MethodGet, path+"?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	out := make([]Summary, len(resp.Messages))
	for i, s := range resp.Messages {
		out[i] = Summary{ID: s.ID, From: s.From.toAddress(), To: addresses(s.To), Subject: s.Subject}
	}
	return out, nil
}

// Get loads one message with its body and raw headers.
func (m *Mailpit) Get(ctx context.Context, id string) (Message, error) {
	var msg struct {
		ID      string `json:"ID"`
		From    apiAddress
		To      []apiAddress
		Subject string
		Text    string
		HTML    string
	}
	if err := m.do(ctx, http.MethodGet, "/api/v1/message/"+url.PathEscape(id), nil, &msg); err != nil {
		return Message{}, err
	}
	var raw map[string][]string
	if err := m.do(ctx, http.MethodGet, "/api/v1/message/"+url.PathEscape(id)+"/headers", nil, &raw); err != nil {
		return Message{}, err
	}
	headers := make(map[string][]string, len(raw))
	for k, v := range raw {
		headers[http.CanonicalHeaderKey(k)] = v
	}
	return Message{ID: msg.ID, From: msg.From.toAddress(), To: addresses(msg.To), Subject: msg.Subject, Text: msg.Text, HTML: msg.HTML, Headers: headers}, nil
}

// Delete removes the given messages. Parallel tests delete only their own ids, never
// the whole inbox.
func (m *Mailpit) Delete(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	return m.do(ctx, http.MethodDelete, "/api/v1/messages", map[string][]string{"IDs": ids}, nil)
}

// NoMailError is WaitForRecipient's timeout: it lists what the inbox held instead,
// so a failing scenario shows whether mail went to the wrong address or nowhere.
type NoMailError struct {
	Recipient string
	Waited    time.Duration
	Inbox     []Summary
	// ListErr is set when the diagnostic listing itself failed.
	ListErr error
}

func (e *NoMailError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "no email for %s within %s; ", e.Recipient, e.Waited)
	switch {
	case e.ListErr != nil:
		fmt.Fprintf(&b, "could not list the inbox: %v", e.ListErr)
	case len(e.Inbox) == 0:
		b.WriteString("the inbox is empty")
	default:
		fmt.Fprintf(&b, "the inbox holds %d message(s):", len(e.Inbox))
		for _, s := range e.Inbox {
			to := make([]string, len(s.To))
			for i, a := range s.To {
				to[i] = a.Address
			}
			fmt.Fprintf(&b, "\n  - to=[%s] from=%s subject=%q", strings.Join(to, ", "), s.From.Address, s.Subject)
		}
	}
	return b.String()
}

// WaitForRecipient polls until a message addressed to recipient is in the inbox and
// returns it. It gives up after timeout with a *NoMailError that lists the inbox.
func (m *Mailpit) WaitForRecipient(ctx context.Context, recipient string, timeout time.Duration) (Message, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var lastErr error
	for {
		found, err := m.Search(ctx, `to:"`+recipient+`"`)
		if err == nil && len(found) > 0 {
			// Search is a substring match; insist on the exact recipient.
			for _, s := range found {
				for _, a := range s.To {
					if strings.EqualFold(a.Address, recipient) {
						return m.Get(context.WithoutCancel(ctx), s.ID)
					}
				}
			}
		}
		lastErr = err
		select {
		case <-ctx.Done():
			diag, lerr := m.List(context.WithoutCancel(ctx), 100)
			if lerr == nil && lastErr != nil {
				lerr = lastErr
			}
			return Message{}, &NoMailError{Recipient: recipient, Waited: timeout, Inbox: diag, ListErr: lerr}
		case <-time.After(m.poll):
		}
	}
}

// IsNoMail reports whether err is a WaitForRecipient timeout.
func IsNoMail(err error) bool {
	var e *NoMailError
	return errors.As(err, &e)
}

// StartMailpit runs the toolchain's mailpit binary on two free loopback ports and
// returns a client plus a stop function. The inbox keeps every message (--max).
func StartMailpit(ctx context.Context) (*Mailpit, func(), error) {
	smtpPort, err := freePort(ctx)
	if err != nil {
		return nil, nil, err
	}
	httpPort, err := freePort(ctx)
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, "mailpit",
		"--smtp", fmt.Sprintf("127.0.0.1:%d", smtpPort),
		"--listen", fmt.Sprintf("127.0.0.1:%d", httpPort),
		"--max", "1000000", "--quiet", "--disable-version-check")
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start mailpit (is it on PATH? run through mise): %w", err)
	}
	stop := func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	mp := NewMailpit(fmt.Sprintf("http://127.0.0.1:%d", httpPort))
	mp.SMTPHost, mp.SMTPPort = "127.0.0.1", smtpPort

	deadline, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	for {
		if _, err := mp.List(deadline, 1); err == nil {
			return mp, stop, nil
		}
		select {
		case <-deadline.Done():
			stop()
			return nil, nil, errors.New("mailpit did not become ready")
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// freePort asks the kernel for a free loopback port. The tiny window between closing
// the probe and mailpit binding is accepted; mailpit then fails loudly on a clash.
func freePort(ctx context.Context) (int, error) {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().(*net.TCPAddr).Port, nil
}
