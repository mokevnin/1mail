package pubsub

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type ContactEvent struct {
	ContactID int64  `json:"contactId"`
	Email     string `json:"email"`
}

func Publish[T any](ps *PubSub, topic string, payload T) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), b)
	return ps.Publisher.Publish(topic, msg)
}

func RegisterHandlers(ps *PubSub) {
	ps.Router.AddNoPublisherHandler(
		"log_contact_created",
		TopicContactCreated,
		ps.Subscriber,
		func(msg *message.Message) error {
			var evt ContactEvent
			if err := json.Unmarshal(msg.Payload, &evt); err != nil {
				return err
			}
			fmt.Printf("contact created: id=%d email=%s\n", evt.ContactID, evt.Email)
			return nil
		},
	)
}
