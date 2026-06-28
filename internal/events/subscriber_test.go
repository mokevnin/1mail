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
