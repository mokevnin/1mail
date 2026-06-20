// Package onemail is the module-root package. It exists only to embed the
// Atlas-generated migration files (which live in ./migrations at the repo
// root) so the single binary can apply them itself, with no atlas CLI on the
// host. The embed directive must live at or above the migrations directory in
// the source tree — go:embed cannot reference parent paths — hence this file
// sits at the module root rather than in internal/migrate.
package onemail

import "embed"

// MigrationsFS holds the committed Atlas migration files. Consumed by
// internal/migrate. Regenerated via `make db-generate` (Atlas), never by hand.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
