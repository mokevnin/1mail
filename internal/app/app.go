package app

import (
	"context"
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/db"
	"github.com/mokevnin/1mail/internal/email"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/jobs"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/messaging/registry"
	"github.com/mokevnin/1mail/internal/pubsub"
	"github.com/mokevnin/1mail/internal/secrets"
	"github.com/mokevnin/1mail/internal/server"
	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/samber/do/v2"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type App struct {
	Config *config.Config
	Server *http.Server

	injector       *do.RootScope
	pubSub         *pubSub
	jobs           *jobsClient
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

// eventsBus is the producer side of the domain-event system (transactional
// outbox over the database/sql pool). No Shutdown: it borrows the sqlDB pool.
type eventsBus struct {
	*events.Bus
}

// pgxPool is the pgx connection pool river runs on (separate from the
// database/sql pool the ent client and pubsub use).
type pgxPool struct {
	*pgxpool.Pool
}

func (p *pgxPool) Shutdown() {
	p.Close()
}

// jobsClient is the river-backed async job queue (workers + enqueue API).
type jobsClient struct {
	*jobs.Client
}

func (j *jobsClient) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return j.Stop(ctx)
}

func New(env string) (*App, error) {
	injector := do.New()
	register(injector, env)

	cfg, err := do.Invoke[*config.Config](injector)
	if err != nil {
		_ = injector.Shutdown()
		return nil, err
	}

	handler, err := do.Invoke[http.Handler](injector)
	if err != nil {
		_ = injector.Shutdown()
		return nil, err
	}

	ps, err := do.Invoke[*pubSub](injector)
	if err != nil {
		_ = injector.Shutdown()
		return nil, err
	}

	jobsCli, err := do.Invoke[*jobsClient](injector)
	if err != nil {
		_ = injector.Shutdown()
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
		jobs:     jobsCli,
	}, nil
}

func (a *App) RunPubSub(ctx context.Context) error {
	return a.pubSub.Router.Run(ctx)
}

// RunJobs starts the river worker pool. It returns once started; workers run
// until the context is cancelled (stop happens via Shutdown).
func (a *App) RunJobs(ctx context.Context) error {
	return a.jobs.Start(ctx)
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

		client, err := do.Invoke[*entClient](i)
		if err != nil {
			return nil, err
		}
		jc, err := do.Invoke[*jobsClient](i)
		if err != nil {
			return nil, err
		}

		ps, err := pubsub.New(database.DB)
		if err != nil {
			return nil, err
		}
		pubsub.RegisterHandlers(ps, emailSender)

		// Domain events: create the outbox topic schema up front (the tx
		// publisher can't self-initialize) and register the consumers on the
		// shared router. The automations subscriber enrolls via the river client.
		if err := events.InitSchema(context.Background(), database.DB); err != nil {
			return nil, err
		}
		if err := events.RegisterSubscribers(ps.Router, database.DB, client.Client, jc.Client); err != nil {
			return nil, err
		}

		return &pubSub{PubSub: ps}, nil
	})

	do.Provide(injector, func(i do.Injector) (*eventsBus, error) {
		database, err := do.Invoke[*sqlDB](i)
		if err != nil {
			return nil, err
		}
		return &eventsBus{Bus: events.New(database.DB)}, nil
	})

	do.Provide(injector, func(i do.Injector) (*pgxPool, error) {
		cfg, err := do.Invoke[*config.Config](i)
		if err != nil {
			return nil, err
		}
		pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}
		return &pgxPool{Pool: pool}, nil
	})

	do.Provide(injector, func(i do.Injector) (*jobsClient, error) {
		cfg, err := do.Invoke[*config.Config](i)
		if err != nil {
			return nil, err
		}
		client, err := do.Invoke[*entClient](i)
		if err != nil {
			return nil, err
		}
		pool, err := do.Invoke[*pgxPool](i)
		if err != nil {
			return nil, err
		}
		cipher, err := secrets.NewCipher(cfg.EncryptionKey)
		if err != nil {
			return nil, err
		}
		resolver := messaging.NewResolver(client.Client, cipher, registry.Default())
		tracker := tracking.New(cfg.JWTSecret, cfg.AppURL)

		jc, err := jobs.NewClient(pool.Pool, client.Client, resolver, tracker)
		if err != nil {
			return nil, err
		}
		return &jobsClient{Client: jc}, nil
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
		bus, err := do.Invoke[*eventsBus](i)
		if err != nil {
			return nil, err
		}
		jc, err := do.Invoke[*jobsClient](i)
		if err != nil {
			return nil, err
		}

		return server.New(cfg, client.Client, ps.PubSub, bus.Bus, jc.Client)
	})
}
