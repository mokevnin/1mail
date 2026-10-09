package events_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/internal/db"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

type recordingEnroller struct {
	mu    sync.Mutex
	calls []string
}

func (r *recordingEnroller) OnEvent(_ context.Context, _, _ int64, action string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, action)
	return nil
}

func (r *recordingEnroller) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

type recordingDispatcher struct {
	mu    sync.Mutex
	names []string
}

func (r *recordingDispatcher) Dispatch(_ context.Context, _ int64, name, _ string, _ []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names = append(r.names, name)
	return nil
}

func (r *recordingDispatcher) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.names)
}

// RegisterSubscribers wires the four consumers onto one router; a published hard
// bounce reaches every one of them. It runs against the real DB (the router reads
// the committed outbox), and removes what it committed on exit.
func TestRegisterSubscribersFansEveryEventOutToAllConsumers(t *testing.T) {
	cfg, err := config.Load("test")
	require.NoError(t, err)
	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	client := db.NewEntClient(sqlDB)

	ctx := context.Background()
	require.NoError(t, events.InitSchema(ctx, sqlDB))
	const dest = "registered-bounce@example.com"
	cleanup := func() {
		bg := context.WithoutCancel(ctx)
		_ = testhelper.PurgeOutbox(bg, sqlDB)
		_, _ = client.Suppression.Delete().Where(suppression.Destination(dest)).Exec(bg)
		_, _ = client.Event.Delete().Where(event.SubjectID(dest)).Exec(bg)
	}
	cleanup()
	t.Cleanup(cleanup)

	router, err := events.NewRouter()
	require.NoError(t, err)
	t.Cleanup(func() { _ = router.Close() })
	enroller, dispatcher := &recordingEnroller{}, &recordingDispatcher{}
	require.NoError(t, events.RegisterSubscribers(router, sqlDB, client, enroller, dispatcher))

	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	go func() { _ = router.Run(runCtx) }()
	<-router.Running()

	require.NoError(t, events.New(sqlDB).WithinTx(ctx, func(_ *ent.Client, pub events.Publisher) error {
		return pub.Publish(ctx, &events.EmailDeliveryFailure{
			Action: events.NameEmailBounced, WorkspaceID: fixtures.AcmeID, ContactID: fixtures.ContactAliceID,
			Email: dest, BounceKind: events.BounceKindPermanent, DedupID: "registered-bounce-1",
		})
	}))

	require.Eventually(t, func() bool {
		persisted, _ := client.Event.Query().Where(event.SubjectID(dest)).Exist(ctx)
		suppressed, _ := client.Suppression.Query().Where(suppression.Destination(dest)).Exist(ctx)
		return persisted && suppressed && enroller.count() == 1 && dispatcher.count() == 1
	}, 15*time.Second, 50*time.Millisecond, "persist, suppression, automations and webhooks must all see the event")
	assert.Equal(t, []string{events.NameEmailBounced}, dispatcher.names)
}
