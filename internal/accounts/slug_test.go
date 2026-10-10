package accounts_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/accounts"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func TestSlugifyTransliteratesAndCollapses(t *testing.T) {
	assert.Equal(t, "privet-mir", accounts.Slugify("Привет Мир"))
	assert.Equal(t, "acme-co", accounts.Slugify("  Acme -- Co!! "))
	assert.Empty(t, accounts.Slugify("!!!"))
}

func TestWorkspaceIDBySlug(t *testing.T) {
	env := testhelper.Setup(t)

	id, err := accounts.WorkspaceIDBySlug(context.Background(), env.DB, fixtures.AcmeSlug)
	require.NoError(t, err)
	assert.EqualValues(t, fixtures.AcmeID, id)
	_, err = accounts.WorkspaceIDBySlug(context.Background(), env.DB, "no-such-workspace")
	require.Error(t, err)
}
