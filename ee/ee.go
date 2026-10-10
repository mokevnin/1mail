// Package ee assembles the Enterprise Edition (ADR 0014): everything under ee/ that
// the single binary plugs into core, gated by the runtime license key. The composition
// root (and the test harness) build one Edition and hand its parts to core seams, so
// there is exactly one place that decides what a license switches on.
//
// Governed by ee/LICENSE, not the AGPL.
package ee

import (
	"net/http"

	"github.com/riverqueue/river"

	"github.com/mokevnin/sphericon/ee/audit"
	"github.com/mokevnin/sphericon/ee/licensekey"
	"github.com/mokevnin/sphericon/ee/operator"
	"github.com/mokevnin/sphericon/ee/retention"
	"github.com/mokevnin/sphericon/ent"
	operatorapi "github.com/mokevnin/sphericon/gen/operator"
	"github.com/mokevnin/sphericon/internal/events"
	"github.com/mokevnin/sphericon/internal/jobs"
	"github.com/mokevnin/sphericon/internal/secrets"
)

// Edition is the Enterprise surface of one running instance.
type Edition struct {
	lic    *licensekey.License
	client *ent.Client

	// Audit is the read side of the Audit log; unlicensed, it reports so and the
	// /site page answers 402.
	Audit *audit.Log
	// Consumers are the extra bus subscribers (events.RegisterSubscribers).
	Consumers []events.Consumer
	// Operators is the platform Operator store and its login (ADR 0026); every call
	// refuses without the `operator` license.
	Operators *operator.Module

	operatorSurface *operator.Surface
}

// New builds the Edition for a license. Subscribers are always registered and each
// checks the license itself, so a key added later needs no rewiring and an unlicensed
// instance stores nothing. The Operator configuration is validated only when the
// `operator` feature is licensed: an unlicensed instance needs none of it.
func New(client *ent.Client, lic *licensekey.License, cipher *secrets.Cipher, siteSecret string, operatorCfg operator.Config) (*Edition, error) {
	e := &Edition{
		lic:       lic,
		client:    client,
		Audit:     audit.NewLog(lic),
		Consumers: []events.Consumer{audit.Consumer(client, lic)},
		Operators: operator.NewModule(client, lic, cipher, operatorCfg),
	}
	if lic.Has(licensekey.FeatureOperator) {
		if err := operatorCfg.Validate(siteSecret); err != nil {
			return nil, err
		}
		e.operatorSurface = operator.NewSurface(e.Operators, operator.NewSessions(client, lic, operatorCfg))
	}
	return e, nil
}

// Operator is the /operator API (ADR 0026) for the composition root to mount, or nil
// without the `operator` license, in which case the whole surface answers 404.
func (e *Edition) Operator() interface {
	Server(opts ...operatorapi.ServerOption) (http.Handler, error)
} {
	if e.operatorSurface == nil {
		return nil
	}
	return e.operatorSurface
}

// Jobs is the Edition's river extension: the advanced-retention prune job (ADR 0014),
// which removes nothing without the retention license.
func (e *Edition) Jobs() jobs.Extension {
	return jobs.Extension{
		Workers: func(w *river.Workers) {
			river.AddWorker(w, retention.NewWorker(e.client, e.lic))
		},
		PeriodicJobs: []*river.PeriodicJob{retention.Periodic()},
	}
}

// Webhooks gates the webhook dispatcher by license: `audit.entry` is forwarded to
// Webhook endpoints only when the Audit feature is licensed.
func (e *Edition) Webhooks(next events.WebhookDispatcher) events.WebhookDispatcher {
	return audit.Forwarder(next, e.lic)
}
