//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	externalapi "github.com/mokevnin/1mail/gen/external"
)

// Automation is what a scenario chooses about an Automation: one email step, no waits.
type Automation struct {
	Name         string
	TriggerEvent string
	Subject      string
	// Body is MJML.
	Body string
}

// ActivateAutomation creates an Automation with a single email step (no wait steps, so
// the email goes out as soon as a Contact is enrolled) and activates it. Creation
// always stores a draft; activating is a separate step.
func (w *Workspace) ActivateAutomation(a Automation) {
	w.t.Helper()
	ctx := w.t.Context()
	if a.Name == "" {
		a.Name = "e2e automation " + uniq()
	}
	res, err := w.api.AutomationsCreate(ctx, &externalapi.CreateAutomationInput{
		Name:         a.Name,
		TriggerEvent: a.TriggerEvent,
		Steps: []externalapi.AutomationStep{{
			Type:    externalapi.AutomationStepTypeEmail,
			Subject: externalapi.NewOptString(a.Subject),
			Body:    externalapi.NewOptString(a.Body),
		}},
	})
	created := ok[externalapi.AutomationResource](w.t, "create automation", res, err)

	act, err := w.api.AutomationsActivate(ctx, externalapi.AutomationsActivateParams{ID: created.ID})
	got := ok[externalapi.AutomationResource](w.t, "activate automation", act, err)
	require.Equal(w.t, externalapi.AutomationStatusActive, got.Status)
}

// RecordEvent records an event for the Contact with the given email through the
// public API. Ingest is accept-then-process: observe the effect with WaitForEmail.
func (w *Workspace) RecordEvent(email, action string) {
	w.t.Helper()
	res, err := w.api.EventsCreate(w.t.Context(), &externalapi.RecordEventsInput{
		Events: []externalapi.EventInput{{
			SubjectId: "subject-" + email,
			Action:    action,
			Email:     externalapi.NewOptNilEmailAddress(externalapi.EmailAddress(email)),
		}},
	})
	require.NoError(w.t, err, "record event")
	_, isNoContent := res.(*externalapi.EventsCreateNoContent)
	require.True(w.t, isNoContent, "record event: unexpected response %T: %+v", res, res)
}

// ExpectNoEmail asserts that nothing addressed to recipient is in the inbox after the
// grace period. It proves absence only when called after a positive wait showed the
// pipeline had caught up, so the grace period is short.
func (w *Workspace) ExpectNoEmail(recipient string, grace time.Duration) {
	w.t.Helper()
	select {
	case <-time.After(grace):
	case <-w.t.Context().Done():
		w.t.Fatal("cancelled while waiting to assert absence")
	}
	found, err := w.env.Mailpit.Search(w.t.Context(), `to:"`+recipient+`"`)
	require.NoError(w.t, err)
	for _, s := range found {
		w.msgIDs = append(w.msgIDs, s.ID)
		for _, a := range s.To {
			assert.False(w.t, strings.EqualFold(a.Address, recipient), "unexpected email to %s: %q", recipient, s.Subject)
		}
	}
}

func TestRecordedEventTriggersAutomationEmail(t *testing.T) {
	t.Parallel()
	w := env.NewWorkspace(t).Ready()
	matching, other := w.NewRecipient(), w.NewRecipient()
	w.ImportContacts(matching, other)
	w.ActivateAutomation(Automation{
		TriggerEvent: "e2e.signed_up",
		Subject:      "Welcome from the automation",
		Body:         `<mjml><mj-body><mj-section><mj-column><mj-text>Glad you signed up</mj-text></mj-column></mj-section></mj-body></mjml>`,
	})

	// A non-matching event first, for a Contact the matching event never touches.
	w.RecordEvent(other, "e2e.something_else")
	w.RecordEvent(matching, "e2e.signed_up")

	msg := w.WaitForEmail(matching)
	assert.Equal(t, w.FromEmail, msg.From.Address)
	assert.Equal(t, "Welcome from the automation", msg.Subject)
	assert.Contains(t, msg.HTML, "Glad you signed up")

	// The matching event's email has arrived, so the earlier non-matching event has
	// long been through the same pipeline: its Contact must have received nothing.
	w.ExpectNoEmail(other, 3*time.Second)
}
