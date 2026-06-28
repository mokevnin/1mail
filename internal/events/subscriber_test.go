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

// msgFor builds a watermill message carrying the given envelope, as the router
// would deliver it.
func msgFor(t *testing.T, env Envelope) *message.Message {
	t.Helper()
	body, err := json.Marshal(env)
	require.NoError(t, err)
	return message.NewMessage(watermill.NewUUID(), body)
}

// The automations consumer enrolls the contact for events that carry one.
func TestAutomationsConsumerEnrollsContact(t *testing.T) {
	enroller := &fakeEnroller{}
	handler := automationsConsumer(enroller)

	require.NoError(t, handler(msgFor(t, Envelope{
		Name: NameEmailOpened, WorkspaceID: 1, ContactID: 7, Subject: "a@b.c",
	})))

	require.Len(t, enroller.calls, 1)
	assert.Equal(t, enrollCall{workspaceID: 1, contactID: 7, action: NameEmailOpened}, enroller.calls[0])
}

// Events without a contact (anonymous collect events) are skipped, not enrolled.
func TestAutomationsConsumerSkipsContactlessEvents(t *testing.T) {
	enroller := &fakeEnroller{}
	handler := automationsConsumer(enroller)

	require.NoError(t, handler(msgFor(t, Envelope{
		Name: "page_view", WorkspaceID: 1, ContactID: 0, Subject: "visitor:x",
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

// The webhooks consumer builds the public payload from the envelope and forwards
// it to the dispatcher with the workspace, event name, and delivery id.
func TestWebhooksConsumerBuildsPayload(t *testing.T) {
	d := &fakeDispatcher{}
	handler := webhooksConsumer(d)

	require.NoError(t, handler(msgFor(t, Envelope{
		ID: "evt_9", Name: NameContactCreated, WorkspaceID: 1, Subject: "a@b.c",
		ContactID: 5, Payload: []byte(`{"email":"a@b.c"}`),
	})))

	require.Len(t, d.calls, 1)
	c := d.calls[0]
	assert.EqualValues(t, 1, c.workspaceID)
	assert.Equal(t, NameContactCreated, c.eventName)
	assert.Equal(t, "evt_9", c.deliveryID)

	var p map[string]any
	require.NoError(t, json.Unmarshal(c.body, &p))
	assert.Equal(t, "evt_9", p["id"])
	assert.Equal(t, NameContactCreated, p["type"])
	assert.Equal(t, "a@b.c", p["subject"])
}
