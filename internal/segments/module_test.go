package segments_test

import (
	"context"
	"testing"

	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/segment"
	"github.com/mokevnin/1mail/internal/segments"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	proSegmentID  = int64(100) // fixture: custom:plan = pro
	otherWsID     = int64(2)   // a workspace that does not own segment 100
	badDefinition = `{"rules":[{"field":"email","operator":"weird","value":"x"}]}`
)

func ptr[T any](v T) *T { return &v }

func TestCreateRejectsInvalidDefinitionForAnyType(t *testing.T) {
	env := testhelper.Setup(t)
	m := segments.New(env.DB)
	ctx := context.Background()

	for _, typ := range []segment.Type{segment.TypeRule, segment.TypeSnapshot} {
		before, err := env.DB.Segment.Query().Count(ctx)
		require.NoError(t, err)

		_, err = m.Create(ctx, wsID, segments.CreateInput{Name: "Bad", Type: typ, Definition: ptr(badDefinition)})
		require.ErrorIs(t, err, segments.ErrInvalidDefinition, typ)

		after, err := env.DB.Segment.Query().Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, before, after, "nothing persisted")
	}
}

func TestCreateAcceptsValidAndEmptyDefinition(t *testing.T) {
	env := testhelper.Setup(t)
	m := segments.New(env.DB)
	ctx := context.Background()

	def := `{"combinator":"and","rules":[{"field":"email","operator":"contains","value":"x"}]}`
	s, err := m.Create(ctx, wsID, segments.CreateInput{Name: "Good", Type: segment.TypeRule, Definition: ptr(def)})
	require.NoError(t, err)
	assert.Equal(t, wsID, s.WorkspaceID)
	assert.Equal(t, def, *s.Definition)

	_, err = m.Create(ctx, wsID, segments.CreateInput{Name: "Empty", Type: segment.TypeRule, Definition: ptr("")})
	require.NoError(t, err)
	_, err = m.Create(ctx, wsID, segments.CreateInput{Name: "Nil", Type: segment.TypeRule})
	require.NoError(t, err)
}

func TestUpdateValidatesLikeCreate(t *testing.T) {
	env := testhelper.Setup(t)
	m := segments.New(env.DB)
	ctx := context.Background()

	_, err := m.Update(ctx, wsID, proSegmentID, segments.UpdateInput{Definition: ptr(badDefinition)})
	require.ErrorIs(t, err, segments.ErrInvalidDefinition)

	// An invalid definition is rejected whatever the type, and leaves the row untouched.
	_, err = m.Update(ctx, wsID, proSegmentID, segments.UpdateInput{Type: ptr(segment.TypeSnapshot), Definition: ptr(badDefinition)})
	require.ErrorIs(t, err, segments.ErrInvalidDefinition)
	s, err := env.DB.Segment.Get(ctx, proSegmentID)
	require.NoError(t, err)
	assert.Equal(t, segment.TypeRule, s.Type)
	assert.NotEqual(t, badDefinition, *s.Definition)

	got, err := m.Update(ctx, wsID, proSegmentID, segments.UpdateInput{Name: ptr("Renamed")})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.Name)
}

func TestUpdateIsWorkspaceScoped(t *testing.T) {
	env := testhelper.Setup(t)
	m := segments.New(env.DB)

	_, err := m.Update(context.Background(), otherWsID, proSegmentID, segments.UpdateInput{Name: ptr("x")})
	require.ErrorIs(t, err, segments.ErrNotFound)
}

func TestPreviewAndCountAgree(t *testing.T) {
	env := testhelper.Setup(t)
	m := segments.New(env.DB)
	ctx := context.Background()

	stored, err := env.DB.Segment.Get(ctx, proSegmentID)
	require.NoError(t, err)
	pred, err := segments.ContactPredicate(*stored.Definition)
	require.NoError(t, err)
	want, err := env.DB.Contact.Query().Where(contact.WorkspaceID(wsID), pred).Count(ctx)
	require.NoError(t, err)
	require.Positive(t, want, "fixture segment should match someone")

	preview, err := m.Preview(ctx, wsID, *stored.Definition)
	require.NoError(t, err)
	assert.Equal(t, want, preview)

	count, err := m.Count(ctx, wsID, proSegmentID)
	require.NoError(t, err)
	assert.Equal(t, want, count)
}

func TestPreviewAndCountErrors(t *testing.T) {
	env := testhelper.Setup(t)
	m := segments.New(env.DB)
	ctx := context.Background()

	_, err := m.Preview(ctx, wsID, badDefinition)
	require.ErrorIs(t, err, segments.ErrInvalidDefinition)

	_, err = m.Count(ctx, otherWsID, proSegmentID)
	require.ErrorIs(t, err, segments.ErrNotFound)
}
