package messaging_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mokevnin/1mail/ent/integration"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/messaging/registry"
	"github.com/mokevnin/1mail/internal/messaging/smtp"
	"github.com/mokevnin/1mail/internal/secrets"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCipher(t *testing.T) *secrets.Cipher {
	t.Helper()
	ks, err := secrets.GenerateKeysetBase64()
	require.NoError(t, err)
	c, err := secrets.NewCipher(ks)
	require.NoError(t, err)
	return c
}

// Resolver loads the workspace's default email integration, decrypts it and
// builds a live EmailSender via the catalog. Workspace 1 ("acme") is seeded.
func TestResolverEmailSender(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	cipher := newTestCipher(t)

	cfg, err := json.Marshal(smtp.Config{Host: "smtp.example.com", Port: 587, From: "noreply@acme.test"})
	require.NoError(t, err)
	enc, err := cipher.Encrypt(cfg)
	require.NoError(t, err)

	_, err = env.DB.Integration.Create().
		SetWorkspaceID(1).
		SetName("Default SMTP").
		SetChannel(integration.ChannelEmail).
		SetProvider(integration.ProviderSMTP).
		SetConfigEncrypted(enc).
		SetEnabled(true).
		SetIsDefault(true).
		Save(ctx)
	require.NoError(t, err)

	resolver := messaging.NewResolver(env.DB, cipher, registry.Default())

	sender, err := resolver.EmailSender(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, sender)
}

func TestResolverNoDefault(t *testing.T) {
	env := testhelper.Setup(t)

	resolver := messaging.NewResolver(env.DB, newTestCipher(t), registry.Default())

	_, err := resolver.EmailSender(context.Background(), 1)
	assert.ErrorIs(t, err, messaging.ErrNoProvider)
}
