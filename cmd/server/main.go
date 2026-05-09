package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mokevnin/1mail/internal/app"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
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

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		report := application.Shutdown(shutdownCtx)
		if !report.Succeed {
			log.Printf("shutdown: %v", report)
		}
	}()

	if err := application.Server.Start(":" + application.Config.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("server stopped: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	report := application.Shutdown(shutdownCtx)
	if !report.Succeed {
		log.Printf("shutdown: %v", report)
	}
}
