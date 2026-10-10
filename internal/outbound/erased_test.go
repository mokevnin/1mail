package outbound_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/outboundmessage"
	"github.com/mokevnin/1mail/internal/erasure"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/outbound"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// A send queued before its Contact was erased must not leave (ADR 0021): the Request
// still carries the Contact as it was loaded, so the chokepoint re-checks that it
// exists, and records nothing that would keep the erased person's address.
func TestSendToAnErasedContactIsDroppedAtTheChokepoint(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	s := env.DB.Scoped(fixtures.AcmeID)
	req := marketing(t, env, "bc:erased", fixtures.ContactAliceID)

	require.NoError(t, erasure.New(env.Bus).Erase(ctx, s, erasure.ByContactID(fixtures.ContactAliceID),
		erasure.Operator{Kind: erasure.OperatorAPIToken}))
	before := len(env.CustomerMail.Messages())

	res, err := newModule(env).Send(ctx, s, req)
	require.NoError(t, err)
	assert.Equal(t, outbound.Skipped, res.Outcome)
	assert.Equal(t, outbound.SkipContactErased, res.Reason)
	assert.Len(t, env.CustomerMail.Messages(), before, "nothing reached the provider")
	n, err := env.DB.OutboundMessage.Query().Where(outboundmessage.IdempotencyKey("bc:erased")).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "no claim or record keeps the address")
}
