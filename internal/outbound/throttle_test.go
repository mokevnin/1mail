package outbound_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/outboundmessage"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/outbound"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// A provider reply meaning "sending too fast" or "daily quota exceeded" is a
// Deferral with a backoff (ADR 0023), observed at the Outbound send seam with a
// fake provider.

func TestProviderTooFastIsADeferralThenSentExactlyOnce(t *testing.T) {
	env := testhelper.Setup(t)
	m := newModule(env)

	env.CustomerMail.SetErr(messaging.ErrThrottled)
	res := send(t, m, env, "bc:fast")
	assert.Equal(t, outbound.Deferral, res.Outcome)
	assert.Equal(t, outbound.ThrottleBackoff, res.Wait)
	assert.Empty(t, res.Reason)

	n, err := acme(env).OutboundMessage().Query().Where(outboundmessage.IdempotencyKey("bc:fast")).Count(context.Background())
	require.NoError(t, err)
	assert.Zero(t, n, "no claim is left behind: nothing failed, no attempt is spent")

	env.CustomerMail.SetErr(nil)
	retry := send(t, m, env, "bc:fast")
	assert.Equal(t, outbound.Sent, retry.Outcome)
	assert.Len(t, env.CustomerMail.Messages(), 1, "the logical send reached the provider once")

	again := send(t, m, env, "bc:fast")
	assert.Equal(t, outbound.Sent, again.Outcome)
	assert.True(t, again.Replayed)
	assert.Len(t, env.CustomerMail.Messages(), 1, "idempotency still holds after deferral and retry")
}

func TestProviderDailyQuotaIsADeferralWithALongerBackoff(t *testing.T) {
	env := testhelper.Setup(t)
	m := newModule(env)

	env.CustomerMail.SetErr(messaging.ErrQuotaExceeded)
	res := send(t, m, env, "bc:quota")
	assert.Equal(t, outbound.Deferral, res.Outcome)
	assert.Equal(t, outbound.QuotaBackoff, res.Wait)
	assert.Greater(t, outbound.QuotaBackoff, outbound.ThrottleBackoff)
}

func TestWrappedThrottleReplyIsStillADeferral(t *testing.T) {
	env := testhelper.Setup(t)
	m := newModule(env)

	env.CustomerMail.SetErr(errors.Join(errors.New("ses: send"), messaging.ErrThrottled))
	assert.Equal(t, outbound.Deferral, send(t, m, env, "bc:wrapped").Outcome)
}

func TestTransactionalThrottleReplyIsStillADeferral(t *testing.T) {
	env := testhelper.Setup(t)
	m := newModule(env)

	env.CustomerMail.SetErr(messaging.ErrThrottled)
	res, err := m.Send(context.Background(), acme(env), transactional("tx:fast", "a@example.com"))
	require.NoError(t, err)
	assert.Equal(t, outbound.Deferral, res.Outcome, "the provider is busy whatever the surface; the caller decides how to wait")
}

func TestGenuineProviderFailureStillFailsTheMessage(t *testing.T) {
	env := testhelper.Setup(t)
	m := newModule(env, outbound.WithLease(time.Nanosecond))
	ctx := context.Background()

	env.CustomerMail.SetErr(errors.New("554 mailbox does not exist"))
	_, err := m.Send(ctx, acme(env), marketing(t, env, "bc:perm", fixtures.ContactAliceID))
	require.Error(t, err, "a real failure is still an error, not a Deferral")

	time.Sleep(time.Millisecond)
	require.NoError(t, m.MarkFailed(ctx, acme(env), "bc:perm", err))
	assert.Equal(t, outboundmessage.StatusFailed, byKey(t, env, "bc:perm").Status)
}
