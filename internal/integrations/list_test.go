package integrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/integrations"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func TestListPagesTheWorkspaceIntegrationsAscendingByID(t *testing.T) {
	env := testhelper.Setup(t)
	m := new(integrations.Module)

	first, err := m.List(context.Background(), env.DB.Scoped(fixtures.AcmeID), pagination.Params{Page: 1, PageSize: 1})
	require.NoError(t, err)
	require.Len(t, first.Items, 1)
	assert.Equal(t, int64(fixtures.IntegrationAcmeDefaultID), first.Items[0].ID)
	assert.Equal(t, 2, first.TotalItems)
	assert.Equal(t, 2, first.TotalPages)

	second, err := m.List(context.Background(), env.DB.Scoped(fixtures.AcmeID), pagination.Params{Page: 2, PageSize: 1})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	assert.Equal(t, int64(fixtures.IntegrationAcmeSesID), second.Items[0].ID)
}
