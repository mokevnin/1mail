package jobs

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/internal/i18n"
	"github.com/mokevnin/1mail/internal/messaging"
)

// Auth-mail flows: the self-service account emails sent through the system
// (platform) sender. Each maps to a public SPA page that posts the token back.
const (
	flowPasswordReset = "password_reset"
	flowEmailVerify   = "email_verify"
	flowEmailChange   = "email_change"
)

// SendAuthMailArgs is the river payload for a self-service account email. Flow
// selects the subject line and the SPA path the link points at.
type SendAuthMailArgs struct {
	Flow  string `json:"flow"`
	Email string `json:"email"`
	Token string `json:"token"`
}

func (SendAuthMailArgs) Kind() string { return "send_auth_mail" }

// SendAuthMailWorker sends account emails via the system (platform) sender.
type SendAuthMailWorker struct {
	river.WorkerDefaults[SendAuthMailArgs]
	sender messaging.EmailSender
	appURL string
}

func (w *SendAuthMailWorker) Work(ctx context.Context, job *river.Job[SendAuthMailArgs]) error {
	return SendAuthMail(ctx, w.sender, w.appURL, job.Args)
}

// SendAuthMail renders and sends a self-service account email through the system
// sender (1mail's own provider — NOT a workspace integration, so it bypasses the
// workspace suppression/eligibility machinery). Pure (no queue) so it runs
// identically under the river worker and the inline adapter.
func SendAuthMail(ctx context.Context, sender messaging.EmailSender, appURL string, args SendAuthMailArgs) error {
	if sender == nil {
		return fmt.Errorf("send auth mail: no system email sender configured")
	}
	subjectID, path, introID := authMailCopy(args.Flow)
	if path == "" {
		return fmt.Errorf("send auth mail: unknown flow %q", args.Flow)
	}
	link := strings.TrimRight(appURL, "/") + path + "?token=" + url.QueryEscape(args.Token)
	body := fmt.Sprintf("%s\n\n%s\n\n%s\n", i18n.T(introID, nil), link, i18n.T("email.auth.footer", nil))
	return sender.Send(ctx, messaging.EmailMessage{
		To:      args.Email,
		Subject: i18n.T(subjectID, nil),
		Text:    body,
	})
}

// authMailCopy returns the subject message id, SPA path, and intro message id
// for a flow (empty path signals an unknown flow). The copy itself is localized
// by internal/i18n at send time.
func authMailCopy(flow string) (subjectID, path, introID string) {
	switch flow {
	case flowPasswordReset:
		return "email.password_reset.subject", "/reset-password", "email.password_reset.intro"
	case flowEmailVerify:
		return "email.email_verify.subject", "/verify-email", "email.email_verify.intro"
	case flowEmailChange:
		return "email.email_change.subject", "/confirm-email-change", "email.email_change.intro"
	default:
		return "", "", ""
	}
}

// EnqueuePasswordReset schedules the password-reset email (river adapter).
func (c *Client) EnqueuePasswordReset(ctx context.Context, email, token string) error {
	return c.enqueueAuthMail(ctx, flowPasswordReset, email, token)
}

// EnqueueEmailVerification schedules the signup email-verification email.
func (c *Client) EnqueueEmailVerification(ctx context.Context, email, token string) error {
	return c.enqueueAuthMail(ctx, flowEmailVerify, email, token)
}

// EnqueueEmailChangeConfirm schedules the confirm-new-email email (sent to the
// requested new address).
func (c *Client) EnqueueEmailChangeConfirm(ctx context.Context, email, token string) error {
	return c.enqueueAuthMail(ctx, flowEmailChange, email, token)
}

func (c *Client) enqueueAuthMail(ctx context.Context, flow, email, token string) error {
	_, err := c.river.Insert(ctx, SendAuthMailArgs{Flow: flow, Email: email, Token: token}, nil)
	return err
}
