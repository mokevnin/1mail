package external_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// The same edit through /api is audited under the token, unlike ingest.
func TestExternalTagCreateIsAuditedUnderTheToken(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.ExternalAnchor(t)

	res, err := c.TagsApply(context.Background(), &externalapi.ApplyTagInput{Name: "api-tag"},
		externalapi.TagsApplyParams{ContactId: entityIDString(fixtures.ContactBobID)})
	require.NoError(t, err)
	require.IsType(t, &externalapi.TagResource{}, res)

	got := env.OutboxEvents(t, events.NameAuditEntry)
	require.Len(t, got, 1)
	e := got[0].(*events.AuditEntry)
	assert.Equal(t, "tag.create", e.Action)
	assert.Equal(t, events.Actor{Kind: events.ActorAPIToken, ID: strconv.Itoa(fixtures.AnchorTokenID), Name: fixtures.AnchorTokenName}, e.Actor)
	assert.Equal(t, "api-tag", e.TargetName)
}
