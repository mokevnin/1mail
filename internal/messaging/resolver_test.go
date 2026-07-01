package messaging_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/messaging/registry"
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

// envCipher builds the cipher over the test environment's ENCRYPTION_KEY — the
// same key the fixtures were sealed with — so fixture-stored configs decrypt.
func envCipher(t *testing.T) *secrets.Cipher {
	t.Helper()
	cfg, err := config.Load("test")
	require.NoError(t, err)
	c, err := secrets.NewCipher(cfg.EncryptionKey)
	require.NoError(t, err)
	return c
}

// Resolver loads the workspace's default email integration, decrypts it and
// builds a live EmailSender via the catalog. Workspace 1 ("acme") is seeded with
// a default SMTP integration.
func TestResolverEmailSender(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	resolver := messaging.NewResolver(env.DB, envCipher(t), registry.Default())

	sender, err := resolver.EmailSender(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, sender)
}

// A workspace without any integration has no default to resolve.
func TestResolverNoDefault(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	ws, err := env.DB.Workspace.Create().
		SetName("No Provider").SetSlug("no-provider").
		SetCollectKey("no-provider-ck").SetIngestKey("no-provider-ik").Save(ctx)
	require.NoError(t, err)

	resolver := messaging.NewResolver(env.DB, newTestCipher(t), registry.Default())

	_, err = resolver.EmailSender(ctx, ws.ID)
	assert.ErrorIs(t, err, messaging.ErrNoProvider)
}
