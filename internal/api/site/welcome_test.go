package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Registration enqueues a platform welcome email. The inline jobs adapter runs it
// synchronously through the system sender, so we assert the real send — not just
// that a job was enqueued.
func TestSiteRegisterSendsWelcomeEmail(t *testing.T) {
	env := testhelper.Setup(t)
	c, err := siteapi.NewClient("http://local/site", noJWT{}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)
	ctx := context.Background()

	res, err := c.SiteAuthRegister(ctx, &siteapi.SiteRegisterInput{
		Name:     "Grace Hopper",
		Email:    siteapi.EmailAddress("grace@example.com"),
		Password: "s3cret-password",
	})
	require.NoError(t, err)
	require.IsType(t, &siteapi.SiteRegisterResult{}, res)

	msgs := env.SystemMail.Messages()
	require.Len(t, msgs, 1, "registration must send exactly one welcome email")
	assert.Equal(t, "grace@example.com", msgs[0].To)
	assert.Contains(t, msgs[0].Subject, "Welcome")
}
