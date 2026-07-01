package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	onemail "github.com/mokevnin/1mail"
	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/internal/app"
	"github.com/mokevnin/1mail/internal/logging"
	"github.com/mokevnin/1mail/internal/migrate"
	"github.com/mokevnin/1mail/internal/telemetry"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Build metadata, injected via -ldflags by the Makefile / GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		slog.Info("1mail version", "version", version, "commit", commit, "built", date)
		return
	}

	// `server migrate` applies pending migrations and exits — handy for a
	// separate orchestration step (init container, release job, manual run).
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := runMigrate(env); err != nil {
			fatal("migrate", err)
		}
		return
	}

	cfg, err := config.Load(env)
	if err != nil {
		fatal("load config", err)
	}
	// Install the configured logger process-wide; river, watermill, and every
	// request-scoped handler emit through it from here on.
	logging.Setup(cfg)
	slog.Info("starting 1mail", "version", version, "commit", commit)

	// Install the global OTel providers (traces + metrics). The ogen servers and
	// job/event instrumentation pick these up from the globals. The shutdown is
	// deferred at the top level so it runs after the HTTP server has stopped and
	// the app has shut down (LIFO) — the last batched spans/metrics flush then.
	telShutdown, err := telemetry.Setup(context.Background(), cfg, env, telemetry.BuildInfo{Version: version, Commit: commit})
	if err != nil {
		fatal("init telemetry", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := telShutdown(shutdownCtx); err != nil {
			slog.Error("telemetry shutdown", "err", err)
		}
	}()

	// Opt-in self-migration on startup, so the single binary/image can come up
	// against a fresh database with no extra step.
	if cfg.AutoMigrate {
		if err := applyMigrations(cfg); err != nil {
			fatal("auto-migrate", err)
		}
		slog.Info("migrations applied")
	}

	application, err := app.New(env)
	if err != nil {
		fatal("init app", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := application.RunEvents(ctx); err != nil {
			slog.Error("events router stopped", "err", err)
		}
	}()

	if err := application.RunJobs(ctx); err != nil {
		slog.Error("job queue failed to start", "err", err)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = application.Server.Shutdown(shutdownCtx)
		report := application.Shutdown(shutdownCtx)
		if !report.Succeed {
			slog.Error("shutdown incomplete", "report", report)
		}
	}()

	if err := application.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "err", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	report := application.Shutdown(shutdownCtx)
	if !report.Succeed {
		slog.Error("shutdown incomplete", "report", report)
	}
}

// fatal logs an error and exits non-zero (slog has no Fatal helper).
func fatal(msg string, err error) {
	slog.Error(msg, "err", err)
	os.Exit(1)
}

func runMigrate(env string) error {
	cfg, err := config.Load(env)
	if err != nil {
		return err
	}
	if err := applyMigrations(cfg); err != nil {
		return err
	}
	slog.Info("migrations applied")
	return nil
}

// applyMigrations opens a short-lived connection (separate from the app's DI
// pool) and applies the embedded migrations.
func applyMigrations(cfg *config.Config) error {
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return migrate.Apply(ctx, db, onemail.MigrationsFS)
}
