package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent/authattempt"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

const (
	loginPath     = "/site/auth/login"
	loginFailures = 5
	// An address no account has.
	unknownLoginEmail = "nobody@nowhere.test"
)

// frozenClock is the injected `now` of the account attempt module.
type frozenClock struct{ t time.Time }

func (c *frozenClock) now() time.Time { return c.t }

func loginEnv(t *testing.T, ipLimit int) (*testhelper.TestEnv, *frozenClock) {
	t.Helper()
	c := &frozenClock{t: time.Now()}
	env := testhelper.Setup(t,
		testhelper.WithClock(c.now),
		testhelper.WithRateLimits(config.RateLimits{LoginFailures: loginFailures, LoginIP: ipLimit}))
	return env, c
}

func login(t *testing.T, env *testhelper.TestEnv, user, password string) *httptest.ResponseRecorder {
	t.Helper()
	return postJSON(t, env, loginPath, fmt.Sprintf(`{"email":%q,"password":%q}`, user, password), nil)
}

func failLogins(t *testing.T, env *testhelper.TestEnv, user string, times int) {
	t.Helper()
	for i := range times {
		rec := login(t, env, user, "wrong-password")
		require.Equal(t, http.StatusUnauthorized, rec.Code, "failure %d", i+1)
	}
}

func TestLoginAnswers429AfterTheConfiguredFailuresEvenWithTheRightPassword(t *testing.T) {
	env, _ := loginEnv(t, 0)
	failLogins(t, env, fixtures.OwnerJohnEmail, loginFailures)

	for name, password := range map[string]string{"wrong": "wrong-password", "correct": fixtures.OwnerJohnPassword} {
		rec := login(t, env, fixtures.OwnerJohnEmail, password)
		require.Equal(t, http.StatusTooManyRequests, rec.Code, name)
		assert.Equal(t, "1", rec.Header().Get("Retry-After"), name)
		assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
		assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
		var body struct {
			Status     int `json:"status"`
			RetryAfter int `json:"retryAfter"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, 429, body.Status)
		assert.Equal(t, 1, body.RetryAfter)
		assert.Empty(t, jwtOf(rec), "a correct password during the delay must not log in")
	}
}

func TestLoginWorksAgainOnceTheDelayHasPassedAndResetsTheCounter(t *testing.T) {
	env, clock := loginEnv(t, 0)
	failLogins(t, env, fixtures.OwnerJohnEmail, loginFailures)
	clock.t = clock.t.Add(2 * time.Second)

	rec := login(t, env, fixtures.OwnerJohnEmail, fixtures.OwnerJohnPassword)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotEmpty(t, jwtOf(rec))

	n, err := env.DB.AuthAttempt.Query().Where(authattempt.Email(fixtures.OwnerJohnEmail)).Count(t.Context())
	require.NoError(t, err)
	assert.Zero(t, n, "a success resets the counter")
}

func TestASuccessBeforeTheThresholdResetsTheCounter(t *testing.T) {
	env, _ := loginEnv(t, 0)
	failLogins(t, env, fixtures.OwnerJohnEmail, loginFailures-1)
	require.Equal(t, http.StatusOK, login(t, env, fixtures.OwnerJohnEmail, fixtures.OwnerJohnPassword).Code)
	failLogins(t, env, fixtures.OwnerJohnEmail, loginFailures-1)
	assert.Equal(t, http.StatusOK, login(t, env, fixtures.OwnerJohnEmail, fixtures.OwnerJohnPassword).Code)
}

func TestUnknownEmailsThrottleLikeKnownOnesAndLeaveRows(t *testing.T) {
	env, _ := loginEnv(t, 0)
	failLogins(t, env, unknownLoginEmail, loginFailures)
	assert.Equal(t, http.StatusTooManyRequests, login(t, env, unknownLoginEmail, "whatever").Code)

	row, err := env.DB.AuthAttempt.Query().Where(authattempt.Email(unknownLoginEmail)).Only(t.Context())
	require.NoError(t, err)
	assert.Equal(t, loginFailures, row.Failures)

	// A counter deep into its delay (fixture) answers 429 straight away.
	assert.Equal(t, http.StatusTooManyRequests, login(t, env, fixtures.GhostLoginAttemptEmail, "whatever").Code)
}

func TestLoginIsLimitedPerIPAcrossAccounts(t *testing.T) {
	const ipLimit = 3
	env, _ := loginEnv(t, ipLimit)
	for i := range ipLimit {
		rec := login(t, env, fmt.Sprintf("spray%d@nowhere.test", i), "x")
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	}
	rec := login(t, env, "spray-last@nowhere.test", "x")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))

	other := postJSON(t, env, loginPath, `{"email":"a@nowhere.test","password":"x"}`, map[string]string{"X-Forwarded-For": "203.0.113.9"})
	assert.Equal(t, http.StatusUnauthorized, other.Code, "another address has its own budget")
}

func TestDisabledLoginLimitsNeverThrottle(t *testing.T) {
	env := testhelper.Setup(t)
	failLogins(t, env, fixtures.OwnerJohnEmail, loginFailures+3)
	assert.Equal(t, http.StatusOK, login(t, env, fixtures.OwnerJohnEmail, fixtures.OwnerJohnPassword).Code)
}

func jwtOf(rec *httptest.ResponseRecorder) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "JWT" {
			return c.Value
		}
	}
	return ""
}
