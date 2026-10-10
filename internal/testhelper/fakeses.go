package testhelper

import (
	"context"
	"sync"

	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/messaging/ses"
	"github.com/mokevnin/1mail/internal/messaging/smtp"
)

// FakeSES stands in for the SES account behind every "ses" Integration the server
// builds in a test: its send quota is whatever the test scripts (by default none is
// reported and the lookup succeeds), so no test ever reaches the real SES API.
type FakeSES struct {
	mu    sync.Mutex
	quota messaging.Quota
	err   error
	calls int
}

// SetQuota makes the next GetSendQuota calls report quota (and succeed).
func (f *FakeSES) SetQuota(perSecond, perDay int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.quota, f.err = messaging.Quota{PerSecond: &perSecond, PerDay: &perDay}, nil
}

// SetQuotaErr makes the next GetSendQuota calls fail with err (nil clears it): a
// missing ses:GetSendQuota permission, or an SES-compatible service without the call.
func (f *FakeSES) SetQuotaErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

// QuotaCalls is how many times a quota lookup reached the fake.
func (f *FakeSES) QuotaCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *FakeSES) sendQuota() (messaging.Quota, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.quota, f.err
}

// quotaSender is the real SES sender with its quota lookup answered by the fake.
type quotaSender struct {
	messaging.EmailSender
	fake *FakeSES
}

func (s quotaSender) SendQuota(context.Context) (messaging.Quota, error) {
	return s.fake.sendQuota()
}

// catalogWith is the built-in catalog with SES's quota lookup answered by fake.
func catalogWith(fake *FakeSES) *messaging.Catalog {
	real := ses.Descriptor()
	real.Build = func(config []byte, signer messaging.Signer) (any, error) {
		built, err := ses.Descriptor().Build(config, signer)
		if err != nil {
			return nil, err
		}
		return quotaSender{EmailSender: built.(messaging.EmailSender), fake: fake}, nil
	}
	return messaging.NewCatalog(smtp.Descriptor(), real)
}
