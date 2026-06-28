package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	onemail "github.com/mokevnin/1mail"
	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/internal/app"
	"github.com/mokevnin/1mail/internal/migrate"

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
		log.Printf("1mail %s (commit %s, built %s)", version, commit, date)
		return
	}

	// `server migrate` applies pending migrations and exits — handy for a
	// separate orchestration step (init container, release job, manual run).
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := runMigrate(env); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		return
	}

	cfg, err := config.Load(env)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	log.Printf("starting 1mail %s (commit %s)", version, commit)

	// Opt-in self-migration on startup, so the single binary/image can come up
	// against a fresh database with no extra step.
	if cfg.AutoMigrate {
		if err := applyMigrations(cfg); err != nil {
			log.Fatalf("auto-migrate: %v", err)
		}
		log.Print("migrations applied")
	}

	application, err := app.New(env)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := application.RunPubSub(ctx); err != nil {
			log.Printf("pubsub router stopped: %v", err)
		}
	}()

	if err := application.RunJobs(ctx); err != nil {
		log.Printf("job queue failed to start: %v", err)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = application.Server.Shutdown(shutdownCtx)
		report := application.Shutdown(shutdownCtx)
		if !report.Succeed {
			log.Printf("shutdown: %v", report)
		}
	}()

	if err := application.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("server stopped: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	report := application.Shutdown(shutdownCtx)
	if !report.Succeed {
		log.Printf("shutdown: %v", report)
	}
}

func runMigrate(env string) error {
	cfg, err := config.Load(env)
	if err != nil {
		return err
	}
	if err := applyMigrations(cfg); err != nil {
		return err
	}
	log.Print("migrations applied")
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
