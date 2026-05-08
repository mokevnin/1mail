package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/internal/db"
	"github.com/mokevnin/1mail/internal/email"
	"github.com/mokevnin/1mail/internal/pubsub"
	"github.com/mokevnin/1mail/internal/server"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	cfg, err := config.Load(env)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer sqlDB.Close()

	client := db.NewEntClient(sqlDB)
	defer client.Close()

	ps, err := pubsub.New(sqlDB)
	if err != nil {
		log.Fatalf("init pubsub: %v", err)
	}
	defer ps.Close()

	emailSender := email.New(cfg.SMTPHost, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom, cfg.SMTPPort)
	pubsub.RegisterHandlers(ps, emailSender)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := ps.Router.Run(ctx); err != nil {
			log.Printf("pubsub router stopped: %v", err)
		}
	}()

	e := server.New(cfg, client, ps)
	log.Fatal(e.Start(":" + cfg.Port))
}
