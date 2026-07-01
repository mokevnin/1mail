package site_test

import (
	"context"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Site segments require a valid JWT cookie, like every other site resource.
func TestSiteSegmentsRequireAuth(t *testing.T) {
	env := testhelper.Setup(t)
	c, err := siteapi.NewClient("http://local/site", noJWT{}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)

	_, err = c.SiteSegmentsList(context.Background(), siteapi.SiteSegmentsListParams{})
	require.Error(t, err)
}

// Fixture workspace "acme" (id 1) owns two seeded segments. Listing, creating,
// reading, updating and deleting are all scoped to the authenticated user's
// workspace; an unknown slug resolves to 404 instead of leaking data.
func TestSiteSegmentsScopedToWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	// A seeded segment of the owned workspace is fetchable by id (selection by key).
	seeded, err := c.SiteSegmentsGet(ctx, siteapi.SiteSegmentsGetParams{Slug: "acme", ID: "1"})
	require.NoError(t, err)
	seededRes, ok := seeded.(*siteapi.SiteSegmentResource)
	require.Truef(t, ok, "got %T", seeded)
	assert.Equal(t, "Active subscribers", seededRes.Name)

	// Create scopes the segment to the workspace and returns the resource.
	created, err := c.SiteSegmentsCreate(ctx, &siteapi.SiteCreateSegmentInput{
		Name:       "VIP customers",
		Type:       siteapi.SiteSegmentTypeRule,
		Definition: siteapi.NewOptNilString(`{"combinator":"and","rules":[{"field":"custom:plan","operator":"=","value":"vip"}]}`),
	}, siteapi.SiteSegmentsCreateParams{Slug: "acme"})
	require.NoError(t, err)
	res, ok := created.(*siteapi.SiteSegmentResource)
	require.Truef(t, ok, "got %T", created)
	assert.Equal(t, "VIP customers", res.Name)
	assert.Equal(t, siteapi.SiteSegmentTypeRule, res.Type)

	// The new segment is readable by id.
	got, err := c.SiteSegmentsGet(ctx, siteapi.SiteSegmentsGetParams{Slug: "acme", ID: res.ID})
	require.NoError(t, err)
	gotRes, ok := got.(*siteapi.SiteSegmentResource)
	require.Truef(t, ok, "got %T", got)
	assert.Equal(t, res.ID, gotRes.ID)

	// Update changes mutable fields.
	updated, err := c.SiteSegmentsUpdate(ctx, &siteapi.SiteUpdateSegmentInput{
		Name: siteapi.NewOptString("VIP renamed"),
		Type: siteapi.NewOptSiteSegmentType(siteapi.SiteSegmentTypeSnapshot),
	}, siteapi.SiteSegmentsUpdateParams{Slug: "acme", ID: res.ID})
	require.NoError(t, err)
	updRes, ok := updated.(*siteapi.SiteSegmentResource)
	require.Truef(t, ok, "got %T", updated)
	assert.Equal(t, "VIP renamed", updRes.Name)
	assert.Equal(t, siteapi.SiteSegmentTypeSnapshot, updRes.Type)

	// Delete removes it.
	del, err := c.SiteSegmentsDelete(ctx, siteapi.SiteSegmentsDeleteParams{Slug: "acme", ID: res.ID})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSegmentsDeleteNoContent{}, del)

	// An unknown / non-owned workspace slug resolves to 404, not a data leak.
	missing, err := c.SiteSegmentsList(ctx, siteapi.SiteSegmentsListParams{Slug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteSegmentsListNotFound{}, missing)
}
