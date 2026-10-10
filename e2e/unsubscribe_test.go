//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func body(text string) string {
	return "<mjml><mj-body><mj-section><mj-column><mj-text>" + text + "</mj-text></mj-column></mj-section></mj-body></mjml>"
}

func TestOneClickUnsubscribeStopsTheNextBroadcast(t *testing.T) {
	t.Parallel()
	w := env.NewWorkspace(t).Ready()
	leaver, stayer := w.NewRecipient(), w.NewRecipient()
	w.ImportContacts(leaver, stayer)

	first := "First " + uniq()
	w.SendBroadcast(Broadcast{Subject: first, Body: body("First issue")})
	delivered := w.WaitForEmailWithSubject(leaver, first)
	w.WaitForEmailWithSubject(stayer, first)

	// Only what the email itself carries: URL and body from its headers.
	w.OneClickUnsubscribe(delivered)
	w.RequireUnsubscribed(leaver)

	second := "Second " + uniq()
	w.SendBroadcast(Broadcast{Subject: second, Body: body("Second issue")})
	w.WaitForEmailWithSubject(stayer, second) // the send ran; now absence is meaningful
	w.RequireNoEmailWithSubject(leaver, second, 3*time.Second)
	assert.NotEmpty(t, delivered.Header("List-Unsubscribe"))
}
