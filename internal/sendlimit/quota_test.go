package sendlimit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/secrets"
	"github.com/mokevnin/1mail/internal/sendlimit"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// fakeSES stands in for the SES account: its sender answers GetSendQuota with the
// scripted quota or error. It is a catalog provider like the real one, so discovery
// builds it from the stored, encrypted config exactly as it builds the real sender.
type fakeSES struct {
	quota messaging.Quota
	err   error
	calls int
}

func (f *fakeSES) Send(context.Context, messaging.EmailMessage) (messaging.Receipt, error) {
	return messaging.Receipt{}, errors.New("fakeSES does not send")
}

func (f *fakeSES) SendQuota(context.Context) (messaging.Quota, error) {
	f.calls++
	return f.quota, f.err
}

func (f *fakeSES) catalog() *messaging.Catalog {
	return messaging.NewCatalog(
		messaging.ProviderDescriptor{
			Channel: messaging.ChannelEmail, Provider: messaging.ProviderSES,
			Build: func([]byte, messaging.Signer) (any, error) { return f, nil },
		},
		// smtp builds a sender that cannot report a quota.
		messaging.ProviderDescriptor{
			Channel: messaging.ChannelEmail, Provider: messaging.ProviderSMTP,
			Build: func([]byte, messaging.Signer) (any, error) { return plainSender{}, nil },
		},
	)
}

type plainSender struct{}

func (plainSender) Send(context.Context, messaging.EmailMessage) (messaging.Receipt, error) {
	return messaging.Receipt{}, nil
}

func envCipher(t *testing.T) *secrets.Cipher {
	t.Helper()
	cfg, err := config.Load("test")
	require.NoError(t, err)
	c, err := secrets.NewCipher(cfg.EncryptionKey)
	require.NoError(t, err)
	return c
}

func reload(t *testing.T, s *ent.Scoped, id int64) *ent.Integration {
	t.Helper()
	row, err := s.Integration().Get(t.Context(), id)
	require.NoError(t, err)
	return row
}

func TestRefreshQuotaStoresTheProviderValuesAndFeedsTheCeiling(t *testing.T) {
	env := testhelper.Setup(t)
	s := env.DB.Scoped(fixtures.AcmeID)
	ses := &fakeSES{quota: messaging.Quota{PerSecond: ptr(14), PerDay: ptr(50000)}}

	require.NoError(t, sendlimit.RefreshQuota(t.Context(), s, envCipher(t), ses.catalog(), reload(t, s, fixtures.IntegrationAcmeSesID)))

	row := reload(t, s, fixtures.IntegrationAcmeSesID)
	assert.Equal(t, ptr(14), row.ProviderMaxPerSecond)
	assert.Equal(t, ptr(50000), row.ProviderMaxPerDay)
	assert.False(t, row.ProviderQuotaUnavailable)
	assert.NotNil(t, row.ProviderQuotaCheckedAt)
	eff := sendlimit.EffectiveOf(row)
	assert.Equal(t, sendlimit.Value{Limit: ptr(14), Source: sendlimit.SourceProvider}, eff.PerSecond)

	// A manual value below the provider's wins as a value.
	require.NoError(t, s.Integration().UpdateOneID(row.ID).SetMaxPerSecond(5).Exec(t.Context()))
	eff = sendlimit.EffectiveOf(reload(t, s, row.ID))
	assert.Equal(t, sendlimit.Value{Limit: ptr(5), Source: sendlimit.SourceManual}, eff.PerSecond)
}

func TestRefreshQuotaFailureKeepsManualOrUnlimitedAndRecordsAWarning(t *testing.T) {
	env := testhelper.Setup(t)
	s := env.DB.Scoped(fixtures.AcmeID)
	ses := &fakeSES{err: errors.New("AccessDenied: ses:GetSendQuota")}
	require.NoError(t, s.Integration().UpdateOneID(fixtures.IntegrationAcmeSesID).SetMaxPerDay(1000).Exec(t.Context()))

	err := sendlimit.RefreshQuota(t.Context(), s, envCipher(t), ses.catalog(), reload(t, s, fixtures.IntegrationAcmeSesID))
	require.NoError(t, err, "a failed lookup is recorded, never returned: it must not fail a save or a job")

	row := reload(t, s, fixtures.IntegrationAcmeSesID)
	assert.True(t, row.ProviderQuotaUnavailable)
	assert.NotNil(t, row.ProviderQuotaCheckedAt, "a failed lookup still counts as a check")
	eff := sendlimit.EffectiveOf(row)
	assert.True(t, eff.ProviderQuotaUnavailable)
	assert.Equal(t, sendlimit.Value{Limit: ptr(1000), Source: sendlimit.SourceManual}, eff.PerDay)
	assert.Equal(t, sendlimit.Value{}, eff.PerSecond, "no manual value and no provider value is unlimited")
}

func TestRefreshQuotaSuccessAfterFailureClearsTheWarning(t *testing.T) {
	env := testhelper.Setup(t)
	s := env.DB.Scoped(fixtures.AcmeID)
	ses := &fakeSES{err: errors.New("AccessDenied")}
	cipher, catalog := envCipher(t), ses.catalog()

	require.NoError(t, sendlimit.RefreshQuota(t.Context(), s, cipher, catalog, reload(t, s, fixtures.IntegrationAcmeSesID)))
	require.True(t, reload(t, s, fixtures.IntegrationAcmeSesID).ProviderQuotaUnavailable)

	ses.err, ses.quota = nil, messaging.Quota{PerSecond: ptr(1)}
	require.NoError(t, sendlimit.RefreshQuota(t.Context(), s, cipher, catalog, reload(t, s, fixtures.IntegrationAcmeSesID)))

	row := reload(t, s, fixtures.IntegrationAcmeSesID)
	assert.False(t, row.ProviderQuotaUnavailable)
	assert.Equal(t, ptr(1), row.ProviderMaxPerSecond)
	assert.Nil(t, row.ProviderMaxPerDay)
}

func TestRefreshQuotaFailureKeepsTheLastKnownProviderValues(t *testing.T) {
	env := testhelper.Setup(t)
	s := env.DB.Scoped(fixtures.AcmeID)
	ses := &fakeSES{quota: messaging.Quota{PerSecond: ptr(14), PerDay: ptr(50000)}}
	cipher, catalog := envCipher(t), ses.catalog()
	require.NoError(t, sendlimit.RefreshQuota(t.Context(), s, cipher, catalog, reload(t, s, fixtures.IntegrationAcmeSesID)))

	ses.err = errors.New("timeout")
	require.NoError(t, sendlimit.RefreshQuota(t.Context(), s, cipher, catalog, reload(t, s, fixtures.IntegrationAcmeSesID)))

	row := reload(t, s, fixtures.IntegrationAcmeSesID)
	assert.True(t, row.ProviderQuotaUnavailable)
	assert.Equal(t, ptr(14), row.ProviderMaxPerSecond, "a transient blip must not lift a ceiling that was known")
}

func TestRefreshQuotaIgnoresProvidersThatCannotReportOne(t *testing.T) {
	env := testhelper.Setup(t)
	s := env.DB.Scoped(fixtures.AcmeID)
	ses := &fakeSES{}

	require.NoError(t, sendlimit.RefreshQuota(t.Context(), s, envCipher(t), ses.catalog(), reload(t, s, fixtures.IntegrationAcmeDefaultID)))

	row := reload(t, s, fixtures.IntegrationAcmeDefaultID)
	assert.False(t, row.ProviderQuotaUnavailable, "SMTP has no quota to be unavailable")
	assert.Nil(t, row.ProviderQuotaCheckedAt)
	assert.Zero(t, ses.calls)
}
