//go:build e2e

package e2e

import (
	"testing"

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
// public API. Ingest is accept-then-process: observe the effect with Inbox.Wait.
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

func TestRecordedEventTriggersAutomationEmail(t *testing.T) {
	t.Parallel()
	w := env.NewWorkspace(t).Ready()
	matching, other := w.NewRecipient(), w.NewRecipient()
	w.ImportContacts(matching, other)
	w.ActivateAutomation(Automation{
		TriggerEvent: "e2e.signed_up",
		Subject:      "Welcome from the automation",
		Body:         mjml("Glad you signed up"),
	})

	// A non-matching event first, for a Contact the matching event never touches.
	w.RecordEvent(other, "e2e.something_else")
	w.RecordEvent(matching, "e2e.signed_up")

	msg := w.Inbox.Wait(Match{To: matching})
	assert.Equal(t, w.FromEmail, msg.From.Address)
	assert.Equal(t, "Welcome from the automation", msg.Subject)
	assert.Contains(t, msg.HTML, "Glad you signed up")

	// The matching event's email has arrived, so the earlier non-matching event has
	// long been through the same pipeline: its Contact must have received nothing.
	w.Inbox.RequireNone(Match{To: other})
}
