package jobs

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/internal/messaging"
)

// SendWelcomeArgs is the river job payload for the platform welcome email.
type SendWelcomeArgs struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (SendWelcomeArgs) Kind() string { return "send_welcome" }

// SendWelcomeWorker sends the welcome email via the system (platform) sender.
type SendWelcomeWorker struct {
	river.WorkerDefaults[SendWelcomeArgs]
	sender messaging.EmailSender
}

func (w *SendWelcomeWorker) Work(ctx context.Context, job *river.Job[SendWelcomeArgs]) error {
	return SendWelcome(ctx, w.sender, job.Args.Email, job.Args.Name)
}

// SendWelcome sends the platform welcome email through the system sender (1mail's
// own provider — NOT a workspace integration). Pure (no queue) so it runs
// identically under the river worker and the inline adapter.
func SendWelcome(ctx context.Context, sender messaging.EmailSender, email, name string) error {
	if sender == nil {
		return fmt.Errorf("send welcome: no system email sender configured")
	}
	greeting := name
	if greeting == "" {
		greeting = "there"
	}
	body := fmt.Sprintf("Hi %s,\n\nWelcome to 1mail! Your account is ready.\n", greeting)
	return sender.Send(ctx, messaging.EmailMessage{
		To:      email,
		Subject: "Welcome to 1mail",
		Text:    body,
	})
}

// EnqueueWelcome schedules the welcome email (river adapter).
func (c *Client) EnqueueWelcome(ctx context.Context, email, name string) error {
	_, err := c.river.Insert(ctx, SendWelcomeArgs{Email: email, Name: name}, nil)
	return err
}
