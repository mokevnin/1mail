package testhelper

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"net/http/httptest"

	"github.com/DATA-DOG/go-txdb"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	onemail "github.com/mokevnin/1mail"
	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/accounts"
	apiauth "github.com/mokevnin/1mail/internal/api/auth"
	apiexternal "github.com/mokevnin/1mail/internal/api/external"
	apisite "github.com/mokevnin/1mail/internal/api/site"
	"github.com/mokevnin/1mail/internal/authtoken"
	"github.com/mokevnin/1mail/internal/automations"
	"github.com/mokevnin/1mail/internal/broadcasts"
	"github.com/mokevnin/1mail/internal/contacts"
	"github.com/mokevnin/1mail/internal/db"
	"github.com/mokevnin/1mail/internal/erasure"
	"github.com/mokevnin/1mail/internal/eventlog"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/jobs"
	"github.com/mokevnin/1mail/internal/mcpserver"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/messaging/registry"
	"github.com/mokevnin/1mail/internal/oauthserver"
	"github.com/mokevnin/1mail/internal/outbound"
	"github.com/mokevnin/1mail/internal/reputation"
	"github.com/mokevnin/1mail/internal/secrets"
	"github.com/mokevnin/1mail/internal/segments"
	"github.com/mokevnin/1mail/internal/server"
	"github.com/mokevnin/1mail/internal/tags"
	"github.com/mokevnin/1mail/internal/tracking"
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

		// river's own tables, so Erasure can clear the jobs that name a Contact (the
		// queue itself stays inline in tests; see JobsOf and EnqueueJob).
		pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			loadErr = err
			return
		}
		defer pool.Close()
		if err := jobs.Migrate(context.Background(), pool); err != nil {
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
	// Tracker is the instance the server and the sender share: mint tracking,
	// unsubscribe and confirmation tokens with it, never with a second one.
	Tracker *tracking.Tracker

	jwtSecret string // for tokens a test needs in a state the Tracker never mints

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
	tracker := tracking.New(baseCfg.JWTSecret, baseCfg.AppURL)
	sender := outbound.New(bus, resolver, tracker)
	inline := jobs.NewInline(client, sender, systemMail, stubTXT, baseCfg.AppURL)
	// Cipher (over the fixture-sealing key) and provider catalog for the site
	// handlers — mirrors the app's DI singletons.
	cipher, err := secrets.NewCipher(baseCfg.EncryptionKey)
	require.NoError(t, err, "build cipher")
	catalog := registry.Default()
	// The transactional send surface resolves a workspace sender directly (not via
	// river), so it gets the same capturing resolver — its sends land in CustomerMail.
	// inline implements every enqueue seam (broadcast, welcome, account mail,
	// sending-domain verify).
	// Domain modules, built once here and shared by /site and /api exactly like the
	// app's DI singletons.
	eventLog := eventlog.New(bus)
	segmentsModule := segments.New()
	contactsModule := contacts.New(bus)
	erasureModule := erasure.New(bus)
	tagsModule := tags.New()
	automationsModule := automations.New()
	broadcastsModule := broadcasts.New(inline)
	acc := accounts.New(client, bus)
	external, err := server.NewExternalAPI(client, apiexternal.Deps{
		Accounts: acc, Bus: bus, Cipher: cipher, Outbound: sender,
		Segments: segmentsModule, EventLog: eventLog, Contacts: contactsModule, Erasure: erasureModule, Tags: tagsModule,
		Automations: automationsModule, Broadcasts: broadcastsModule, Reputation: reputation.New(),
		BootstrapToken: baseCfg.BootstrapToken,
	})
	require.NoError(t, err, "build external API")
	mcpHandler, err := mcpserver.New(onemail.ExternalOpenAPI, external, apiauth.NewExternalSecurityHandler(client), mcpserver.WithResourceMetadataURL(oauthserver.ResourceMetadataURL(baseCfg.AppURL)))
	require.NoError(t, err, "build MCP handler")
	handler, err := server.New(baseCfg, txDB, client, apisite.Deps{
		Accounts: acc, OAuth: oauthserver.NewService(client), Bus: bus, Cipher: cipher, Catalog: catalog, Outbound: sender,
		Segments: segmentsModule, EventLog: eventLog, Contacts: contactsModule, Erasure: erasureModule, Tags: tagsModule,
		Automations: automationsModule, Broadcasts: broadcastsModule,
		Welcome: inline, SysMail: inline, DomainVerify: inline,
		Tokens: authtoken.New(baseCfg.JWTSecret), Tracker: tracker, AppURL: baseCfg.AppURL,
	}, external, mcpHandler)
	require.NoError(t, err, "build server")

	return &TestEnv{
		DB: client, SQLDB: txDB, Bus: bus, Server: handler, Tracker: tracker, jwtSecret: baseCfg.JWTSecret,
		SystemMail: systemMail, CustomerMail: customerMail,
	}
}

// CapturingSender is a messaging.EmailSender that records sends for assertions.
// Set Err to make Send fail with it — used to drive the send-path error branches
// (e.g. the verified-domain gate) that the real providers are stubbed out of.
type CapturingSender struct {
	mu   sync.Mutex
	sent []messaging.EmailMessage
	Err  error
}

// SetErr makes subsequent Send calls fail with err (nil clears it).
func (s *CapturingSender) SetErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Err = err
}

// DefaultFrom is the integration-configured sender the fixtures' default
// integration uses; its domain (codebasics.dev) is a verified Sending domain, so
// sends without an explicit From pass Outbound send's domain gate (ADR 0010).
func (s *CapturingSender) DefaultFrom() (string, string) { return "hello@codebasics.dev", "CodeBasics" }

func (s *CapturingSender) Send(_ context.Context, msg messaging.EmailMessage) (messaging.Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Err != nil {
		return messaging.Receipt{}, s.Err
	}
	s.sent = append(s.sent, msg)
	return messaging.Receipt{MessageID: fmt.Sprintf("<test-%d@1mail.test>", len(s.sent))}, nil
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

func (r fixedResolver) EmailSender(context.Context, *ent.Scoped) (messaging.EmailSender, error) {
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
