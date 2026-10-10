// Package ee assembles the Enterprise Edition (ADR 0014): everything under ee/ that
// the single binary plugs into core, gated by the runtime license key. The composition
// root (and the test harness) build one Edition and hand its parts to core seams, so
// there is exactly one place that decides what a license switches on.
//
// Governed by ee/LICENSE, not the AGPL.
package ee

import (
	"github.com/mokevnin/1mail/ee/audit"
	"github.com/mokevnin/1mail/ee/licensekey"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/events"
)

// Edition is the Enterprise surface of one running instance.
type Edition struct {
	lic *licensekey.License

	// Audit is the read side of the Audit log; unlicensed, it reports so and the
	// /site page answers 402.
	Audit *audit.Log
	// Consumers are the extra bus subscribers (events.RegisterSubscribers).
	Consumers []events.Consumer
}

// New builds the Edition for a license. Subscribers are always registered and each
// checks the license itself, so a key added later needs no rewiring and an unlicensed
// instance stores nothing.
func New(client *ent.Client, lic *licensekey.License) *Edition {
	return &Edition{
		lic:       lic,
		Audit:     audit.NewLog(lic),
		Consumers: []events.Consumer{audit.Consumer(client, lic)},
	}
}

// Webhooks gates the webhook dispatcher by license: `audit.entry` is forwarded to
// Webhook endpoints only when the Audit feature is licensed.
func (e *Edition) Webhooks(next events.WebhookDispatcher) events.WebhookDispatcher {
	return audit.Forwarder(next, e.lic)
}
