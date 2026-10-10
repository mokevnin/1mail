package accounts

import (
	"context"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/authattempt"
)

// Kind is the action an attempt counter belongs to. Each kind has its own counter
// per address and its own Rule.
type Kind = authattempt.Kind

const (
	KindLogin         = authattempt.KindLogin
	KindPasswordReset = authattempt.KindPasswordReset
)

// Rule is the throttle of one Kind (ADR 0018): after Threshold failures within
// Window the next attempt must wait Base, doubling with every further failure up to
// Cap, measured from the last failure. A Threshold of 0 disables the Kind: nothing
// is counted and nothing waits.
type Rule struct {
	Threshold int
	Window    time.Duration
	Base      time.Duration
	Cap       time.Duration
}

// ResetRule is the password-reset budget: at most threshold mails per address per
// hour (checked with Reached; there is no delay).
func ResetRule(threshold int) Rule { return Rule{Threshold: threshold, Window: time.Hour} }

// LoginRule is the login throttle: threshold failures in 15 minutes, then 1 s, 2 s,
// ... capped at 15 minutes. No hard lockout: a delay always ends.
func LoginRule(threshold int) Rule {
	return Rule{Threshold: threshold, Window: 15 * time.Minute, Base: time.Second, Cap: 15 * time.Minute}
}

// Attempts counts failed attempts per normalized email and Kind in Postgres, so the
// count follows the account (not the source IP) and is exact across replicas. Rows
// are written for unknown emails too, so the table cannot be used to enumerate
// accounts. It takes the raw client: AuthAttempt has no Workspace (see the package
// doc). now is injectable, because this is the one path whose time cannot come from
// httprate.
type Attempts struct {
	ent   *ent.Client
	now   func() time.Time
	rules map[Kind]Rule
}

// AttemptsOption tunes NewAttempts.
type AttemptsOption func(*Attempts)

// WithClock replaces time.Now, for tests.
func WithClock(now func() time.Time) AttemptsOption { return func(a *Attempts) { a.now = now } }

// WithRule sets the throttle of a Kind. Until it is set a Kind is not throttled.
func WithRule(kind Kind, rule Rule) AttemptsOption {
	return func(a *Attempts) { a.rules[kind] = rule }
}

// NewAttempts builds the module over the ent client.
func NewAttempts(client *ent.Client, opts ...AttemptsOption) *Attempts {
	// Every Kind starts disabled but with its window, so a bare module (the purge
	// job's) still knows when a row is stale.
	a := &Attempts{ent: client, now: time.Now, rules: map[Kind]Rule{
		KindLogin:         LoginRule(0),
		KindPasswordReset: ResetRule(0),
	}}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// NormalizeEmail is the counter key: trimmed and lower-cased.
func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func (a *Attempts) rule(kind Kind) (Rule, bool) {
	r, ok := a.rules[kind]
	return r, ok && r.Threshold > 0
}

// Limit is the failure threshold of the Kind (0 when it is not throttled).
func (a *Attempts) Limit(kind Kind) int {
	r, _ := a.rule(kind)
	return r.Threshold
}

// Now is the module's clock, so callers compute reset times on the same time base.
func (a *Attempts) Now() time.Time { return a.now() }

// RecordFailure counts one failed attempt. Failures older than the Rule's window
// start a new count.
func (a *Attempts) RecordFailure(ctx context.Context, kind Kind, email string) error {
	rule, ok := a.rule(kind)
	if !ok {
		return nil
	}
	email = NormalizeEmail(email)
	now := a.now()
	n, err := a.ent.AuthAttempt.Update().
		Where(authattempt.Email(email), authattempt.KindEQ(kind), authattempt.LastAttemptAtGTE(now.Add(-rule.Window))).
		AddFailures(1).
		SetLastAttemptAt(now).
		Save(ctx)
	if err != nil || n > 0 {
		return err
	}
	// No current row: insert one, or restart a stale one (the unique key makes a
	// concurrent first failure collapse into one row).
	return a.ent.AuthAttempt.Create().
		SetEmail(email).
		SetKind(kind).
		SetFailures(1).
		SetLastAttemptAt(now).
		OnConflictColumns(authattempt.FieldEmail, authattempt.FieldKind).
		UpdateNewValues().
		Exec(ctx)
}

// RecordSuccess resets the counter of the address.
func (a *Attempts) RecordSuccess(ctx context.Context, kind Kind, email string) error {
	if _, ok := a.rule(kind); !ok {
		return nil
	}
	_, err := a.ent.AuthAttempt.Delete().
		Where(authattempt.Email(NormalizeEmail(email)), authattempt.KindEQ(kind)).
		Exec(ctx)
	return err
}

// Delay reports how long the address must still wait before the next attempt; 0
// means it may proceed.
func (a *Attempts) Delay(ctx context.Context, kind Kind, email string) (time.Duration, error) {
	rule, ok := a.rule(kind)
	if !ok {
		return 0, nil
	}
	row, err := a.ent.AuthAttempt.Query().
		Where(authattempt.Email(NormalizeEmail(email)), authattempt.KindEQ(kind)).
		Only(ctx)
	if ent.IsNotFound(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	now := a.now()
	if row.Failures < rule.Threshold || now.Sub(row.LastAttemptAt) > rule.Window {
		return 0, nil
	}
	wait := rule.Base
	for range row.Failures - rule.Threshold {
		wait *= 2
		if wait >= rule.Cap {
			wait = rule.Cap
			break
		}
	}
	wait = min(wait, rule.Cap)
	return max(row.LastAttemptAt.Add(wait).Sub(now), 0), nil
}

// Reached reports whether the address has used up the Rule's threshold within the
// window. It is the plain count-in-window check for a Kind that is a budget rather
// than a delay (password reset: at most Threshold mails per Window); the Rule's
// Base and Cap are not involved. A Kind that is not throttled is never reached.
func (a *Attempts) Reached(ctx context.Context, kind Kind, email string) (bool, error) {
	rule, ok := a.rule(kind)
	if !ok {
		return false, nil
	}
	row, err := a.ent.AuthAttempt.Query().
		Where(authattempt.Email(NormalizeEmail(email)), authattempt.KindEQ(kind),
			authattempt.LastAttemptAtGTE(a.now().Add(-rule.Window))).
		Only(ctx)
	if ent.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return row.Failures >= rule.Threshold, nil
}

// Purge deletes the rows whose last failure is past their Kind's window (a delay is
// never longer than the window, so they no longer matter) and reports how many.
func (a *Attempts) Purge(ctx context.Context) (int, error) {
	now := a.now()
	total := 0
	for kind, rule := range a.rules {
		n, err := a.ent.AuthAttempt.Delete().
			Where(authattempt.KindEQ(kind), authattempt.LastAttemptAtLT(now.Add(-rule.Window))).
			Exec(ctx)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}
