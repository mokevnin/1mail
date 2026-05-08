package pubsub

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/mokevnin/1mail/internal/email"
)

type ContactEvent struct {
	ContactID int64  `json:"contactId"`
	Email     string `json:"email"`
}

type UserRegisteredEvent struct {
	UserID int64  `json:"userId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

func Publish[T any](ps *PubSub, topic string, payload T) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), b)
	return ps.Publisher.Publish(topic, msg)
}

func RegisterHandlers(ps *PubSub, emailSender *email.Sender) {
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

	ps.Router.AddNoPublisherHandler(
		"send_welcome_email",
		TopicUserRegistered,
		ps.Subscriber,
		func(msg *message.Message) error {
			var evt UserRegisteredEvent
			if err := json.Unmarshal(msg.Payload, &evt); err != nil {
				return err
			}
			body := fmt.Sprintf("Hi %s,\n\nWelcome to 1mail! Your account has been created.\n", evt.Name)
			if err := emailSender.Send(evt.Email, "Welcome to 1mail!", body); err != nil {
				log.Printf("send welcome email to %s: %v", evt.Email, err)
			}
			return nil
		},
	)
}
