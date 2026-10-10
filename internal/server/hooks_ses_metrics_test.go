package server_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/server"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// SES bounce and complaint notifications are counted by provider and status.
func TestSESHookCountsBounceAndComplaintOutcomes(t *testing.T) {
	env := testhelper.Setup(t)
	testhelper.StartMetrics(t)
	h := server.NewSESHooks(env.DB, env.Bus, acceptAll, noConfirm)

	bounce := `{"notificationType":"Bounce","mail":{"source":"hello@codebasics.dev"},
	  "bounce":{"bounceType":"Permanent","bouncedRecipients":[{"emailAddress":"` + strings.ToUpper(fixtures.ContactAliceEmail) + `"}]}}`
	require.Equal(t, 200, postSES(t, h, fixtures.AcmeIngestKey, sesNotificationBody(t, bounce)).Code)
	complaint := `{"eventType":"Complaint","complaint":{"complainedRecipients":[{"emailAddress":"stranger@example.com"}]}}`
	require.Equal(t, 200, postSES(t, h, fixtures.AcmeIngestKey, sesNotificationBody(t, complaint)).Code)

	scrape := testhelper.ScrapeMetrics(t)
	assert.InDelta(t, 1, scrape.Value(t, "email_send_outcomes_total", map[string]string{"provider": "ses", "status": "bounce"}), 0)
	assert.InDelta(t, 1, scrape.Value(t, "email_send_outcomes_total", map[string]string{"provider": "ses", "status": "complaint"}), 0)
}
