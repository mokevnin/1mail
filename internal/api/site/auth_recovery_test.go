package site_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// publicClient builds an unauthenticated site client for the NoAuth recovery
// endpoints (forgot/reset/verify/confirm).
func publicClient(t *testing.T, env *testhelper.TestEnv) *siteapi.Client {
	t.Helper()
	c, err := siteapi.NewClient("http://local/site", noJWT{}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)
	return c
}

// tokenFromEmail extracts the token query param from an account email's link.
func tokenFromEmail(t *testing.T, body string) string {
	t.Helper()
	idx := strings.Index(body, "token=")
	require.GreaterOrEqualf(t, idx, 0, "email should carry a token link: %q", body)
	rest := body[idx+len("token="):]
	if end := strings.IndexAny(rest, " \n\r\t"); end >= 0 {
		rest = rest[:end]
	}
	tok, err := url.QueryUnescape(rest)
	require.NoError(t, err)
	return tok
}

// lastSystemEmail returns the most recent captured platform email.
func lastSystemEmail(t *testing.T, env *testhelper.TestEnv) (subject, body string) {
	t.Helper()
	msgs := env.SystemMail.Messages()
	require.NotEmpty(t, msgs, "expected a system email to be sent")
	last := msgs[len(msgs)-1]
	return last.Subject, last.Text
}

func TestForgotPasswordFullResetFlow(t *testing.T) {
	env := testhelper.Setup(t)
	c := publicClient(t, env)
	ctx := context.Background()

	require.NoError(t, c.SiteAuthForgotPassword(ctx, &siteapi.SiteForgotPasswordInput{Email: "info@1mail.com"}))

	_, body := lastSystemEmail(t, env)
	token := tokenFromEmail(t, body)

	res, err := c.SiteAuthResetPassword(ctx, &siteapi.SiteResetPasswordInput{Token: token, Password: "brandnewpass1"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteAuthResetPasswordOK{}, res)

	// The new password logs in; the old one no longer does.
	assert.Equal(t, http.StatusOK, loginStatus(t, env, "info@1mail.com", "brandnewpass1"))
	assert.Equal(t, http.StatusForbidden, loginStatus(t, env, "info@1mail.com", "password"))

	// The token is single-use: replaying it after the reset fails (the binding —
	// the password hash — has changed).
	again, err := c.SiteAuthResetPassword(ctx, &siteapi.SiteResetPasswordInput{Token: token, Password: "another123"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, again, "reset token must not be reusable")
}

func TestForgotPasswordUnknownEmailIsSilent(t *testing.T) {
	env := testhelper.Setup(t)
	c := publicClient(t, env)

	require.NoError(t, c.SiteAuthForgotPassword(context.Background(),
		&siteapi.SiteForgotPasswordInput{Email: "nobody@example.com"}))
	assert.Empty(t, env.SystemMail.Messages(), "no email for an unknown address (no enumeration)")
}

func TestResetPasswordRejectsInvalidToken(t *testing.T) {
	env := testhelper.Setup(t)
	c := publicClient(t, env)

	res, err := c.SiteAuthResetPassword(context.Background(),
		&siteapi.SiteResetPasswordInput{Token: "not-a-token", Password: "whatever123"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, res)
}

func TestResetPasswordRejectsWrongPurposeToken(t *testing.T) {
	env := testhelper.Setup(t)
	pub := publicClient(t, env)
	authed := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	// Mint a verify token via resend, then try to use it as a reset token.
	require.NoError(t, authed.SiteUserResendVerification(ctx))
	_, body := lastSystemEmail(t, env)
	verifyToken := tokenFromEmail(t, body)

	res, err := pub.SiteAuthResetPassword(ctx, &siteapi.SiteResetPasswordInput{Token: verifyToken, Password: "whatever123"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, res, "a verify token must not reset a password")
}

func TestVerifyEmailFlow(t *testing.T) {
	env := testhelper.Setup(t)
	authed := siteClient(t, env, "info@1mail.com")
	pub := publicClient(t, env)
	ctx := context.Background()

	// Seed user starts unverified.
	me, err := authed.SiteUserGetMe(ctx)
	require.NoError(t, err)
	require.False(t, me.EmailVerified)

	require.NoError(t, authed.SiteUserResendVerification(ctx))
	_, body := lastSystemEmail(t, env)
	token := tokenFromEmail(t, body)

	res, err := pub.SiteAuthVerifyEmail(ctx, &siteapi.SiteVerifyEmailInput{Token: token})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteAuthVerifyEmailOK{}, res)

	me, err = authed.SiteUserGetMe(ctx)
	require.NoError(t, err)
	assert.True(t, me.EmailVerified)

	// Replaying the link is idempotent (still OK).
	res, err = pub.SiteAuthVerifyEmail(ctx, &siteapi.SiteVerifyEmailInput{Token: token})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteAuthVerifyEmailOK{}, res)
}

func TestEmailChangeRequiresCurrentPassword(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteUserEmailChange(context.Background(), &siteapi.SiteEmailChangeInput{
		NewEmail:        "new@example.com",
		CurrentPassword: "wrong",
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteUserEmailChangeForbidden{}, res)
	assert.Empty(t, env.SystemMail.Messages())
}

func TestEmailChangeRejectsTakenAddress(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// Register a second user so its address is taken.
	pub := publicClient(t, env)
	reg, err := pub.SiteAuthRegister(ctx, &siteapi.SiteRegisterInput{
		Name: "Jane", Email: "jane@example.com", Password: "password123",
	})
	require.NoError(t, err)
	require.IsType(t, &siteapi.SiteRegisterResult{}, reg)

	c := siteClient(t, env, "info@1mail.com")
	res, err := c.SiteUserEmailChange(ctx, &siteapi.SiteEmailChangeInput{
		NewEmail:        "jane@example.com",
		CurrentPassword: "password",
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteUserEmailChangeConflict{}, res)
}

func TestEmailChangeConfirmSwapsEmail(t *testing.T) {
	env := testhelper.Setup(t)
	authed := siteClient(t, env, "info@1mail.com")
	pub := publicClient(t, env)
	ctx := context.Background()

	res, err := authed.SiteUserEmailChange(ctx, &siteapi.SiteEmailChangeInput{
		NewEmail:        "moved@example.com",
		CurrentPassword: "password",
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteUserEmailChangeAccepted{}, res)

	// The confirmation link is sent to the NEW address.
	subject, body := lastSystemEmail(t, env)
	assert.Contains(t, subject, "new")
	token := tokenFromEmail(t, body)

	confirm, err := pub.SiteAuthConfirmEmailChange(ctx, &siteapi.SiteConfirmEmailChangeInput{Token: token})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteAuthConfirmEmailChangeOK{}, confirm)

	// Login now works with the new email, not the old one.
	assert.Equal(t, http.StatusOK, loginStatus(t, env, "moved@example.com", "password"))
	assert.Equal(t, http.StatusForbidden, loginStatus(t, env, "info@1mail.com", "password"))
}
