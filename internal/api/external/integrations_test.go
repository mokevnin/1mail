package external_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/integration"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func TestExternalIntegrationsListReportsSendLimitAndUsage(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	denied, err := env.ExternalScoped(t, "contacts:read").IntegrationsList(ctx, externalapi.IntegrationsListParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.IntegrationsListUnauthorized{}, denied, "integrations:read is required")

	env.DB.Integration.UpdateOneID(fixtures.IntegrationAcmeDefaultID).SetMaxPerSecond(14).SetMaxPerDay(50000).ExecX(ctx)

	res, err := env.ExternalScoped(t, "integrations:read").IntegrationsList(ctx, externalapi.IntegrationsListParams{})
	require.NoError(t, err)
	list, ok := res.(*externalapi.IntegrationsListOK)
	require.Truef(t, ok, "got %T", res)
	require.Len(t, list.Items, 2, "Acme's two Integrations; Globex's is not visible")

	def := list.Items[0]
	assert.Equal(t, fixtures.IntegrationAcmeDefaultName, def.Name)
	assert.True(t, def.IsDefault)
	assert.Equal(t, int32(14), def.SendLimit.PerSecond.Limit.Or(0))
	assert.Equal(t, externalapi.SendLimitSourceManual, def.SendLimit.PerSecond.Source.Or(""))
	assert.Equal(t, int32(50000), def.SendLimit.PerDay.Limit.Or(0))
	assert.Equal(t, int32(2), def.SendLimit.SentLast24h, "two sends in the last 24 hours; the third is older")
	assert.Empty(t, def.SendLimit.Warnings)

	ses := list.Items[1]
	assert.True(t, ses.SendLimit.PerSecond.Limit.Null)
	assert.True(t, ses.SendLimit.PerSecond.Source.Null)
	assert.Equal(t, []externalapi.SendLimitWarning{externalapi.SendLimitWarningUnlimited}, ses.SendLimit.Warnings)
	assert.Equal(t, int32(1), ses.SendLimit.SentLast24h)
}

func TestExternalIntegrationsListIsPaginatedAndWorkspaceScoped(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	c := env.ExternalScoped(t, "integrations:read")

	res, err := c.IntegrationsList(ctx, externalapi.IntegrationsListParams{PageSize: externalapi.NewOptInt32(1)})
	require.NoError(t, err)
	list := res.(*externalapi.IntegrationsListOK)
	assert.Len(t, list.Items, 1)
	assert.Equal(t, int32(2), list.TotalItems)
	assert.Equal(t, int32(2), list.TotalPages)

	all, err := c.IntegrationsList(ctx, externalapi.IntegrationsListParams{})
	require.NoError(t, err)
	n, err := env.DB.Scoped(fixtures.AcmeID).Integration().Query().Where(integration.ProviderIn(integration.ProviderSMTP, integration.ProviderSes)).Count(ctx)
	require.NoError(t, err)
	assert.Len(t, all.(*externalapi.IntegrationsListOK).Items, n)
	for _, it := range all.(*externalapi.IntegrationsListOK).Items {
		assert.NotEqual(t, fixtures.IntegrationGlobexName, it.Name)
	}
}
