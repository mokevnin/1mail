//go:build e2e

// Package e2e is the end-to-end suite (ADR 0024): the whole application runs in the
// test process on a real listener, a Workspace's SMTP Integration points at a Mailpit
// started by the suite, everything is driven through the public HTTP API and the
// result is observed only in the Mailpit inbox. Tests never read the database, the
// outbox or an in-memory sender.
//
// Scenarios are written as domain steps on a [Workspace] (ImportContacts,
// SendBroadcast, WaitForEmail ...); the HTTP transport stays inside this package, so a
// second transport (MCP) can later replay the same scenarios.
package e2e

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/mokevnin/1mail/internal/accounts"
	"github.com/mokevnin/1mail/internal/app"
)

// Env is the running application under test plus its Mailpit. One Env serves every
// test of the run; tests isolate themselves with their own Workspace.
type Env struct {
	// BaseURL is the application's public URL (the base of every link in an email).
	BaseURL string
	Mailpit *Mailpit

	accounts *accounts.Accounts
}

// Boot starts Mailpit and the full application (event router, job queue, HTTP server)
// on a free loopback port and returns the Env with a stop function. The database named
// by DATABASE_URL must already be migrated (the e2e task does that on a fresh one).
func Boot() (*Env, func(), error) {
	ctx := context.Background()
	mp, stopMailpit, err := StartMailpit(ctx)
	if err != nil {
		return nil, nil, err
	}

	// The listener comes first: the public URL (APP_URL) is derived from its address.
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		stopMailpit()
		return nil, nil, err
	}
	a, err := app.New("e2e", app.WithListener(ln), app.WithE2EDKIMLookup())
	if err != nil {
		_ = ln.Close()
		stopMailpit()
		return nil, nil, fmt.Errorf("start application: %w", err)
	}
	if err := a.BindMetrics(); err != nil {
		_ = a.Shutdown(ctx)
		stopMailpit()
		return nil, nil, err
	}

	// The arrange step goes through the product's own Accounts module, taken from the
	// application's container (no second pool).
	acc, err := a.Accounts()
	if err != nil {
		_ = a.Shutdown(ctx)
		stopMailpit()
		return nil, nil, fmt.Errorf("resolve accounts: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	serveErr := make(chan error, 1)
	go func() { _ = a.RunEvents(runCtx) }()
	if err := a.RunJobs(runCtx); err != nil {
		cancel()
		_ = a.Shutdown(ctx)
		stopMailpit()
		return nil, nil, fmt.Errorf("start job queue: %w", err)
	}
	go func() { serveErr <- a.Serve() }()

	stop := func() {
		cancel()
		sctx, scancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer scancel()
		_ = a.Stop(sctx)
		_ = a.Shutdown(sctx)
		stopMailpit()
		select {
		case err := <-serveErr:
			_ = err
		case <-sctx.Done():
		}
	}
	return &Env{
		BaseURL:  a.Config.AppURL,
		Mailpit:  mp,
		accounts: acc,
	}, stop, nil
}
