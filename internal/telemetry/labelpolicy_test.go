package telemetry_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/jobs"
	"github.com/mokevnin/1mail/internal/outbound"
	"github.com/mokevnin/1mail/internal/secrets"
	"github.com/mokevnin/1mail/internal/telemetry"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/mokevnin/1mail/internal/tracking"
)

// forbiddenLabel matches label names that identify a tenant or a person (ADR 0018):
// workspace id or slug, contact, email, recipient. Span attributes are out of scope.
var forbiddenLabel = regexp.MustCompile(`(?i)workspace|slug|contact|email|recipient`)

// The metric label policy (ADR 0018) is enforced against the real exposition of the
// metrics listener after real instrumented work ran: a domain event handled by the
// watermill router (carrying a Workspace id and a Contact email) and a job run by the
// river runtime. Any tenant-bearing label or label value would show up here.
func TestMetricsExpositionCarriesNoTenantLabels(t *testing.T) {
	ctx := context.Background()
	cfg, err := config.Load("test")
	require.NoError(t, err)

	// Global providers must be installed before the instrumented components are built.
	prevMP, prevTP := otelGlobals()
	t.Cleanup(func() { restoreOtelGlobals(prevMP, prevTP) })
	stop, err := telemetry.Setup(ctx, &config.Config{OtelServiceName: "1mail-test"}, "test", telemetry.BuildInfo{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = stop(context.Background()) })

	srv := telemetry.NewMetricsServer("127.0.0.1:0")
	require.NoError(t, srv.Listen())
	go func() { _ = srv.Serve() }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })

	env := testhelper.Setup(t)
	workspace := env.DB.Workspace.GetX(ctx, fixtures.AcmeID)
	const contactEmail = "label-policy@example.com"

	runHandledDomainEvent(t, cfg, contactEmail)
	runJob(t, env, cfg, contactEmail)

	body := scrapeBody(t, "http://"+srv.Addr().String()+"/metrics")

	// Proof that the instrumented paths ran: their metrics are in the exposition.
	require.Contains(t, body, "watermill_messages_processed", "domain-event handler metrics missing")
	require.Contains(t, body, "river_", "job metrics missing")

	parser := expfmt.NewTextParser(model.UTF8Validation)
	families, err := parser.TextToMetricFamilies(strings.NewReader(body))
	require.NoError(t, err)
	require.NotEmpty(t, families)
	for name, family := range families {
		for _, m := range family.GetMetric() {
			for _, label := range m.GetLabel() {
				assert.NotRegexp(t, forbiddenLabel, label.GetName(), "forbidden label name on %s", name)
				for _, secret := range []string{workspace.Slug, contactEmail} {
					assert.NotContains(t, label.GetValue(), secret, "tenant value in label %s of %s", label.GetName(), name)
				}
			}
		}
	}
}

func scrapeBody(t *testing.T, url string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(body)
}

// runHandledDomainEvent publishes a tenant-bearing event through the outbox and waits
// until the real router (with the OTel middleware) has handled it.
func runHandledDomainEvent(t *testing.T, cfg *config.Config, contactEmail string) {
	t.Helper()
	ctx := context.Background()
	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, events.InitSchema(ctx, sqlDB))
	t.Cleanup(func() { _ = testhelper.PurgeOutbox(context.WithoutCancel(ctx), sqlDB) })

	router, err := events.NewRouter()
	require.NoError(t, err)
	t.Cleanup(func() { _ = router.Close() })
	sub, err := events.NewSubscriber(sqlDB, "label-policy")
	require.NoError(t, err)
	handled := make(chan struct{}, 1)
	router.AddConsumerHandler("label_policy", events.TopicDomainEvents, sub, func(msg *message.Message) error {
		var env events.Envelope
		if err := json.Unmarshal(msg.Payload, &env); err != nil {
			return err
		}
		select {
		case handled <- struct{}{}:
		default:
		}
		return nil
	})
	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	go func() { _ = router.Run(runCtx) }()
	<-router.Running()

	require.NoError(t, events.New(sqlDB).WithinTx(ctx, func(_ *ent.Client, pub events.Publisher) error {
		return pub.Publish(ctx, &events.ContactCreated{WorkspaceID: fixtures.AcmeID, ContactID: fixtures.ContactAliceID, Email: contactEmail})
	}))
	select {
	case <-handled:
	case <-time.After(10 * time.Second):
		t.Fatal("domain event was not handled within 10s")
	}
	// The middleware records its metrics right after the handler returns.
	time.Sleep(200 * time.Millisecond)
}

// runJob runs a real river runtime and waits for a job to be worked.
func runJob(t *testing.T, env *testhelper.TestEnv, cfg *config.Config, contactEmail string) {
	t.Helper()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, jobs.Migrate(ctx, pool))
	cipher, err := secrets.NewCipher(cfg.EncryptionKey)
	require.NoError(t, err)
	mod := outbound.New(env.Bus, nil, tracking.New("test-secret", "http://local"))
	client, err := jobs.NewClient(pool, env.DB, mod, cipher, env.SystemMail, nil, cfg.AppURL)
	require.NoError(t, err)
	admin, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	require.NoError(t, err)
	clear := func() {
		_, _ = admin.JobDeleteMany(context.Background(), river.NewJobDeleteManyParams().Queues(river.QueueDefault, jobs.QueueBroadcasts, jobs.QueueWebhooks))
	}
	clear()
	t.Cleanup(clear)
	require.NoError(t, client.Start(ctx))
	t.Cleanup(func() { _ = client.Stop(context.Background()) })

	require.NoError(t, client.EnqueueWelcome(ctx, contactEmail, "Policy"))
	require.Eventually(t, func() bool {
		for _, m := range env.SystemMail.Messages() {
			if m.To == contactEmail {
				return true
			}
		}
		return false
	}, 10*time.Second, 50*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
}

func otelGlobals() (metric.MeterProvider, trace.TracerProvider) {
	return otel.GetMeterProvider(), otel.GetTracerProvider()
}

func restoreOtelGlobals(mp metric.MeterProvider, tp trace.TracerProvider) {
	otel.SetMeterProvider(mp)
	otel.SetTracerProvider(tp)
}
