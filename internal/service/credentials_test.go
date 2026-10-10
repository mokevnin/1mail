package service_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/service"
)

func TestAPITokenRoundTripsThroughItsValueAndParses(t *testing.T) {
	prefix, err := service.GenerateTokenPrefix()
	require.NoError(t, err)
	secret, err := service.GenerateTokenSecret()
	require.NoError(t, err)
	assert.Len(t, prefix, 12)
	assert.Len(t, secret, 40)

	parsed := service.ParseToken(service.TokenValue(prefix, secret))
	require.NotNil(t, parsed)
	assert.Equal(t, prefix, parsed.Prefix)
	assert.Equal(t, secret, parsed.Secret)
}

func TestParseTokenRejectsMalformedValues(t *testing.T) {
	for _, bad := range []string{"", "omtk", "omtk_prefix", "xxxx_prefix_secret", "omtk_UPPER_secret", "omtk_prefix_sec ret", "omtk__secret"} {
		assert.Nil(t, service.ParseToken(bad), "%q", bad)
	}
}

func TestTokenSecretHashVerifiesOnlyTheOriginal(t *testing.T) {
	hash, err := service.HashTokenSecret("s3cret")
	require.NoError(t, err)
	assert.NotEqual(t, "s3cret", hash)
	assert.True(t, service.VerifyTokenSecret("s3cret", hash))
	assert.False(t, service.VerifyTokenSecret("other", hash))
	assert.False(t, service.VerifyTokenSecret("s3cret", "not-a-bcrypt-hash"))
}

func TestWorkspaceKeysAreTypedAndUnique(t *testing.T) {
	collect, err := service.GenerateCollectKey()
	require.NoError(t, err)
	ingest, err := service.GenerateIngestKey()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(collect, "omck_"))
	assert.True(t, strings.HasPrefix(ingest, "omik_"))
	assert.Len(t, collect, len("omck_")+40)
	assert.Len(t, ingest, len("omik_")+40)

	again, err := service.GenerateCollectKey()
	require.NoError(t, err)
	assert.NotEqual(t, collect, again)
}

func TestWebhookSecretIsStandardWebhooksFormat(t *testing.T) {
	secret, err := service.GenerateWebhookSecret()
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(secret, "whsec_"))
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	require.NoError(t, err)
	assert.Len(t, raw, 24)
}

func TestValidWebhookURLAcceptsOnlyAbsoluteHTTP(t *testing.T) {
	for url, want := range map[string]bool{
		"https://example.com/hook":  true,
		"http://example.com:8080/x": true,
		"ftp://example.com/hook":    false,
		"example.com/hook":          false,
		"https://":                  false,
		"/relative":                 false,
		"":                          false,
		"http://%zz":                false,
	} {
		assert.Equal(t, want, service.ValidWebhookURL(url), url)
	}
}

func TestPasswordHashVerifiesOnlyTheOriginal(t *testing.T) {
	hash, err := service.HashPassword("correct horse")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "$argon2id$"))
	assert.True(t, service.VerifyPassword(hash, "correct horse"))
	assert.False(t, service.VerifyPassword(hash, "wrong"))
	assert.False(t, service.VerifyPassword("not-a-phc-hash", "correct horse"), "an undecodable hash never matches")
	assert.False(t, service.VerifyPassword("", ""))
}

func TestSlugifyTransliteratesAndCollapses(t *testing.T) {
	assert.Equal(t, "privet-mir", service.Slugify("Привет Мир"))
	assert.Equal(t, "acme-co", service.Slugify("  Acme -- Co!! "))
	assert.Empty(t, service.Slugify("!!!"))
}

func TestIsUniqueViolationRecognizesPostgresCode23505(t *testing.T) {
	assert.True(t, service.IsUniqueViolation(&pgconn.PgError{Code: "23505"}))
	assert.True(t, service.IsUniqueViolation(fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: "23505"})))
	assert.False(t, service.IsUniqueViolation(&pgconn.PgError{Code: "23503"}))
	assert.False(t, service.IsUniqueViolation(errors.New("plain")))
	assert.False(t, service.IsUniqueViolation(nil))
}
