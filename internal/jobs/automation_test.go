package jobs_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent/automation"
	"github.com/mokevnin/1mail/ent/automationrun"
	"github.com/mokevnin/1mail/internal/jobs"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const emailStep = `{"type":"email","subject":"Hi {{ first_name }}","body":"<mjml><mj-body><mj-section><mj-column><mj-text>Hi {{ first_name }}</mj-text></mj-column></mj-section></mj-body></mjml>"}`

// drive runs RunStep until the run finishes (or a safety cap), returning how many
// steps executed.
func drive(t *testing.T, env *testhelper.TestEnv, resolver jobs.SenderResolver, runID int64) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		res, err := jobs.RunStep(ctx, env.DB, resolver, runID)
		require.NoError(t, err)
		if res.Done {
			return
		}
	}
	t.Fatal("run did not finish within cap")
}

func TestAutomationEnrollAndRun(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	c, err := env.DB.Contact.Create().SetWorkspaceID(acmeWorkspaceID).
		SetEmail("auto@test.dev").SetFirstName("Ada").Save(ctx)
	require.NoError(t, err)

	_, err = env.DB.Automation.Create().SetWorkspaceID(acmeWorkspaceID).
		SetName("Welcome").SetTriggerEvent("contact.created").SetStatus(automation.StatusActive).
		SetDefinition("[" + emailStep + `,{"type":"wait","seconds":3600},` + emailStep + "]").
		Save(ctx)
	require.NoError(t, err)

	runIDs, err := jobs.EvaluateTrigger(ctx, env.DB, acmeWorkspaceID, c.ID, "contact.created")
	require.NoError(t, err)
	require.Len(t, runIDs, 1)

	// Enroll-once: a second matching event creates no new run.
	again, err := jobs.EvaluateTrigger(ctx, env.DB, acmeWorkspaceID, c.ID, "contact.created")
	require.NoError(t, err)
	assert.Empty(t, again)

	fs := &fakeSender{}
	drive(t, env, fakeResolver{sender: fs}, runIDs[0])

	// Two email steps fired (the wait step does not send).
	assert.Len(t, fs.sent, 2)
	assert.Equal(t, "Hi Ada", fs.sent[0].Subject)
	assert.NotContains(t, fs.sent[0].HTML, "{{")

	got := env.DB.AutomationRun.GetX(ctx, runIDs[0])
	assert.Equal(t, automationrun.StatusCompleted, got.Status)
}

func TestAutomationInactiveDoesNotEnroll(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	c, err := env.DB.Contact.Create().SetWorkspaceID(acmeWorkspaceID).SetEmail("x@test.dev").Save(ctx)
	require.NoError(t, err)
	_, err = env.DB.Automation.Create().SetWorkspaceID(acmeWorkspaceID).
		SetName("Draft").SetTriggerEvent("contact.created").Save(ctx) // status defaults to draft
	require.NoError(t, err)

	runIDs, err := jobs.EvaluateTrigger(ctx, env.DB, acmeWorkspaceID, c.ID, "contact.created")
	require.NoError(t, err)
	assert.Empty(t, runIDs)
}

func TestAutomationWaitDefersNextStep(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	c, err := env.DB.Contact.Create().SetWorkspaceID(acmeWorkspaceID).SetEmail("w@test.dev").Save(ctx)
	require.NoError(t, err)
	a, err := env.DB.Automation.Create().SetWorkspaceID(acmeWorkspaceID).
		SetName("Delayed").SetTriggerEvent("contact.created").SetStatus(automation.StatusActive).
		SetDefinition(`[{"type":"wait","seconds":3600},` + emailStep + `]`).Save(ctx)
	require.NoError(t, err)
	_ = a

	runIDs, err := jobs.EvaluateTrigger(ctx, env.DB, acmeWorkspaceID, c.ID, "contact.created")
	require.NoError(t, err)
	require.Len(t, runIDs, 1)

	// First step is a wait: it defers (ResumeAt set), no send yet.
	res, err := jobs.RunStep(ctx, env.DB, fakeResolver{sender: &fakeSender{}}, runIDs[0])
	require.NoError(t, err)
	assert.False(t, res.Done)
	require.NotNil(t, res.ResumeAt)
	assert.NotNil(t, env.DB.AutomationRun.GetX(ctx, runIDs[0]).ResumeAt)
}
