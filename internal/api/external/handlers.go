package external

import (
	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
)

type Handlers struct {
	ent            *ent.Client
	bootstrapToken string
}

func NewHandlers(client *ent.Client, bootstrapToken string) *Handlers {
	return &Handlers{ent: client, bootstrapToken: bootstrapToken}
}

var _ externalapi.StrictServerInterface = (*Handlers)(nil)
