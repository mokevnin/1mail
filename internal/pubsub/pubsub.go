package pubsub

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	TopicContactCreated = "contact.created"
	TopicContactUpdated = "contact.updated"
	TopicContactDeleted = "contact.deleted"
	TopicUserRegistered = "user.registered"
)

type PubSub struct {
	Publisher  *watermillsql.Publisher
	Subscriber *watermillsql.Subscriber
	Router     *message.Router
}

// safeTableName turns a topic into a dot-free physical table name. Topics use a
// dotted convention (e.g. "user.registered"), but watermill embeds the topic
// verbatim into a quoted table identifier ("watermill_user.registered"). The dot
// breaks tools that don't quote the identifier — notably testfixtures, which then
// emits public.watermill_user.registered (a Postgres cross-database reference).
func safeTableName(prefix, topic string) string {
	return fmt.Sprintf(`"%s%s"`, prefix, strings.ReplaceAll(topic, ".", "_"))
}

func New(db *sql.DB) (*PubSub, error) {
	logger := watermill.NewStdLogger(false, false)

	// One shared schema adapter so the publisher and subscriber resolve the same
	// table name — a mismatch would silently stop messages from being consumed.
	schemaAdapter := watermillsql.DefaultPostgreSQLSchema{
		GenerateMessagesTableName: func(topic string) string {
			return safeTableName("watermill_", topic)
		},
	}
	offsetsAdapter := watermillsql.DefaultPostgreSQLOffsetsAdapter{
		GenerateMessagesOffsetsTableName: func(topic string) string {
			return safeTableName("watermill_offsets_", topic)
		},
	}

	publisher, err := watermillsql.NewPublisher(db, watermillsql.PublisherConfig{
		SchemaAdapter:        schemaAdapter,
		AutoInitializeSchema: true,
	}, logger)
	if err != nil {
		return nil, err
	}

	subscriber, err := watermillsql.NewSubscriber(db, watermillsql.SubscriberConfig{
		SchemaAdapter:    schemaAdapter,
		OffsetsAdapter:   offsetsAdapter,
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
