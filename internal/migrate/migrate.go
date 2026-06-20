// Package migrate applies the embedded Atlas migration files against a
// database, with no atlas CLI required on the host. It lets the distributed
// binary be self-contained: `server migrate` (or AUTO_MIGRATE on startup)
// brings a fresh database up to the current schema.
//
// It deliberately does NOT use Atlas's migrate.Executor: the SDK ships no
// public DB-backed RevisionReadWriter (that code lives in the atlas CLI's
// internal packages), so versioned execution with atlas_schema_revisions
// tracking is not available as a library. Instead it tracks applied versions
// in its own schema_migrations table. This is correct because dev databases
// (managed by the atlas CLI via `make db-migrate`, table atlas_schema_revisions)
// and production databases (managed by this binary, table schema_migrations)
// are always separate — they never share a revisions table, so byte-level
// compatibility with the CLI is unnecessary.
//
// It still reuses the Atlas SDK for the genuinely hard parts: parsing the
// version out of each filename and splitting each file into statements with a
// real PostgreSQL-aware lexer (dollar-quoting, functions, etc.).
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"

	"ariga.io/atlas/sql/migrate"
)

// Apply runs every embedded migration that has not yet been recorded in the
// schema_migrations table, in version order. Each file's statements run inside
// a single transaction together with the bookkeeping insert, so a file applies
// atomically (all-or-nothing).
//
// Note: a migration that needs a statement which cannot run inside a
// transaction (e.g. CREATE INDEX CONCURRENTLY) is not supported by this
// per-file transaction model. None of the current migrations use one.
func Apply(ctx context.Context, db *sql.DB, root fs.FS) error {
	sub, err := fs.Sub(root, "migrations")
	if err != nil {
		return fmt.Errorf("migrate: open migrations dir: %w", err)
	}

	if err := ensureTable(ctx, db); err != nil {
		return err
	}
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	names, err := fs.Glob(sub, "*.sql")
	if err != nil {
		return fmt.Errorf("migrate: glob: %w", err)
	}
	sort.Strings(names)

	for _, name := range names {
		b, err := fs.ReadFile(sub, name)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", name, err)
		}
		// LocalFile gives us Atlas's filename->version parsing and its
		// PostgreSQL-aware statement splitter for free.
		f := migrate.NewLocalFile(name, b)
		version := f.Version()
		if _, ok := applied[version]; ok {
			continue
		}
		stmts, err := f.Stmts()
		if err != nil {
			return fmt.Errorf("migrate: parse %s: %w", name, err)
		}
		if err := applyFile(ctx, db, version, stmts); err != nil {
			return fmt.Errorf("migrate: apply %s: %w", name, err)
		}
	}
	return nil
}

func ensureTable(ctx context.Context, db *sql.DB) error {
	const q = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`
	if _, err := db.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("migrate: ensure schema_migrations: %w", err)
	}
	return nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("migrate: read applied versions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	applied := make(map[string]struct{})
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("migrate: scan version: %w", err)
		}
		applied[v] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: iterate versions: %w", err)
	}
	return applied, nil
}

func applyFile(ctx context.Context, db *sql.DB, version string, stmts []string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return err
	}
	return tx.Commit()
}
