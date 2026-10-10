package sendlimit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/sendlimit"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func TestEffectiveOfReportsManualValuesAndTheirSource(t *testing.T) {
	perSecond, perDay := 14, 50000

	none := sendlimit.EffectiveOf(&ent.Integration{})
	assert.True(t, none.Unlimited())
	assert.Nil(t, none.PerSecond.Limit)
	assert.Empty(t, none.PerSecond.Source, "no ceiling has no source")

	both := sendlimit.EffectiveOf(&ent.Integration{MaxPerSecond: &perSecond, MaxPerDay: &perDay})
	assert.False(t, both.Unlimited())
	assert.Equal(t, sendlimit.Value{Limit: &perSecond, Source: sendlimit.SourceManual}, both.PerSecond)
	assert.Equal(t, sendlimit.Value{Limit: &perDay, Source: sendlimit.SourceManual}, both.PerDay)
	assert.Equal(t, sendlimit.Limits{PerSecond: &perSecond, PerDay: &perDay}, both.Limits())

	onlyDay := sendlimit.EffectiveOf(&ent.Integration{MaxPerDay: &perDay})
	assert.False(t, onlyDay.Unlimited(), "one ceiling is enough to be limited")
}

func TestUsageCountsSentMessagesPerIntegrationOverTheLastDay(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := t.Context()

	got, err := sendlimit.Usage(ctx, env.DB.Scoped(fixtures.AcmeID), time.Now())
	require.NoError(t, err)
	assert.Equal(t, map[int64]int{
		fixtures.IntegrationAcmeDefaultID: 2, // the third send is 30 hours old
		fixtures.IntegrationAcmeSesID:     1,
	}, got, "the pending, unstamped and Globex rows do not count")
}
