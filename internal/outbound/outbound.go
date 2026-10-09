// Package outbound is the one place an email leaves 1mail on a Workspace's behalf
// (ADR 0015). Every surface — Broadcast, Automation step, Transactional — hands it
// a Request and gets back a typed Outcome; none re-implements any part of what lies
// between "this message is wanted" and "the provider accepted it":
//
//   - replay: an already-recorded send returns its recorded Outcome (idempotency);
//   - holds: Workspace freeze, no Integration, unverified Sending domain — reversible,
//     per-source, never consumed as a per-recipient failure;
//   - claim: the Outbound message row is taken before the provider is called, under
//     the Request's idempotency key, with a lease against racing retries;
//   - Send-eligibility, per message, fail-closed (a Skipped is final for that
//     destination);
//   - render, the unsubscribe footer and RFC 8058 headers (marketing only), DKIM
//     signing (inside the provider adapter), the provider call;
//   - the record of the result and the email.sent Event, in one transaction.
//
// The provider is the seam: messaging.EmailSender has an SMTP and an SES adapter in
// production and an in-memory one in tests.
package outbound

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/outboundmessage"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/emailrender"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/tracking"
)

// Outcome is what an Outbound send did. Sent, Skipped and Failed are final for the
// destination; Held is a reversible hold on the source — nothing was consumed, the
// caller should try the same Request again later.
type Outcome string

const (
	// Sent: the provider accepted the message (not "delivered" — see GLOSSARY).
	Sent Outcome = "sent"
	// Skipped: Send-eligibility forbids this destination. Reason is an eligibility.Reason*.
	Skipped Outcome = "skipped"
	// Failed: this one message can never be sent (e.g. its template failed to render).
	Failed Outcome = "failed"
	// Held: the source cannot send right now. Reason is a Hold* constant.
	Held Outcome = "held"
)

// Reasons an Outbound send is Held.
const (
	HoldSuspended        = "workspace_suspended"
	HoldNoIntegration    = "no_integration"
	HoldUnverifiedDomain = "unverified_domain"
)

// HoldDetail words a Hold reason for a human (an API problem detail). The single
// place the wording lives, so every surface that reports a Held outcome agrees.
func HoldDetail(reason string) string {
	switch reason {
	case HoldSuspended:
		return "sending is suspended for this workspace"
	case HoldNoIntegration:
		return "no default email provider configured"
	case HoldUnverifiedDomain:
		return "sender domain is not a verified sending domain"
	default:
		return "sending is currently on hold for this workspace: " + reason
	}
}

// ErrInProgress means another attempt holds a live claim on this idempotency key.
// It is retryable; a stale claim (older than the lease) is taken over instead.
var ErrInProgress = errors.New("outbound: a send with this key is already in progress")

// ErrNoTracker means a marketing send (a Request with a Source) was attempted
// without the Tracker that builds its unsubscribe link. A marketing message must
// never leave without one (ADR 0012), so this is a configuration error, never a
// silent downgrade.
var ErrNoTracker = errors.New("outbound: marketing send requires an unsubscribe tracker")

// Result is the outcome of a Send.
type Result struct {
	Outcome Outcome
	// Reason explains Skipped (eligibility reason), Failed (cause) and Held (hold).
	Reason string
	// MessageID is the Outbound message row; zero when Held (nothing was recorded).
	MessageID int64
	// Replayed is true when the Outcome was read back from an earlier attempt.
	Replayed bool
}

// Senders resolves a Workspace's email provider adapter (its default Integration).
// *messaging.Resolver satisfies it.
type Senders interface {
	EmailSender(ctx context.Context, workspaceID int64) (messaging.EmailSender, error)
}

// Freezer is an extra reason a Workspace may not send right now, consulted after the
// core suspension check (ADR 0007, 0009): the EE Billing hold plugs in here. It
// returns "" when the Workspace may send, otherwise a Hold* style reason.
type Freezer interface {
	Frozen(ctx context.Context, ws *ent.Workspace) (reason string, err error)
}

// DefaultLease is how long a pending claim blocks other attempts before a retry may
// take it over.
const DefaultLease = 5 * time.Minute

// Module is the Outbound send module.
type Module struct {
	ent      *ent.Client
	bus      *events.Bus
	senders  Senders
	tracker  *tracking.Tracker
	lease    time.Duration
	freezers []Freezer
}

// Option customizes a Module.
type Option func(*Module)

// WithLease overrides DefaultLease (tests use a tiny lease to exercise takeover).
func WithLease(d time.Duration) Option { return func(m *Module) { m.lease = d } }

// WithFreezers adds extra freeze reasons after the core suspension check.
func WithFreezers(f ...Freezer) Option {
	return func(m *Module) { m.freezers = append(m.freezers, f...) }
}

// New builds the module. tracker may be nil only if no marketing Request is sent.
func New(client *ent.Client, bus *events.Bus, senders Senders, tracker *tracking.Tracker, opts ...Option) *Module {
	m := &Module{ent: client, bus: bus, senders: senders, tracker: tracker, lease: DefaultLease}
	for _, o := range opts {
		o(m)
	}
	return m
}

// Ref ties a Request to the surface's own records. They are stored on the Outbound
// message as plain ids and stamped on the email.sent Event.
type Ref struct {
	BroadcastID        int64
	BroadcastRecipient int64
	AutomationID       int64
	AutomationRunID    int64
	AutomationStep     *int
	TemplateID         int64
}

// Request describes one email to send. Content is the already-bound Message
// content (a marketing surface passes its own copy, Transactional the live
// Template's current content — see ADR 0003, 0005); the module never looks it up.
type Request struct {
	Kind outboundmessage.Kind
	// Key is unique per logical send: a retry of the same send repeats it. Required.
	Key string

	// Destination is the email address. Contact, when the destination resolves to
	// one, supplies merge tags and attribution; ContactID alone attributes only.
	Destination string
	Contact     *ent.Contact
	ContactID   int64

	// Source is the unsubscribe Sending source ("broadcasts", "automation:<id>");
	// "" is a Transactional send (Suppression only, no unsubscribe).
	Source string

	Subject string
	Body    string // MJML
	// Variables are extra merge tags; they win over Contact fields.
	Variables map[string]any

	// FromEmail/FromName override the Integration's configured sender.
	FromEmail string
	FromName  string

	// TrackID, when non-zero, turns on open/click tracking for this message
	// (the Broadcast recipient id the tracking token encodes).
	TrackID int64

	Ref Ref
}

func (r Request) validate() error {
	switch {
	case r.Key == "":
		return errors.New("outbound: idempotency key required")
	case eligibility.NormalizeDestination(r.Destination) == "":
		return errors.New("outbound: destination required")
	}
	return nil
}

// Send performs one Outbound send. A returned error is a retryable infrastructure
// failure (provider unreachable, database error, ErrInProgress); every decided
// outcome — including Skipped, Failed and Held — is a Result with a nil error.
func (m *Module) Send(ctx context.Context, s *ent.Scoped, req Request) (Result, error) {
	if err := req.validate(); err != nil {
		return Result{}, err
	}
	if req.Source != "" && m.tracker == nil {
		return Result{}, ErrNoTracker
	}
	dest := eligibility.NormalizeDestination(req.Destination)

	// A recorded send replays its Outcome — even while the Workspace is frozen: the
	// message already left (or was already decided), nothing new is being sent.
	existing, err := m.find(ctx, s, req.Key)
	if err != nil {
		return Result{}, err
	}
	if existing != nil && existing.Status != outboundmessage.StatusPending {
		return replayResult(existing), nil
	}

	ws, err := m.workspace(ctx, s)
	if err != nil {
		return Result{}, err
	}
	g, err := m.gate(ctx, ws, req.FromEmail, req.FromName)
	if err != nil {
		return Result{}, err
	}
	if g.hold != "" {
		return Result{Outcome: Held, Reason: g.hold}, nil
	}

	msg, replay, err := m.claim(ctx, s, req, dest, g, existing)
	if err != nil {
		return Result{}, err
	}
	if replay != nil {
		return *replay, nil
	}

	// Send-eligibility, per message at send time, fail-closed: if the check cannot
	// be made the message is not sent and the attempt is retried.
	dec, err := eligibility.Check(ctx, m.ent, s.WorkspaceID(), eligibility.ChannelEmail, dest, req.Source)
	if err != nil {
		m.release(ctx, s, msg)
		return Result{}, err
	}
	if !dec.Eligible {
		return m.finish(ctx, s, msg, outboundmessage.StatusSkipped, Skipped, dec.Reason)
	}

	built, err := m.compose(req, dest, ws)
	if err != nil {
		return m.finish(ctx, s, msg, outboundmessage.StatusFailed, Failed, err.Error())
	}
	built.msg.From, built.msg.FromName = g.from, g.fromName
	built.msg.To = dest

	receipt, err := g.sender.Send(ctx, built.msg)
	if err != nil {
		if errors.Is(err, messaging.ErrUnverifiedSendingDomain) {
			// The domain lost verification between our check and the signer's: a hold,
			// not a failure. The claim is dropped so the same Request can run again.
			_, _ = s.OutboundMessage().Delete().Where(holds(msg)...).Exec(ctx)
			return Result{Outcome: Held, Reason: HoldUnverifiedDomain}, nil
		}
		m.release(ctx, s, msg)
		return Result{}, fmt.Errorf("outbound: send to %s: %w", dest, err)
	}

	return m.recordSent(ctx, s, req, msg, g, dest, receipt)
}

// Preflight reports whether a Workspace could send right now from fromEmail (""
// uses the Integration's configured sender), without sending or recording anything.
// Planning code (Broadcast) calls it to fail or pause fast instead of finding out
// per recipient; the answer is the same gate Send applies.
func (m *Module) Preflight(ctx context.Context, s *ent.Scoped, fromEmail string) (hold string, err error) {
	ws, err := m.workspace(ctx, s)
	if err != nil {
		return "", err
	}
	g, err := m.gate(ctx, ws, fromEmail, "")
	if err != nil {
		return "", err
	}
	return g.hold, nil
}

// MarkFailed gives up on a pending claim for good (a queue worker calls it when the
// final retry is spent), recording cause. It is a no-op for a message that already
// reached a final status or whose claim another attempt currently holds.
func (m *Module) MarkFailed(ctx context.Context, s *ent.Scoped, key string, cause error) error {
	return s.OutboundMessage().Update().
		Where(
			outboundmessage.IdempotencyKey(key),
			outboundmessage.StatusEQ(outboundmessage.StatusPending),
			// Never fail a message another attempt is working on: only a claim that
			// was released (a definite provider failure) or whose lease has expired.
			outboundmessage.ClaimedAtLT(time.Now().Add(-m.lease)),
		).
		SetStatus(outboundmessage.StatusFailed).
		SetReason(cause.Error()).
		Exec(ctx)
}

// workspace loads the Workspace row the scoped client is confined to. Workspace is
// the tenant root, not a Workspace-owned entity, so it is read through the raw client.
func (m *Module) workspace(ctx context.Context, s *ent.Scoped) (*ent.Workspace, error) {
	ws, err := m.ent.Workspace.Get(ctx, s.WorkspaceID())
	if err != nil {
		return nil, fmt.Errorf("outbound: load workspace %d: %w", s.WorkspaceID(), err)
	}
	return ws, nil
}

// gateResult is the outcome of the source-level checks plus what they resolved.
type gateResult struct {
	hold     string
	sender   messaging.EmailSender
	from     string
	fromName string
	domain   string
}

// gate runs the source-level checks, in order: Workspace freeze, an Integration to
// send through, and a verified Sending domain for the effective From address.
func (m *Module) gate(ctx context.Context, ws *ent.Workspace, fromEmail, fromName string) (gateResult, error) {
	if ws.SuspendedAt != nil {
		return gateResult{hold: HoldSuspended}, nil
	}
	for _, f := range m.freezers {
		reason, err := f.Frozen(ctx, ws)
		if err != nil {
			return gateResult{}, fmt.Errorf("outbound: freeze check: %w", err)
		}
		if reason != "" {
			return gateResult{hold: reason}, nil
		}
	}

	sender, err := m.senders.EmailSender(ctx, ws.ID)
	if errors.Is(err, messaging.ErrNoProvider) {
		return gateResult{hold: HoldNoIntegration}, nil
	}
	if err != nil {
		return gateResult{}, fmt.Errorf("outbound: resolve sender: %w", err)
	}

	// The effective From is the Request's, else the Integration's configured one —
	// the address that will really go on the wire, so the domain gate and the
	// Sending-domain stamp agree with the DKIM signer.
	from, name := fromEmail, fromName
	if d, ok := sender.(messaging.DefaultFromer); ok {
		df, dn := d.DefaultFrom()
		from, name = messaging.FirstNonEmpty(from, df), messaging.FirstNonEmpty(name, dn)
	}
	verified, err := messaging.HasVerifiedSendingDomain(ctx, m.ent, ws.ID, from)
	if err != nil {
		return gateResult{}, fmt.Errorf("outbound: check sending domain: %w", err)
	}
	if !verified {
		return gateResult{hold: HoldUnverifiedDomain}, nil
	}
	return gateResult{sender: sender, from: from, fromName: name, domain: messaging.DomainOf(from)}, nil
}

func replayResult(msg *ent.OutboundMessage) Result {
	r := Result{MessageID: msg.ID, Replayed: true}
	r.Reason = lo.FromPtr(msg.Reason)
	switch msg.Status {
	case outboundmessage.StatusSent:
		r.Outcome = Sent
	case outboundmessage.StatusSkipped:
		r.Outcome = Skipped
	default:
		r.Outcome = Failed
	}
	return r
}

// composed is a rendered message ready for the provider (From/To are added by Send).
type composed struct {
	msg messaging.EmailMessage
}

// compose renders the content for one recipient and, for a marketing send, layers
// on the unsubscribe footer, the open/click tracking and the one-click header. Any
// error here is deterministic for this message, so the caller records it as Failed.
func (m *Module) compose(req Request, dest string, ws *ent.Workspace) (composed, error) {
	email, err := emailrender.RenderEmail(req.Subject, req.Body, bindings(req.Contact, req.Variables))
	if err != nil {
		return composed{}, err
	}
	out := composed{msg: messaging.EmailMessage{Subject: email.Subject, HTML: email.HTML, Text: email.Text}}
	if req.Source == "" {
		return out, nil // Transactional: no unsubscribe, no tracking
	}

	contactID := req.ContactID
	if req.Contact != nil {
		contactID = req.Contact.ID
	}
	unsub := tracking.UnsubTarget{
		Source:      req.Source,
		Destination: dest,
		WorkspaceID: ws.ID,
		ContactID:   contactID,
		BroadcastID: req.Ref.BroadcastID,
	}
	if req.TrackID != 0 {
		// Click/open tracking is layered on after rendering so re-parsing the HTML
		// cannot mangle the pixel or links; Rewrite also appends the footer.
		html, err := m.tracker.Rewrite(email.HTML, req.TrackID, unsub, ws.PostalAddress)
		if err != nil {
			return composed{}, fmt.Errorf("tracking rewrite: %w", err)
		}
		out.msg.HTML = html
	} else {
		footer, err := m.tracker.UnsubscribeFooter(unsub, ws.PostalAddress)
		if err != nil {
			return composed{}, fmt.Errorf("unsubscribe footer: %w", err)
		}
		out.msg.HTML = email.HTML + footer
	}
	// RFC 8058 one-click header: the same source-scoped token the footer carries
	// (ADR 0012); the provider adapter covers it with the DKIM signature.
	url, err := m.tracker.UnsubscribeURL(unsub)
	if err != nil {
		return composed{}, fmt.Errorf("unsubscribe url: %w", err)
	}
	out.msg.ListUnsubscribeURL = url
	return out, nil
}

// bindings builds the Liquid merge-tag context: the Contact's custom fields first,
// then its core fields (so a custom field can never shadow email/first_name/
// last_name), then the Request's own variables, which win.
func bindings(c *ent.Contact, vars map[string]any) map[string]any {
	b := map[string]any{}
	if c != nil {
		for k, v := range c.CustomFields {
			b[k] = v
		}
		b["email"], b["first_name"], b["last_name"] = "", "", ""
		if c.Email != nil {
			b["email"] = *c.Email
		}
		if c.FirstName != nil {
			b["first_name"] = *c.FirstName
		}
		if c.LastName != nil {
			b["last_name"] = *c.LastName
		}
	}
	for k, v := range vars {
		b[k] = v
	}
	return b
}

// recordSent marks the claim Sent and publishes email.sent in one transaction, so
// the send fact reaches the Event log iff the message is recorded as sent. The
// provider call precedes this transaction, so delivery to the provider is
// at-least-once; the idempotency key and the Event DedupID make a retry safe.
func (m *Module) recordSent(ctx context.Context, s *ent.Scoped, req Request, msg *ent.OutboundMessage, g gateResult, dest string, receipt messaging.Receipt) (Result, error) {
	contactID := req.ContactID
	if req.Contact != nil {
		contactID = req.Contact.ID
	}
	now := time.Now()
	err := m.bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		// The claim was made through the scoped client; the transaction's write goes
		// through the same Workspace, taken from the claimed row.
		upd := tx.Scoped(msg.WorkspaceID).OutboundMessage().Update().
			Where(holds(msg)...).
			SetStatus(outboundmessage.StatusSent).
			SetSentAt(now)
		if receipt.MessageID != "" {
			upd.SetProviderMessageID(receipt.MessageID)
		}
		n, err := upd.Save(ctx)
		if err != nil {
			return err
		}
		if n == 0 {
			// Our claim was taken over while the provider call ran (we outlived the
			// lease). The taker owns the record; rolling back keeps it, and its
			// email.sent Event, the only ones.
			return ErrInProgress
		}
		return pub.Publish(ctx, &events.EmailEngagement{
			Action:            events.NameEmailSent,
			WorkspaceID:       s.WorkspaceID(),
			ContactID:         contactID,
			Email:             dest,
			BroadcastID:       req.Ref.BroadcastID,
			DedupID:           fmt.Sprintf("email.sent:%d", msg.ID),
			Surface:           string(req.Kind),
			SendingSource:     req.Source,
			SendingDomain:     g.domain,
			AutomationID:      req.Ref.AutomationID,
			OutboundMessageID: msg.ID,
			ProviderMessageID: receipt.MessageID,
		})
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Outcome: Sent, MessageID: msg.ID}, nil
}

// TestRequest is an operator-initiated test send of a message to an address the
// author names explicitly (e.g. a Broadcast preview).
type TestRequest struct {
	To        string
	Subject   string
	Body      string // MJML
	Variables map[string]any
	FromEmail string
	FromName  string
}

// TestSubjectPrefix marks a preview send so it is never mistaken for the real mailing.
const TestSubjectPrefix = "[Test] "

// SendBroadcastTest sends the Broadcast to one address with sample merge data for
// an author to preview it. It returns "" when the message was sent, otherwise the
// human-readable reason it was not (a send error, a Hold, or content that did not
// render), so every surface words a refused preview the same way.
func (m *Module) SendBroadcastTest(ctx context.Context, s *ent.Scoped, b *ent.Broadcast, to string) string {
	res, err := m.SendTest(ctx, s, TestRequest{
		To:        to,
		Subject:   TestSubjectPrefix + b.Subject,
		Body:      b.Body,
		Variables: map[string]any{"first_name": "Alex", "last_name": "Sample", "email": to},
		FromEmail: lo.FromPtr(b.FromEmail),
		FromName:  lo.FromPtr(b.FromName),
	})
	switch {
	case err != nil:
		return "send failed: " + err.Error()
	case res.Outcome == Sent:
		return ""
	case res.Outcome == Held:
		return HoldDetail(res.Reason)
	default: // Failed: the content did not render
		return res.Reason
	}
}

// SendTest sends one message for an author to preview. The author names the
// address, so it skips Send-eligibility and carries no unsubscribe or tracking, and
// it records no Outbound message and no email.sent Event — but it still passes the
// Workspace freeze and the Sending-domain gate and is DKIM-signed like any send: a
// suspended Workspace sends nothing, test or not (ADR 0015). Outcomes are Sent,
// Failed (the content did not render) or Held; there is no replay.
func (m *Module) SendTest(ctx context.Context, s *ent.Scoped, req TestRequest) (Result, error) {
	ws, err := m.workspace(ctx, s)
	if err != nil {
		return Result{}, err
	}
	g, err := m.gate(ctx, ws, req.FromEmail, req.FromName)
	if err != nil {
		return Result{}, err
	}
	if g.hold != "" {
		return Result{Outcome: Held, Reason: g.hold}, nil
	}
	to := eligibility.NormalizeDestination(req.To)
	built, err := m.compose(Request{Subject: req.Subject, Body: req.Body, Variables: req.Variables}, to, ws)
	if err != nil {
		return Result{Outcome: Failed, Reason: err.Error()}, nil
	}
	built.msg.From, built.msg.FromName, built.msg.To = g.from, g.fromName, to
	if _, err := g.sender.Send(ctx, built.msg); err != nil {
		if errors.Is(err, messaging.ErrUnverifiedSendingDomain) {
			return Result{Outcome: Held, Reason: HoldUnverifiedDomain}, nil
		}
		return Result{}, fmt.Errorf("outbound: test send to %s: %w", to, err)
	}
	return Result{Outcome: Sent}, nil
}
