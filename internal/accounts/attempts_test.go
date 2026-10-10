package accounts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/authattempt"
	"github.com/mokevnin/1mail/internal/accounts"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// clock is the injected `now` of the attempts module.
type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newAttempts(t *testing.T) (*accounts.Attempts, *clock, *testhelper.TestEnv) {
	t.Helper()
	env := testhelper.Setup(t)
	c := &clock{t: time.Now()}
	a := accounts.NewAttempts(env.DB,
		accounts.WithClock(c.now),
		accounts.WithRule(accounts.KindLogin, accounts.LoginRule(5)),
	)
	return a, c, env
}

func fail(t *testing.T, a *accounts.Attempts, email string, times int) {
	t.Helper()
	for range times {
		require.NoError(t, a.RecordFailure(t.Context(), accounts.KindLogin, email))
	}
}

func delay(t *testing.T, a *accounts.Attempts, email string) time.Duration {
	t.Helper()
	d, err := a.Delay(t.Context(), accounts.KindLogin, email)
	require.NoError(t, err)
	return d
}

func TestNoDelayBeforeTheThreshold(t *testing.T) {
	a, _, _ := newAttempts(t)
	fail(t, a, "x@attempts.test", 4)
	assert.Zero(t, delay(t, a, "x@attempts.test"))
}

func TestDelayDoublesPerFailureAndIsCapped(t *testing.T) {
	a, _, _ := newAttempts(t)
	fail(t, a, "x@attempts.test", 5)
	assert.Equal(t, 1*time.Second, delay(t, a, "x@attempts.test"))
	want := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}
	for _, w := range want {
		fail(t, a, "x@attempts.test", 1)
		assert.Equal(t, w, delay(t, a, "x@attempts.test"))
	}
	fail(t, a, "x@attempts.test", 40)
	assert.Equal(t, 15*time.Minute, delay(t, a, "x@attempts.test"), "capped at 15 minutes")
}

func TestDelayShrinksAsTheClockAdvancesAndEnds(t *testing.T) {
	a, c, _ := newAttempts(t)
	fail(t, a, "x@attempts.test", 7) // 4 s
	c.advance(3 * time.Second)
	assert.Equal(t, 1*time.Second, delay(t, a, "x@attempts.test"))
	c.advance(1 * time.Second)
	assert.Zero(t, delay(t, a, "x@attempts.test"))
}

func TestSuccessResetsTheCounter(t *testing.T) {
	a, _, _ := newAttempts(t)
	fail(t, a, "x@attempts.test", 4)
	require.NoError(t, a.RecordSuccess(t.Context(), accounts.KindLogin, "x@attempts.test"))
	fail(t, a, "x@attempts.test", 4)
	assert.Zero(t, delay(t, a, "x@attempts.test"), "the earlier failures no longer count")
}

func TestFailuresOutsideTheWindowStartOver(t *testing.T) {
	a, c, _ := newAttempts(t)
	fail(t, a, "x@attempts.test", 4)
	c.advance(16 * time.Minute)
	fail(t, a, "x@attempts.test", 1)
	assert.Zero(t, delay(t, a, "x@attempts.test"), "4 old failures plus 1 new one is not 5")
}

func TestEmailsAreNormalizedAndKindsAreSeparate(t *testing.T) {
	a, _, env := newAttempts(t)
	require.NoError(t, a.RecordFailure(t.Context(), accounts.KindLogin, " X@Attempts.Test "))
	fail(t, a, "x@attempts.test", 4)
	assert.Equal(t, 1*time.Second, delay(t, a, "X@ATTEMPTS.TEST"))

	d, err := a.Delay(t.Context(), accounts.KindPasswordReset, "x@attempts.test")
	require.NoError(t, err)
	assert.Zero(t, d, "another kind has its own counter")

	n, err := env.DB.AuthAttempt.Query().Where(authattempt.Email("x@attempts.test")).Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, n, "one login counter for every spelling of the address")
}

func TestUnknownEmailsGetRowsLikeKnownOnes(t *testing.T) {
	a, _, env := newAttempts(t)
	fail(t, a, "nobody@attempts.test", 5)
	assert.Equal(t, 1*time.Second, delay(t, a, "nobody@attempts.test"))
	row, err := env.DB.AuthAttempt.Query().
		Where(authattempt.Email("nobody@attempts.test"), authattempt.KindEQ(authattempt.KindLogin)).Only(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 5, row.Failures)
}

func TestDisabledRuleNeverDelaysOrWrites(t *testing.T) {
	env := testhelper.Setup(t)
	a := accounts.NewAttempts(env.DB, accounts.WithRule(accounts.KindLogin, accounts.LoginRule(0)))
	fail(t, a, "x@attempts.test", 20)
	assert.Zero(t, delay(t, a, "x@attempts.test"))
	n, err := env.DB.AuthAttempt.Query().Where(authattempt.Email("x@attempts.test")).Count(t.Context())
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestPurgeRemovesStaleRowsOnly(t *testing.T) {
	env := testhelper.Setup(t)
	a := accounts.NewAttempts(env.DB)
	removed, err := a.Purge(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	_, err = env.DB.AuthAttempt.Get(t.Context(), fixtures.StaleLoginAttemptID)
	assert.Error(t, err, "the stale row is gone")
	_, err = env.DB.AuthAttempt.Get(t.Context(), fixtures.FreshLoginAttemptID)
	assert.NoError(t, err, "the current row stays")
}
