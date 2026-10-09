package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/automation"
	"github.com/mokevnin/1mail/ent/automationrun"
	"github.com/mokevnin/1mail/ent/outboundmessage"
	"github.com/mokevnin/1mail/internal/automations"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/outbound"
	"github.com/mokevnin/1mail/internal/tags"
)

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
	ent *ent.Client
	mod *outbound.Module
}

func (w *RunStepWorker) Work(ctx context.Context, job *river.Job[RunStepArgs]) error {
	res, err := RunStep(ctx, w.ent, w.mod, job.Args.RunID)
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
// so a test can drive a whole automation by looping until Done. An email step is
// one Outbound send under this automation's Sending source (ADR 0015): the module
// owns eligibility, the unsubscribe footer and headers, and the send record, and
// RunStep only turns its Outcome into the enrollment's next state.
func RunStep(ctx context.Context, client *ent.Client, mod *outbound.Module, runID int64) (StepResult, error) {
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
	steps, err := automations.Decode(a.Definition)
	if err != nil {
		_, _ = run.Update().SetStatus(automationrun.StatusFailed).Save(ctx)
		return StepResult{}, fmt.Errorf("automation %d definition: %w", a.ID, err)
	}

	if run.CurrentStep >= len(steps) {
		_, _ = run.Update().SetStatus(automationrun.StatusCompleted).ClearResumeAt().Save(ctx)
		return StepResult{Done: true}, nil
	}

	switch s := steps[run.CurrentStep]; s.Type {
	case automations.StepWait:
		resume := time.Now().Add(time.Duration(s.Seconds) * time.Second)
		if _, err := run.Update().SetCurrentStep(run.CurrentStep + 1).SetResumeAt(resume).Save(ctx); err != nil {
			return StepResult{}, err
		}
		return StepResult{ResumeAt: &resume}, nil

	case automations.StepApplyTag, automations.StepRemoveTag:
		// A tag step changes the Contact's Tags and moves straight on; it sends nothing.
		var err error
		if s.Type == automations.StepApplyTag {
			_, err = tags.New(client).Apply(ctx, run.WorkspaceID, run.ContactID, s.Tag)
		} else {
			err = tags.New(client).Remove(ctx, run.WorkspaceID, run.ContactID, s.Tag)
		}
		if err != nil {
			return StepResult{}, err
		}
		if _, err := run.Update().SetCurrentStep(run.CurrentStep + 1).ClearResumeAt().Save(ctx); err != nil {
			return StepResult{}, err
		}
		return StepResult{}, nil // continue immediately

	case automations.StepEmail:
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
		step := run.CurrentStep
		res, err := mod.Send(ctx, outbound.Request{
			WorkspaceID: run.WorkspaceID,
			Kind:        outboundmessage.KindAutomation,
			Key:         fmt.Sprintf("automation:%d:%d", run.ID, step),
			Destination: *c.Email,
			Contact:     c,
			// Each Automation is its own unsubscribe Sending source; the unsubscribe
			// click exits this enrollment (ADR 0001).
			Source:  eligibility.AutomationSource(a.ID),
			Subject: s.Subject,
			Body:    s.Body,
			Ref:     outbound.Ref{AutomationID: a.ID, AutomationRunID: run.ID, AutomationStep: &step},
		})
		if err != nil {
			return StepResult{}, err // retryable: the worker's retry replays the same key
		}
		switch res.Outcome {
		case outbound.Sent:
			// Advance past the email step. A crash between the send and this write
			// replays the recorded Sent on retry, so the step never sends twice.
			if _, err := run.Update().SetCurrentStep(step + 1).ClearResumeAt().Save(ctx); err != nil {
				return StepResult{}, err
			}
			return StepResult{}, nil // continue immediately
		case outbound.Skipped:
			// An ineligible destination (suppressed, or unsubscribed from this
			// automation / from everything) exits the enrollment — a run never
			// silently keeps walking steps while skipping every email.
			_, _ = run.Update().SetStatus(automationrun.StatusExited).ClearResumeAt().Save(ctx)
			return StepResult{Done: true}, nil
		case outbound.Failed:
			_, _ = run.Update().SetStatus(automationrun.StatusFailed).Save(ctx)
			return StepResult{}, fmt.Errorf("email step %d: %s", step, res.Reason)
		default: // outbound.Held: the enrollment waits, unchanged, and asks again later
			resume := time.Now().Add(holdRetryDelay)
			return StepResult{ResumeAt: &resume}, nil
		}

	default:
		_, _ = run.Update().SetStatus(automationrun.StatusFailed).Save(ctx)
		return StepResult{}, fmt.Errorf("unknown step type %q", s.Type)
	}
}
