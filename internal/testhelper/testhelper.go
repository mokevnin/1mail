package testhelper

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"net/http/httptest"

	"github.com/DATA-DOG/go-txdb"
	"github.com/go-testfixtures/testfixtures/v3"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/db"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/jobs"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/secrets"
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

		// The domain-event outbox topic table is a watermill-sql table, not part
		// of the ent schema, so create it here too (once per process, on the real
		// DB) — otherwise the transactional publisher hits "relation does not exist".
		if err := events.InitSchema(context.Background(), sqlDB); err != nil {
			loadErr = err
			return
		}

		cipher, err := secrets.NewCipher(cfg.EncryptionKey)
		if err != nil {
			loadErr = err
			return
		}

		loader, err := testfixtures.New(
			testfixtures.Database(sqlDB),
			testfixtures.Dialect("postgres"),
			// Template options MUST precede Directory: Directory eagerly reads and
			// renders the files when its option is applied. Fixtures use templating
			// for relative dates and load-time encryption (internal/fixtures).
			testfixtures.Template(),
			testfixtures.TemplateFuncs(fixtures.TemplateFuncs(cipher)),
			testfixtures.Directory(filepath.Join(projectRoot, "fixtures")),
			// ResetSequencesTo keeps the explicit fixture ids clear of app-inserted rows.
			testfixtures.ResetSequencesTo(100000),
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
	SQLDB  *sql.DB     // the same txdb connection the ent client and bus ride
	Bus    *events.Bus // domain-event bus over SQLDB (nested savepoint per WithinTx)
	Server http.Handler

	// Captured sends from the inline jobs adapter, for assertions.
	SystemMail   *CapturingSender // platform mail (welcome, …)
	CustomerMail *CapturingSender // workspace/campaign mail (broadcasts)
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
	// Bus over the same txdb connection: WithinTx opens a nested (savepoint)
	// transaction, so producer writes + outbox inserts roll back with the test.
	bus := events.New(txDB)

	// Inline jobs adapter (Rails :inline equivalent): enqueued jobs run
	// synchronously against capturing senders, so tests assert the real effect.
	systemMail := &CapturingSender{}
	customerMail := &CapturingSender{}
	resolver := fixedResolver{sender: customerMail}
	// stubTXT keeps sending-domain verification off real DNS: every lookup is
	// NXDOMAIN, so an inline verify deterministically resolves to "not published".
	stubTXT := func(context.Context, string) ([]string, error) {
		return nil, &net.DNSError{IsNotFound: true}
	}
	inline := jobs.NewInline(client, bus, resolver, nil, systemMail, stubTXT, baseCfg.AppURL)
	// The transactional send surface resolves a workspace sender directly (not via
	// river), so it gets the same capturing resolver — its sends land in CustomerMail.
	// inline implements every enqueue seam (broadcast, welcome, account mail,
	// sending-domain verify).
	handler, err := server.New(baseCfg, client, txDB, bus, inline, inline, inline, inline, resolver)
	require.NoError(t, err, "build server")

	return &TestEnv{
		DB: client, SQLDB: txDB, Bus: bus, Server: handler,
		SystemMail: systemMail, CustomerMail: customerMail,
	}
}

// CapturingSender is a messaging.EmailSender that records sends for assertions.
type CapturingSender struct {
	mu   sync.Mutex
	sent []messaging.EmailMessage
}

func (s *CapturingSender) Send(_ context.Context, msg messaging.EmailMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, msg)
	return nil
}

// Messages returns a copy of the captured sends.
func (s *CapturingSender) Messages() []messaging.EmailMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]messaging.EmailMessage(nil), s.sent...)
}

// fixedResolver returns the same capturing sender for any workspace, so inline
// broadcast sends work without a configured integration.
type fixedResolver struct{ sender messaging.EmailSender }

func (r fixedResolver) EmailSender(context.Context, int64) (messaging.EmailSender, error) {
	return r.sender, nil
}

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
