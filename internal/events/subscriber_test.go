package events

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type enrollCall struct {
	workspaceID, contactID int64
	action                 string
}

type fakeEnroller struct{ calls []enrollCall }

func (f *fakeEnroller) OnEvent(_ context.Context, workspaceID, contactID int64, action string) error {
	f.calls = append(f.calls, enrollCall{workspaceID, contactID, action})
	return nil
}

// msgFor builds the watermill message the router would deliver for a typed event.
func msgFor(t *testing.T, ev DomainEvent) *message.Message {
	t.Helper()
	data, err := json.Marshal(ev)
	require.NoError(t, err)
	env := Envelope{ID: "evt_test", Name: ev.EventName(), Version: ev.EventVersion(), WorkspaceID: ev.Workspace(), Data: data}
	body, err := json.Marshal(env)
	require.NoError(t, err)
	return message.NewMessage(watermill.NewUUID(), body)
}

// The automations consumer enrolls the contact on the event's semantic action.
func TestAutomationsConsumerEnrollsContact(t *testing.T) {
	enroller := &fakeEnroller{}
	handler := automationsConsumer(enroller)

	require.NoError(t, handler(msgFor(t, &EmailEngagement{
		Action: NameEmailOpened, WorkspaceID: 1, ContactID: 7, Email: "a@b.c",
	})))

	require.Len(t, enroller.calls, 1)
	assert.Equal(t, enrollCall{workspaceID: 1, contactID: 7, action: NameEmailOpened}, enroller.calls[0])
}

// Collected (customer) events carry no contact, so they are not enrolled.
func TestAutomationsConsumerSkipsContactlessEvents(t *testing.T) {
	enroller := &fakeEnroller{}
	handler := automationsConsumer(enroller)

	require.NoError(t, handler(msgFor(t, &CollectedEvent{
		WorkspaceID: 1, SubjectID: "visitor:x", Action: "page_view",
	})))

	assert.Empty(t, enroller.calls)
}

type dispatchCall struct {
	workspaceID           int64
	eventName, deliveryID string
	body                  []byte
}

type fakeDispatcher struct{ calls []dispatchCall }

func (f *fakeDispatcher) Dispatch(_ context.Context, ws int64, name, delivery string, body []byte) error {
	f.calls = append(f.calls, dispatchCall{ws, name, delivery, body})
	return nil
}

// The webhooks consumer dispatches with the semantic action and a payload built
// from the event's projection.
func TestWebhooksConsumerBuildsPayload(t *testing.T) {
	d := &fakeDispatcher{}
	handler := webhooksConsumer(d)

	require.NoError(t, handler(msgFor(t, &ContactCreated{WorkspaceID: 1, ContactID: 5, Email: "a@b.c"})))

	require.Len(t, d.calls, 1)
	c := d.calls[0]
	assert.EqualValues(t, 1, c.workspaceID)
	assert.Equal(t, NameContactCreated, c.eventName)

	var p map[string]any
	require.NoError(t, json.Unmarshal(c.body, &p))
	assert.Equal(t, NameContactCreated, p["type"])
	assert.Equal(t, "a@b.c", p["subject"])
}
