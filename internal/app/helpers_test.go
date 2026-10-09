package app

import (
	"github.com/samber/do/v2"

	"github.com/mokevnin/1mail/ent"
)

// invokeEnt resolves the app's ent client from its container, the same instance the
// operator methods use.
func invokeEnt(a *App) (*ent.Client, error) {
	c, err := do.Invoke[*entClient](a.injector)
	if err != nil {
		return nil, err
	}
	return c.Client, nil
}
