package pubsub

import (
	"database/sql"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
)

const (
	TopicContactCreated = "contact.created"
	TopicContactUpdated = "contact.updated"
	TopicContactDeleted = "contact.deleted"
)

type PubSub struct {
	Publisher  *watermillsql.Publisher
	Subscriber *watermillsql.Subscriber
	Router     *message.Router
}

func New(db *sql.DB) (*PubSub, error) {
	logger := watermill.NewStdLogger(false, false)

	publisher, err := watermillsql.NewPublisher(db, watermillsql.PublisherConfig{
		SchemaAdapter:        watermillsql.DefaultPostgreSQLSchema{},
		AutoInitializeSchema: true,
	}, logger)
	if err != nil {
		return nil, err
	}

	subscriber, err := watermillsql.NewSubscriber(db, watermillsql.SubscriberConfig{
		SchemaAdapter:    watermillsql.DefaultPostgreSQLSchema{},
		OffsetsAdapter:   watermillsql.DefaultPostgreSQLOffsetsAdapter{},
		InitializeSchema: true,
	}, logger)
	if err != nil {
		return nil, err
	}

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, err
	}

	return &PubSub{
		Publisher:  publisher,
		Subscriber: subscriber,
		Router:     router,
	}, nil
}

func (ps *PubSub) Close() {
	_ = ps.Publisher.Close()
	_ = ps.Subscriber.Close()
	_ = ps.Router.Close()
}
