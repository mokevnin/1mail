package tags_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/tag"
	"github.com/mokevnin/1mail/internal/tags"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixtures (workspace 1): tag 1 "vip" on contacts 1 and 3; tag 2 "newsletter" on
// contact 1; tag 3 "unused" on nobody. Contact 2 has no tags.
const (
	wsID      = int64(1)
	otherWsID = testhelper.GlobexWorkspaceID
)

func names(ts []*ent.Tag) []string {
	return lo.Map(ts, func(t *ent.Tag, _ int) string { return t.Name })
}

func TestListReturnsTheWorkspaceCatalogue(t *testing.T) {
	env := testhelper.Setup(t)
	m := tags.New(env.DB)

	got, err := m.List(context.Background(), wsID)
	require.NoError(t, err)
	assert.Equal(t, []string{"newsletter", "unused", "vip"}, names(got))

	other, err := m.List(context.Background(), otherWsID)
	require.NoError(t, err)
	assert.Equal(t, []string{testhelper.GlobexTagName}, names(other), "each workspace sees only its own tags")
}

func TestForContactListsOnlyItsTags(t *testing.T) {
	env := testhelper.Setup(t)
	m := tags.New(env.DB)
	ctx := context.Background()

	got, err := m.ForContact(ctx, wsID, 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"newsletter", "vip"}, names(got))

	none, err := m.ForContact(ctx, wsID, 2)
	require.NoError(t, err)
	assert.Empty(t, none)

	_, err = m.ForContact(ctx, wsID, 999999)
	require.ErrorIs(t, err, tags.ErrContactNotFound)
	_, err = m.ForContact(ctx, otherWsID, 1)
	require.ErrorIs(t, err, tags.ErrContactNotFound, "a contact of another workspace is invisible")
}

func TestApplyAutoCreatesOnFirstUseAndIsIdempotent(t *testing.T) {
	env := testhelper.Setup(t)
	m := tags.New(env.DB)
	ctx := context.Background()

	before, err := env.DB.Tag.Query().Count(ctx)
	require.NoError(t, err)

	got, err := m.Apply(ctx, wsID, 2, "  churn-risk ")
	require.NoError(t, err)
	assert.Equal(t, "churn-risk", got.Name, "names are trimmed")

	again, err := m.Apply(ctx, wsID, 2, "churn-risk")
	require.NoError(t, err)
	assert.Equal(t, got.ID, again.ID, "applying twice reuses the one Tag")

	after, err := env.DB.Tag.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, before+1, after)

	has, err := m.ForContact(ctx, wsID, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"churn-risk"}, names(has))

	// An existing Tag is reused, not duplicated.
	vip, err := m.Apply(ctx, wsID, 2, "vip")
	require.NoError(t, err)
	assert.Equal(t, int64(1), vip.ID)
}

func TestApplyRejectsBlankNameAndUnknownContact(t *testing.T) {
	env := testhelper.Setup(t)
	m := tags.New(env.DB)
	ctx := context.Background()

	_, err := m.Apply(ctx, wsID, 1, "   ")
	require.ErrorIs(t, err, tags.ErrInvalidName)

	_, err = m.Apply(ctx, wsID, 999999, "x")
	require.ErrorIs(t, err, tags.ErrContactNotFound)
	_, err = m.Apply(ctx, otherWsID, 1, "x")
	require.ErrorIs(t, err, tags.ErrContactNotFound)

	n, err := env.DB.Tag.Query().Where(tag.Name("x")).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "a failed apply creates no Tag")
}

func TestRemoveDetachesAndKeepsTheTag(t *testing.T) {
	env := testhelper.Setup(t)
	m := tags.New(env.DB)
	ctx := context.Background()

	require.NoError(t, m.Remove(ctx, wsID, 1, "vip"))
	got, err := m.ForContact(ctx, wsID, 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"newsletter"}, names(got))

	// The Tag stays in the catalogue, and contact 3 keeps it.
	exists, err := env.DB.Tag.Query().Where(tag.Name("vip")).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, exists)
	c3, err := m.ForContact(ctx, wsID, 3)
	require.NoError(t, err)
	assert.Equal(t, []string{"vip"}, names(c3))

	// Idempotent: removing an absent or unknown tag succeeds.
	require.NoError(t, m.Remove(ctx, wsID, 1, "vip"))
	require.NoError(t, m.Remove(ctx, wsID, 1, "never-existed"))

	require.ErrorIs(t, m.Remove(ctx, wsID, 999999, "vip"), tags.ErrContactNotFound)
	require.ErrorIs(t, m.Remove(ctx, otherWsID, 3, "vip"), tags.ErrContactNotFound)
}
