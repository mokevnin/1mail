package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/automation"
	"github.com/mokevnin/1mail/ent/automationrun"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/emailrender"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/tracking"
)

// step is one node in an automation definition (a linear list for the MVP).
type step struct {
	Type    string `json:"type"` // "email" | "wait"
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body,omitempty"` // MJML
	Seconds int    `json:"seconds,omitempty"`
}

// --- trigger evaluation: enroll contacts into matching automations ---

type EvaluateTriggerArgs struct {
	WorkspaceID int64  `json:"workspace_id"`
	ContactID   int64  `json:"contact_id"`
	Action      string `json:"action"`
}

func (EvaluateTriggerArgs) Kind() string { return "automation_evaluate_trigger" }

type EvaluateTriggerWorker struct {
	river.WorkerDefaults[EvaluateTriggerArgs]
	ent *ent.Client
}

func (w *EvaluateTriggerWorker) Work(ctx context.Context, job *river.Job[EvaluateTriggerArgs]) error {
	runIDs, err := EvaluateTrigger(ctx, w.ent, job.Args.WorkspaceID, job.Args.ContactID, job.Args.Action)
	if err != nil {
		return err
	}
	rc := river.ClientFromContext[pgx.Tx](ctx)
	for _, id := range runIDs {
		if _, err := rc.Insert(ctx, RunStepArgs{RunID: id}, nil); err != nil {
			return err
		}
	}
	return nil
}

// EvaluateTrigger enrolls a contact into every active automation in the workspace
// whose trigger_event matches action, creating one AutomationRun each (enroll-
// once-ever via the unique index). Returns the new run IDs to advance. Pure (no
// queue) so it can be tested directly.
func EvaluateTrigger(ctx context.Context, client *ent.Client, workspaceID, contactID int64, action string) ([]int64, error) {
	autos, err := client.Automation.Query().
		Where(
			automation.WorkspaceID(workspaceID),
			automation.StatusEQ(automation.StatusActive),
			automation.TriggerEvent(action),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var runIDs []int64
	for _, a := range autos {
		// Check-then-insert so the common "already enrolled" path doesn't trip the
		// unique constraint (a violation would poison the surrounding transaction).
		// The unique index stays as a race safety net.
		exists, err := client.AutomationRun.Query().
			Where(automationrun.AutomationID(a.ID), automationrun.ContactID(contactID)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}
		run, err := client.AutomationRun.Create().
			SetAutomationID(a.ID).
			SetContactID(contactID).
			SetWorkspaceID(workspaceID).
			Save(ctx)
		if err != nil {
			continue // lost an enrollment race; skip
		}
		runIDs = append(runIDs, run.ID)
	}
	return runIDs, nil
}

// --- run stepping: walk the contact through the automation's steps ---

type RunStepArgs struct {
	RunID int64 `json:"run_id"`
}

func (RunStepArgs) Kind() string { return "automation_run_step" }

type RunStepWorker struct {
	river.WorkerDefaults[RunStepArgs]
	ent      *ent.Client
	bus      *events.Bus
	resolver SenderResolver
	tracker  *tracking.Tracker
}

func (w *RunStepWorker) Work(ctx context.Context, job *river.Job[RunStepArgs]) error {
	res, err := RunStep(ctx, w.ent, w.bus, w.resolver, w.tracker, job.Args.RunID)
	if err != nil {
		return err
	}
	if res.Done {
		return nil
	}
	opts := &river.InsertOpts{}
	if res.ResumeAt != nil {
		opts.ScheduledAt = *res.ResumeAt
	}
	_, err = river.ClientFromContext[pgx.Tx](ctx).Insert(ctx, RunStepArgs{RunID: job.Args.RunID}, opts)
	return err
}

// StepResult tells the worker how to schedule the next step. Done = the run is
// finished. ResumeAt set = schedule the next step then (a wait); nil = run the
// next step immediately.
type StepResult struct {
	Done     bool
	ResumeAt *time.Time
}

// RunStep executes the run's current step and advances it. Exported and queue-free
// so a test can drive a whole automation by looping until Done. Email sends carry
// an unsubscribe footer scoped to this automation (the tracker may be nil, e.g. in
// tests that don't assert on it); open/click tracking is still broadcast-only.
func RunStep(ctx context.Context, client *ent.Client, bus *events.Bus, resolver SenderResolver, tracker *tracking.Tracker, runID int64) (StepResult, error) {
	run, err := client.AutomationRun.Get(ctx, runID)
	if err != nil {
		return StepResult{}, fmt.Errorf("load run %d: %w", runID, err)
	}
	if run.Status != automationrun.StatusActive {
		return StepResult{Done: true}, nil
	}

	a, err := client.Automation.Get(ctx, run.AutomationID)
	if err != nil {
		return StepResult{}, err
	}
	var steps []step
	if err := json.Unmarshal([]byte(a.Definition), &steps); err != nil {
		_, _ = run.Update().SetStatus(automationrun.StatusFailed).Save(ctx)
		return StepResult{}, fmt.Errorf("automation %d definition: %w", a.ID, err)
	}

	if run.CurrentStep >= len(steps) {
		_, _ = run.Update().SetStatus(automationrun.StatusCompleted).ClearResumeAt().Save(ctx)
		return StepResult{Done: true}, nil
	}

	switch s := steps[run.CurrentStep]; s.Type {
	case "wait":
		resume := time.Now().Add(time.Duration(s.Seconds) * time.Second)
		if _, err := run.Update().SetCurrentStep(run.CurrentStep + 1).SetResumeAt(resume).Save(ctx); err != nil {
			return StepResult{}, err
		}
		return StepResult{ResumeAt: &resume}, nil

	case "email":
		c, err := client.Contact.Get(ctx, run.ContactID)
		if err != nil {
			return StepResult{}, err
		}
		// Email channel: a Contact with no email address can't receive this step.
		// Not an opt-out — the sequence simply completes.
		if c.Email == nil {
			_, _ = run.Update().SetStatus(automationrun.StatusCompleted).ClearResumeAt().Save(ctx)
			return StepResult{Done: true}, nil
		}
		// Send-eligibility (ADR 0001): each Automation is its own unsubscribe
		// source, and Suppression is the global hard floor. An ineligible
		// destination (suppressed, or unsubscribed from this automation / from
		// everything) exits the enrollment — a run never silently keeps walking
		// steps while skipping every email.
		requireConfirmed, err := eligibility.RequiresConfirmation(ctx, client, run.WorkspaceID)
		if err != nil {
			return StepResult{}, fmt.Errorf("eligibility: %w", err)
		}
		decision, err := eligibility.Check(ctx, client, run.WorkspaceID,
			eligibility.ChannelEmail, *c.Email, eligibility.AutomationSource(a.ID), true, requireConfirmed)
		if err != nil {
			return StepResult{}, fmt.Errorf("eligibility: %w", err)
		}
		if !decision.Eligible {
			_, _ = run.Update().SetStatus(automationrun.StatusExited).ClearResumeAt().Save(ctx)
			return StepResult{Done: true}, nil
		}
		sender, err := resolver.EmailSender(ctx, run.WorkspaceID)
		if err != nil {
			_, _ = run.Update().SetStatus(automationrun.StatusFailed).Save(ctx)
			return StepResult{}, fmt.Errorf("resolve sender: %w", err)
		}
		email, err := emailrender.RenderEmail(s.Subject, s.Body, contactBindings(c))
		if err != nil {
			_, _ = run.Update().SetStatus(automationrun.StatusFailed).Save(ctx)
			return StepResult{}, fmt.Errorf("render: %w", err)
		}
		// Unsubscribe footer scoped to this automation: each Automation is its own
		// sending source, and the click exits the enrollment (ADR 0001).
		html := email.HTML
		var listUnsubURL string
		if tracker != nil {
			unsub := tracking.UnsubTarget{
				Source:      eligibility.AutomationSource(a.ID),
				Destination: *c.Email,
				WorkspaceID: run.WorkspaceID,
				ContactID:   c.ID,
			}
			// Workspace carries the CAN-SPAM postal address rendered in the footer.
			ws, werr := client.Workspace.Get(ctx, run.WorkspaceID)
			if werr != nil {
				return StepResult{}, fmt.Errorf("load workspace %d: %w", run.WorkspaceID, werr)
			}
			footer, ferr := tracker.UnsubscribeFooter(unsub, ws.PostalAddress)
			if ferr != nil {
				_, _ = run.Update().SetStatus(automationrun.StatusFailed).Save(ctx)
				return StepResult{}, fmt.Errorf("unsubscribe footer: %w", ferr)
			}
			html += footer
			// RFC 8058 one-click header reuses the footer's source-scoped token (ADR 0012).
			url, uerr := tracker.UnsubscribeURL(unsub)
			if uerr != nil {
				_, _ = run.Update().SetStatus(automationrun.StatusFailed).Save(ctx)
				return StepResult{}, fmt.Errorf("unsubscribe url: %w", uerr)
			}
			listUnsubURL = url
		}
		// From/FromName left empty: the provider falls back to the integration's
		// configured sender (messaging.FirstNonEmpty in the smtp/ses senders).
		if err := sender.Send(ctx, messaging.EmailMessage{
			To:                 *c.Email,
			Subject:            email.Subject,
			HTML:               html,
			Text:               email.Text,
			ListUnsubscribeURL: listUnsubURL,
		}); err != nil {
			_, _ = run.Update().SetStatus(automationrun.StatusFailed).Save(ctx)
			return StepResult{}, fmt.Errorf("send: %w", err)
		}
		// Advance the enrollment + publish email.sent atomically (transactional
		// outbox), so the send fact lands in the Event log alongside broadcast sends
		// (Events are the source of truth). The deterministic DedupID (run + step)
		// makes persist idempotent if the step is retried after a send.
		sentStep := run.CurrentStep
		if err := bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
			if _, err := tx.AutomationRun.UpdateOneID(run.ID).
				SetCurrentStep(sentStep + 1).ClearResumeAt().Save(ctx); err != nil {
				return err
			}
			return pub.Publish(ctx, &events.EmailEngagement{
				Action:      events.NameEmailSent,
				WorkspaceID: run.WorkspaceID,
				ContactID:   run.ContactID,
				Email:       *c.Email,
				DedupID:     fmt.Sprintf("email.sent:automation:%d:%d", run.ID, sentStep),
			})
		}); err != nil {
			return StepResult{}, err
		}
		return StepResult{}, nil // continue immediately

	default:
		_, _ = run.Update().SetStatus(automationrun.StatusFailed).Save(ctx)
		return StepResult{}, fmt.Errorf("unknown step type %q", s.Type)
	}
}
