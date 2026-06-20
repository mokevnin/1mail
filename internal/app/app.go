package app

import (
	"context"
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/db"
	"github.com/mokevnin/1mail/internal/email"
	"github.com/mokevnin/1mail/internal/pubsub"
	"github.com/mokevnin/1mail/internal/server"
	"github.com/samber/do/v2"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type App struct {
	Config *config.Config
	Server *http.Server

	injector       *do.RootScope
	pubSub         *pubSub
	shutdownOnce   sync.Once
	shutdownReport *do.ShutdownReport
}

type sqlDB struct {
	*sql.DB
}

func (d *sqlDB) Shutdown() error {
	return d.Close()
}

type entClient struct {
	*ent.Client
}

func (c *entClient) Shutdown() error {
	return c.Close()
}

type pubSub struct {
	*pubsub.PubSub
}

func (ps *pubSub) Shutdown() {
	ps.Close()
}

func New(env string) (*App, error) {
	injector := do.New()
	register(injector, env)

	cfg, err := do.Invoke[*config.Config](injector)
	if err != nil {
		injector.Shutdown()
		return nil, err
	}

	handler, err := do.Invoke[http.Handler](injector)
	if err != nil {
		injector.Shutdown()
		return nil, err
	}

	ps, err := do.Invoke[*pubSub](injector)
	if err != nil {
		injector.Shutdown()
		return nil, err
	}

	return &App{
		Config: cfg,
		Server: &http.Server{
			Addr:              ":" + cfg.Port,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
		injector: injector,
		pubSub:   ps,
	}, nil
}

func (a *App) RunPubSub(ctx context.Context) error {
	return a.pubSub.Router.Run(ctx)
}

func (a *App) Shutdown(ctx context.Context) *do.ShutdownReport {
	a.shutdownOnce.Do(func() {
		a.shutdownReport = a.injector.ShutdownWithContext(ctx)
	})
	return a.shutdownReport
}

func register(injector do.Injector, env string) {
	do.Provide(injector, func(do.Injector) (*config.Config, error) {
		return config.Load(env)
	})

	do.Provide(injector, func(i do.Injector) (*sqlDB, error) {
		cfg, err := do.Invoke[*config.Config](i)
		if err != nil {
			return nil, err
		}

		database, err := sql.Open("pgx", cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}

		return &sqlDB{DB: database}, nil
	})

	do.Provide(injector, func(i do.Injector) (*entClient, error) {
		database, err := do.Invoke[*sqlDB](i)
		if err != nil {
			return nil, err
		}

		return &entClient{Client: db.NewEntClient(database.DB)}, nil
	})

	do.Provide(injector, func(i do.Injector) (*email.Sender, error) {
		cfg, err := do.Invoke[*config.Config](i)
		if err != nil {
			return nil, err
		}

		return email.New(cfg.SMTPHost, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom, cfg.SMTPPort), nil
	})

	do.Provide(injector, func(i do.Injector) (*pubSub, error) {
		database, err := do.Invoke[*sqlDB](i)
		if err != nil {
			return nil, err
		}
		emailSender, err := do.Invoke[*email.Sender](i)
		if err != nil {
			return nil, err
		}

		ps, err := pubsub.New(database.DB)
		if err != nil {
			return nil, err
		}
		pubsub.RegisterHandlers(ps, emailSender)

		return &pubSub{PubSub: ps}, nil
	})

	do.Provide(injector, func(i do.Injector) (http.Handler, error) {
		cfg, err := do.Invoke[*config.Config](i)
		if err != nil {
			return nil, err
		}
		client, err := do.Invoke[*entClient](i)
		if err != nil {
			return nil, err
		}
		ps, err := do.Invoke[*pubSub](i)
		if err != nil {
			return nil, err
		}

		return server.New(cfg, client.Client, ps.PubSub)
	})
}
