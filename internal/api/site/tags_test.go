package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func siteTagNames(items []siteapi.SiteTagResource) []string {
	return lo.Map(items, func(t siteapi.SiteTagResource, _ int) string { return t.Name })
}

func TestSiteTagsRequireAuth(t *testing.T) {
	env := testhelper.Setup(t)
	c, err := siteapi.NewClient("http://local/site", noJWT{}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)

	_, err = c.SiteTagsList(context.Background(), siteapi.SiteTagsListParams{Slug: "acme"})
	require.Error(t, err)
}

// Fixtures: workspace "acme" has tags vip (contacts 1, 3), newsletter (contact 1)
// and unused; contact 2 has none.
func TestSiteTagsListApplyRemove(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	all, err := c.SiteTagsList(ctx, siteapi.SiteTagsListParams{Slug: "acme"})
	require.NoError(t, err)
	listed, ok := all.(*siteapi.SiteTagsListOK)
	require.Truef(t, ok, "got %T", all)
	assert.Equal(t, []string{"newsletter", "unused", "vip"}, siteTagNames(listed.Items))

	mine, err := c.SiteTagsListForContact(ctx, siteapi.SiteTagsListForContactParams{Slug: "acme", ContactId: "1"})
	require.NoError(t, err)
	assert.Equal(t, []string{"newsletter", "vip"}, siteTagNames(mine.(*siteapi.SiteTagsListForContactOK).Items))

	applied, err := c.SiteTagsApply(ctx, &siteapi.SiteApplyTagInput{Name: "churn-risk"}, siteapi.SiteTagsApplyParams{Slug: "acme", ContactId: "2"})
	require.NoError(t, err)
	tag, ok := applied.(*siteapi.SiteTagResource)
	require.Truef(t, ok, "got %T", applied)
	assert.Equal(t, "churn-risk", tag.Name)

	mine, err = c.SiteTagsListForContact(ctx, siteapi.SiteTagsListForContactParams{Slug: "acme", ContactId: "2"})
	require.NoError(t, err)
	assert.Equal(t, []string{"churn-risk"}, siteTagNames(mine.(*siteapi.SiteTagsListForContactOK).Items))

	removed, err := c.SiteTagsRemove(ctx, siteapi.SiteTagsRemoveParams{Slug: "acme", ContactId: "2", Name: "churn-risk"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteTagsRemoveNoContent{}, removed)
	mine, err = c.SiteTagsListForContact(ctx, siteapi.SiteTagsListForContactParams{Slug: "acme", ContactId: "2"})
	require.NoError(t, err)
	assert.Empty(t, mine.(*siteapi.SiteTagsListForContactOK).Items)

	// Errors: unknown contact, blank name, unknown workspace.
	nf, err := c.SiteTagsApply(ctx, &siteapi.SiteApplyTagInput{Name: "x"}, siteapi.SiteTagsApplyParams{Slug: "acme", ContactId: "999999"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteTagsApplyNotFound{}, nf)
	blank, err := c.SiteTagsApply(ctx, &siteapi.SiteApplyTagInput{Name: " "}, siteapi.SiteTagsApplyParams{Slug: "acme", ContactId: "2"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteTagsApplyUnprocessableEntity{}, blank)
	ws, err := c.SiteTagsList(ctx, siteapi.SiteTagsListParams{Slug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteTagsListNotFound{}, ws)
}
