// Package fixtures holds shared helpers for loading the committed YAML fixtures
// in fixtures/*.yml. The same fixtures back the test suite (internal/testhelper)
// and the dev seed (cmd/seed), so the loader options live here to stay in sync.
package fixtures

import (
	"text/template"
	"time"

	"github.com/mokevnin/1mail/internal/secrets"
)

// timeLayout is the Postgres-friendly timestamp layout the template date helpers
// emit. go-testfixtures hands these strings straight to the driver.
const timeLayout = "2006-01-02 15:04:05"

// TemplateFuncs returns the funcs available inside fixture YAML when the loader
// is built with testfixtures.Template(). Two problems make templating necessary:
//
//   - Dates must be relative to "now". Analytics reads broadcast_recipients.sent_at
//     within a window relative to time.Now(); frozen dates render an empty chart.
//     {{daysAgo 2}} / {{daysAhead 3}} keep seeded data inside the window forever.
//   - Encrypted columns can't be static. integration.config_encrypted and
//     webhook_endpoint.secret_encrypted are sealed with the env's ENCRYPTION_KEY,
//     which differs between test and dev — one static ciphertext can't decrypt
//     under both. {{encrypt "<json>"}} seals at load time with the current env's
//     cipher, so the blob is always valid for the database being loaded.
//
// All times are anchored to a single instant per loader build so one load is
// internally consistent.
func TemplateFuncs(cipher *secrets.Cipher) template.FuncMap {
	base := time.Now().UTC()
	at := func(d time.Duration) string { return base.Add(d).Format(timeLayout) }
	day := 24 * time.Hour
	return template.FuncMap{
		"now":       func() string { return base.Format(timeLayout) },
		"daysAgo":   func(n int) string { return at(-time.Duration(n) * day) },
		"daysAhead": func(n int) string { return at(time.Duration(n) * day) },
		"hoursAgo":  func(n int) string { return at(-time.Duration(n) * time.Hour) },
		"encrypt":   func(s string) (string, error) { return cipher.Encrypt([]byte(s)) },
	}
}
