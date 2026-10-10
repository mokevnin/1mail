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

type namedDispatcher struct {
	names []string
	body  []byte
}

func (d *namedDispatcher) Dispatch(_ context.Context, _ *ent.Scoped, name, _ string, body []byte) error {
	d.names = append(d.names, name)
	d.body = body
	return nil
}

type countingEnroller struct{ calls int }

func (e *countingEnroller) OnEvent(context.Context, int64, int64, string) error {
	e.calls++
	return nil
}

func auditMessage(t *testing.T) *message.Message {
	t.Helper()
	data, err := json.Marshal(&AuditEntry{WorkspaceID: 1, Actor: Actor{Kind: ActorUser}, Action: "membership.update", TargetType: "membership"})
	require.NoError(t, err)
	body, err := json.Marshal(Envelope{ID: "evt", Name: NameAuditEntry, Version: 1, WorkspaceID: 1, Data: data})
	require.NoError(t, err)
	return message.NewMessage(watermill.NewUUID(), body)
}

// The automations consumer skips an Audit entry: an administrative action never
// enrolls anyone.
func TestAutomationsConsumerSkipsAuditEntries(t *testing.T) {
	enroller := &countingEnroller{}
	require.NoError(t, automationsConsumer(enroller)(auditMessage(t)))
	assert.Zero(t, enroller.calls)
}

// The webhooks consumer hands an Audit entry to the dispatcher under its bus type
// (not the entry's action), so an endpoint selects it as "audit.entry" (ADR 0022).
func TestWebhooksConsumerForwardsAuditEntriesAsAuditEntry(t *testing.T) {
	d := &namedDispatcher{}
	require.NoError(t, webhooksConsumer(&ent.Client{}, d)(auditMessage(t)))
	assert.Equal(t, []string{NameAuditEntry}, d.names)

	var payload webhookPayload
	require.NoError(t, json.Unmarshal(d.body, &payload))
	assert.Equal(t, NameAuditEntry, payload.Type)
	assert.JSONEq(t, `{"workspaceId":1,"actor":{"kind":"user"},"action":"membership.update","targetType":"membership"}`, string(payload.Data))
}
