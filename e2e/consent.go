//go:build e2e

package e2e

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-faster/jx"
	"github.com/stretchr/testify/require"

	externalapi "github.com/mokevnin/1mail/gen/external"
)

// angleURL picks the <...> entries of a List-Unsubscribe value (RFC 2369).
var angleURL = regexp.MustCompile(`<([^>]*)>`)

// OneClickUnsubscribe performs the RFC 8058 one-click request exactly as a mailbox
// provider would: it uses only the https URL from the message's List-Unsubscribe
// header and the body from List-Unsubscribe-Post, and requires success (2xx).
func (w *Workspace) OneClickUnsubscribe(msg Message) {
	w.t.Helper()
	var target string
	for _, m := range angleURL.FindAllStringSubmatch(msg.Header("List-Unsubscribe"), -1) {
		if strings.HasPrefix(m[1], "https://") || strings.HasPrefix(m[1], "http://") {
			target = m[1]
			break
		}
	}
	require.NotEmpty(w.t, target, "List-Unsubscribe carries an http(s) URL, got %q", msg.Header("List-Unsubscribe"))
	body := msg.Header("List-Unsubscribe-Post")
	require.Equal(w.t, "List-Unsubscribe=One-Click", body, "List-Unsubscribe-Post announces one-click")

	req, err := http.NewRequestWithContext(w.t.Context(), http.MethodPost, target, strings.NewReader(body))
	require.NoError(w.t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	require.NoError(w.t, err)
	out, err := io.ReadAll(res.Body)
	require.NoError(w.t, err)
	require.NoError(w.t, res.Body.Close())
	require.Truef(w.t, res.StatusCode/100 == 2, "one-click POST to %s answered %d: %s", target, res.StatusCode, out)
}

// RequireUnsubscribed asserts, through the external API, that the Contact with this
// email has opted out of Broadcasts. The API reads it from the Contact's data export
// (its opt-outs); the opt-out is recorded asynchronously, so it is polled.
func (w *Workspace) RequireUnsubscribed(email string) {
	w.t.Helper()
	ctx := w.t.Context()
	require.Eventually(w.t, func() bool {
		res, err := w.api.ContactsExport(ctx, externalapi.ContactsExportParams{Email: externalapi.NewOptEmailAddress(externalapi.EmailAddress(email))})
		if err != nil {
			return false
		}
		dl, isDL := res.(*externalapi.ContactsExportOKApplicationOctetStreamHeaders)
		if !isDL {
			return false
		}
		raw, err := io.ReadAll(dl.Response)
		if err != nil {
			return false
		}
		var doc externalapi.ContactExportDocument
		if err := doc.Decode(jx.DecodeBytes(raw)); err != nil {
			return false
		}
		for _, u := range doc.Unsubscribes {
			if strings.EqualFold(u.Destination, email) && u.SendingSource == "broadcasts" {
				return true
			}
		}
		return false
	}, EmailTimeout, 100*time.Millisecond, "%s never read as unsubscribed from broadcasts", email)
}

// WaitForEmailWithSubject is WaitForEmail for a specific message: it waits for one
// addressed to recipient whose subject is exactly subject (an earlier Broadcast's
// mail to the same address does not count).
func (w *Workspace) WaitForEmailWithSubject(recipient, subject string) Message {
	w.t.Helper()
	require.Eventually(w.t, func() bool { return w.find(recipient, subject) != "" },
		EmailTimeout, 100*time.Millisecond, "no email %q for %s", subject, recipient)
	msg, err := w.env.Mailpit.Get(context.WithoutCancel(w.t.Context()), w.find(recipient, subject))
	require.NoError(w.t, err)
	w.msgIDs = append(w.msgIDs, msg.ID)
	return msg
}

// RequireNoEmailWithSubject asserts that no message with subject reaches recipient
// within window. Absence cannot be awaited, so callers first wait for a sibling's
// delivery of the same send and keep the window short.
func (w *Workspace) RequireNoEmailWithSubject(recipient, subject string, window time.Duration) {
	w.t.Helper()
	require.Never(w.t, func() bool { return w.find(recipient, subject) != "" },
		window, 100*time.Millisecond, "unexpected email %q for %s", subject, recipient)
}

// find returns the id of the inbox message for recipient with subject, or "".
func (w *Workspace) find(recipient, subject string) string {
	found, err := w.env.Mailpit.Search(w.t.Context(), `to:"`+recipient+`"`)
	if err != nil {
		return ""
	}
	for _, s := range found {
		if s.Subject != subject {
			continue
		}
		for _, a := range s.To {
			if strings.EqualFold(a.Address, recipient) {
				return s.ID
			}
		}
	}
	return ""
}
