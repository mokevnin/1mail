package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/sending"
	"github.com/mokevnin/1mail/internal/tracking"
)

// Inline is the synchronous adapter of the enqueue seam (the Rails `:inline`
// equivalent): instead of inserting into river, it runs each job's pure function
// immediately. It satisfies the same narrow enqueue interfaces the river Client
// does (BroadcastEnqueuer, WelcomeEnqueuer), so the test harness (and a future
// single-process mode) exercise the enqueue→execute path for real rather than
// asserting "a job was enqueued".
type Inline struct {
	ent          *ent.Client
	bus          *events.Bus
	resolver     SenderResolver
	tracker      *tracking.Tracker
	systemSender messaging.EmailSender
	lookup       sending.TXTLookup
	appURL       string
}

// NewInline builds the inline adapter. systemSender sends platform mail; resolver
// resolves a workspace's customer sender for broadcasts; bus carries the send's
// transactional outbox publish (email.sent). lookup resolves DKIM TXT for sending-
// domain verification (a stub in tests avoids real DNS). appURL builds account-
// email links.
func NewInline(entClient *ent.Client, bus *events.Bus, resolver SenderResolver, tracker *tracking.Tracker, systemSender messaging.EmailSender, lookup sending.TXTLookup, appURL string) *Inline {
	return &Inline{ent: entClient, bus: bus, resolver: resolver, tracker: tracker, systemSender: systemSender, lookup: lookup, appURL: appURL}
}

// EnqueueBroadcast runs the broadcast send now. A future scheduledAt is skipped:
// the inline adapter has no scheduler, and running a future job now would be
// wrong (e.g. it must not fire a broadcast scheduled for tomorrow).
func (i *Inline) EnqueueBroadcast(ctx context.Context, broadcastID int64, scheduledAt *time.Time) error {
	if scheduledAt != nil && scheduledAt.After(time.Now()) {
		return nil
	}
	return SendBroadcast(ctx, i.ent, i.bus, i.resolver, i.tracker, broadcastID)
}

// EnqueueWelcome sends the welcome email now via the system sender.
func (i *Inline) EnqueueWelcome(ctx context.Context, email, name string) error {
	return SendWelcome(ctx, i.systemSender, email, name)
}

// EnqueuePasswordReset sends the password-reset email now.
func (i *Inline) EnqueuePasswordReset(ctx context.Context, email, token string) error {
	return SendAuthMail(ctx, i.systemSender, i.appURL, SendAuthMailArgs{Flow: flowPasswordReset, Email: email, Token: token})
}

// EnqueueEmailVerification sends the signup email-verification email now.
func (i *Inline) EnqueueEmailVerification(ctx context.Context, email, token string) error {
	return SendAuthMail(ctx, i.systemSender, i.appURL, SendAuthMailArgs{Flow: flowEmailVerify, Email: email, Token: token})
}

// EnqueueEmailChangeConfirm sends the confirm-new-email email now.
func (i *Inline) EnqueueEmailChangeConfirm(ctx context.Context, email, token string) error {
	return SendAuthMail(ctx, i.systemSender, i.appURL, SendAuthMailArgs{Flow: flowEmailChange, Email: email, Token: token})
}

// EnqueueSendingDomainVerify re-checks the domain's DKIM DNS now, notifying the
// workspace owner if this check flips the domain to unverified (ADR 0010 slice 3).
func (i *Inline) EnqueueSendingDomainVerify(ctx context.Context, sendingDomainID int64) error {
	_, flipped, err := VerifySendingDomainByID(ctx, i.ent, i.lookup, sendingDomainID)
	if err != nil {
		return err
	}
	if flipped {
		if nerr := NotifySendingDomainUnverified(ctx, i.ent, i.systemSender, sendingDomainID); nerr != nil {
			slog.WarnContext(ctx, "notify sending domain unverified failed",
				"sending_domain_id", sendingDomainID, "err", nerr)
		}
	}
	return nil
}

// EnqueueMemberInvite sends the workspace invite email now via the system sender.
func (i *Inline) EnqueueMemberInvite(ctx context.Context, email, inviteURL, workspaceName, inviterName string) error {
	return SendMemberInvite(ctx, i.systemSender, SendMemberInviteArgs{
		Email:         email,
		InviteURL:     inviteURL,
		WorkspaceName: workspaceName,
		InviterName:   inviterName,
	})
}
