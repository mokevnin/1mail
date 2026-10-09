package external_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/mokevnin/1mail/ent/segment"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/segments"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixture segment 1 ("Active subscribers", rule) is read by id; the list is
// scoped to the token's workspace.
func TestExternalSegmentsRead(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	c := client(t, env, seedToken(t, env.DB, []string{"segments:read"}))

	list, err := c.SegmentsList(ctx, externalapi.SegmentsListParams{})
	require.NoError(t, err)
	ok, isOK := list.(*externalapi.SegmentsListOK)
	require.Truef(t, isOK, "got %T", list)
	total, err := env.DB.Segment.Query().Where(segment.WorkspaceID(1)).Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, total, ok.TotalItems)

	got, err := c.SegmentsGet(ctx, externalapi.SegmentsGetParams{ID: "1"})
	require.NoError(t, err)
	seg, isSeg := got.(*externalapi.SegmentResource)
	require.Truef(t, isSeg, "got %T", got)
	assert.Equal(t, "Active subscribers", seg.Name)
	assert.Equal(t, externalapi.SegmentTypeRule, seg.Type)

	missing, err := c.SegmentsGet(ctx, externalapi.SegmentsGetParams{ID: "999999"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsGetNotFound{}, missing)
}

// Create and update go through the segments module: a definition that does not fit
// the Contact schema is a 422, and a valid one is stored.
func TestExternalSegmentsWrite(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	c := client(t, env, seedToken(t, env.DB, []string{"segments:read", "segments:write"}))

	bad, err := c.SegmentsCreate(ctx, &externalapi.CreateSegmentInput{
		Name: "Bad", Type: externalapi.SegmentTypeRule,
		Definition: externalapi.NewOptString(`{"combinator":"and","rules":[{"field":"nope","operator":"=","value":"x"}]}`),
	})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsCreateUnprocessableEntity{}, bad)

	const def = `{"combinator":"and","rules":[{"field":"custom:plan","operator":"=","value":"pro"}]}`
	created, err := c.SegmentsCreate(ctx, &externalapi.CreateSegmentInput{
		Name: "Pros", Type: externalapi.SegmentTypeRule, Definition: externalapi.NewOptString(def),
	})
	require.NoError(t, err)
	seg, isSeg := created.(*externalapi.SegmentResource)
	require.Truef(t, isSeg, "got %T", created)
	assert.Equal(t, "Pros", seg.Name)

	upd, err := c.SegmentsUpdate(ctx, &externalapi.UpdateSegmentInput{Name: externalapi.NewOptString("Pros 2")},
		externalapi.SegmentsUpdateParams{ID: seg.ID})
	require.NoError(t, err)
	assert.Equal(t, "Pros 2", upd.(*externalapi.SegmentResource).Name)

	badUpd, err := c.SegmentsUpdate(ctx, &externalapi.UpdateSegmentInput{Definition: externalapi.NewOptString(`{`)},
		externalapi.SegmentsUpdateParams{ID: seg.ID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsUpdateUnprocessableEntity{}, badUpd)

	del, err := c.SegmentsDelete(ctx, externalapi.SegmentsDeleteParams{ID: seg.ID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsDeleteNoContent{}, del)
	again, err := c.SegmentsDelete(ctx, externalapi.SegmentsDeleteParams{ID: seg.ID})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsDeleteNotFound{}, again)
}

// Preview matches the segments module's count for the same definition.
func TestExternalSegmentsPreview(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	c := client(t, env, seedToken(t, env.DB, []string{"segments:read"}))

	const def = `{"combinator":"and","rules":[{"field":"custom:plan","operator":"=","value":"pro"}]}`
	want, err := segments.New(env.DB).Preview(ctx, 1, def)
	require.NoError(t, err)
	res, err := c.SegmentsPreview(ctx, &externalapi.PreviewSegmentInput{Definition: externalapi.NewOptNilString(def)})
	require.NoError(t, err)
	ok, isOK := res.(*externalapi.PreviewSegmentResult)
	require.Truef(t, isOK, "got %T", res)
	assert.EqualValues(t, want, ok.Count)

	bad, err := c.SegmentsPreview(ctx, &externalapi.PreviewSegmentInput{Definition: externalapi.NewOptNilString(`{`)})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsPreviewUnprocessableEntity{}, bad)
}

func TestExternalSegmentsScopes(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	none := client(t, env, seedToken(t, env.DB, []string{"contacts:read"}))
	list, err := none.SegmentsList(ctx, externalapi.SegmentsListParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsListUnauthorized{}, list)
	prev, err := none.SegmentsPreview(ctx, &externalapi.PreviewSegmentInput{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsPreviewUnauthorized{}, prev)

	// Read scope cannot write.
	ro := client(t, env, seedToken(t, env.DB, []string{"segments:read"}))
	cr, err := ro.SegmentsCreate(ctx, &externalapi.CreateSegmentInput{Name: "x", Type: externalapi.SegmentTypeSnapshot})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsCreateUnauthorized{}, cr)
	up, err := ro.SegmentsUpdate(ctx, &externalapi.UpdateSegmentInput{}, externalapi.SegmentsUpdateParams{ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsUpdateUnauthorized{}, up)
	del, err := ro.SegmentsDelete(ctx, externalapi.SegmentsDeleteParams{ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsDeleteUnauthorized{}, del)

	// Write scope cannot read.
	wo := client(t, env, seedToken(t, env.DB, []string{"segments:write"}))
	get, err := wo.SegmentsGet(ctx, externalapi.SegmentsGetParams{ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsGetUnauthorized{}, get)
}

// Another tenant's segment is invisible and immutable through this token.
func TestExternalSegmentsWorkspaceIsolation(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	ws2, err := env.DB.Workspace.Create().
		SetName("Globex").SetSlug("globex").SetCollectKey("globex-collect-key").
		SetIngestKey("globex-ingest-key").Save(ctx)
	require.NoError(t, err)
	other, err := env.DB.Segment.Create().SetWorkspaceID(ws2.ID).SetName("Globex only").
		SetType(segment.TypeSnapshot).Save(ctx)
	require.NoError(t, err)
	id := externalapi.EntityId(strconv.FormatInt(other.ID, 10))

	c := client(t, env, seedToken(t, env.DB, []string{"segments:read", "segments:write"}))

	list, err := c.SegmentsList(ctx, externalapi.SegmentsListParams{})
	require.NoError(t, err)
	for _, s := range list.(*externalapi.SegmentsListOK).Items {
		assert.NotEqual(t, "Globex only", s.Name, "another tenant's segment leaked")
	}
	get, err := c.SegmentsGet(ctx, externalapi.SegmentsGetParams{ID: id})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsGetNotFound{}, get)
	upd, err := c.SegmentsUpdate(ctx, &externalapi.UpdateSegmentInput{Name: externalapi.NewOptString("pwned")},
		externalapi.SegmentsUpdateParams{ID: id})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsUpdateNotFound{}, upd)
	del, err := c.SegmentsDelete(ctx, externalapi.SegmentsDeleteParams{ID: id})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.SegmentsDeleteNotFound{}, del)
}
