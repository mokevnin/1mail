package events

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent"
)

type countingDispatcher struct{ calls int }

func (d *countingDispatcher) Dispatch(context.Context, *ent.Scoped, string, string, []byte) error {
	d.calls++
	return nil
}

type countingEnroller struct{ calls int }

func (e *countingEnroller) OnEvent(context.Context, int64, int64, string) error {
	e.calls++
	return nil
}

// The webhooks and automations consumers skip an Audit entry: it is never fanned out
// as an Event to a Webhook endpoint (forwarding is explicit, ADR 0022) and never
// enrolls anyone.
func TestWebhooksAndAutomationsConsumersSkipAuditEntries(t *testing.T) {
	data, err := json.Marshal(&AuditEntry{WorkspaceID: 1, Actor: Actor{Kind: ActorUser}, Action: "membership.update", TargetType: "membership"})
	require.NoError(t, err)
	body, err := json.Marshal(Envelope{ID: "evt", Name: NameAuditEntry, Version: 1, WorkspaceID: 1, Data: data})
	require.NoError(t, err)
	msg := message.NewMessage(watermill.NewUUID(), body)

	dispatcher, enroller := &countingDispatcher{}, &countingEnroller{}
	require.NoError(t, webhooksConsumer(&ent.Client{}, dispatcher)(msg))
	require.NoError(t, automationsConsumer(enroller)(msg))
	assert.Zero(t, dispatcher.calls)
	assert.Zero(t, enroller.calls)
}
