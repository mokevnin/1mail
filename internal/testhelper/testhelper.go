package testhelper

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/DATA-DOG/go-txdb"
	"github.com/go-testfixtures/testfixtures/v3"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/db"
	"github.com/mokevnin/1mail/internal/server"
	ht "github.com/ogen-go/ogen/http"
	"github.com/stretchr/testify/require"
)

// Baseline is applied ONCE per test process: migrate schema + load fixtures
// (committed). Each test then opens a go-txdb connection — a real transaction
// that is rolled back on Close — so tests are fully isolated and the DB stays
// at the fixture state. No hand-rolled transaction plumbing.
var (
	once    sync.Once
	baseCfg *config.Config
	loadErr error
)

func initBaseline() {
	once.Do(func() {
		_, file, _, _ := runtime.Caller(0)
		projectRoot := filepath.Join(filepath.Dir(file), "..", "..")

		cfg, err := config.Load("test")
		if err != nil {
			loadErr = err
			return
		}

		sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
		if err != nil {
			loadErr = err
			return
		}
		defer func() { _ = sqlDB.Close() }()

		if err := db.NewEntClient(sqlDB).Schema.Create(context.Background()); err != nil {
			loadErr = err
			return
		}

		loader, err := testfixtures.New(
			testfixtures.Database(sqlDB),
			testfixtures.Dialect("postgres"),
			testfixtures.Directory(filepath.Join(projectRoot, "fixtures")),
		)
		if err != nil {
			loadErr = err
			return
		}
		if err := loader.Load(); err != nil {
			loadErr = err
			return
		}

		// Register a transactional driver backed by the real pgx connection.
		txdb.Register("txdb", "pgx", cfg.DatabaseURL)
		baseCfg = cfg
	})
}

type TestEnv struct {
	DB     *ent.Client // transaction-bound; rolled back when the test finishes
	Server http.Handler
}

func Setup(t *testing.T) *TestEnv {
	t.Helper()
	initBaseline()
	require.NoError(t, loadErr, "init test baseline")

	// dsn arg is just a pool identifier; each Open is its own transaction.
	txDB, err := sql.Open("txdb", t.Name())
	require.NoError(t, err, "open txdb")
	// All queries must ride the SAME transaction; >1 conn would each get a
	// separate tx and not see this test's writes (e.g. seeded tokens).
	txDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = txDB.Close() }) // rollback

	client := db.NewEntClient(txDB)
	handler, err := server.New(baseCfg, client, nil, noopEnqueuer{})
	require.NoError(t, err, "build server")

	return &TestEnv{DB: client, Server: handler}
}

// noopEnqueuer satisfies the site handlers' BroadcastEnqueuer without a river
// runtime: tests assert on the resulting state/status, not on async dispatch
// (the send engine is tested directly via jobs.SendBroadcast).
type noopEnqueuer struct{}

func (noopEnqueuer) EnqueueBroadcast(context.Context, int64, *time.Time) error { return nil }

// Transport returns an ogen http.Client that dispatches in-memory to the
// server (no socket). Pass it via the generated client's WithClient option so
// the typed client builds URLs and encodes/decodes DTOs for you. extraHeaders
// are applied to every request (e.g. x-collect-key).
func (env *TestEnv) Transport(extraHeaders map[string]string) ht.Client {
	return injectClient{handler: env.Server, headers: extraHeaders}
}

type injectClient struct {
	handler http.Handler
	headers map[string]string
}

func (c injectClient) Do(r *http.Request) (*http.Response, error) {
	for k, v := range c.headers {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, r)
	return rec.Result(), nil
}
