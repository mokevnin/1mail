package site

import (
	"github.com/mokevnin/1mail/ent"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/pubsub"
)

type Handlers struct {
	ent    *ent.Client
	pubsub *pubsub.PubSub
}

func NewHandlers(client *ent.Client, ps *pubsub.PubSub) *Handlers {
	return &Handlers{ent: client, pubsub: ps}
}

var _ siteapi.Handler = (*Handlers)(nil)
