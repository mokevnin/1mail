package testhelper

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-testfixtures/testfixtures/v3"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/mokevnin/1mail/backend/config"
	"github.com/mokevnin/1mail/backend/ent"
	"github.com/mokevnin/1mail/backend/internal/db"
	"github.com/mokevnin/1mail/backend/internal/server"
	"github.com/stretchr/testify/require"
)

type TestEnv struct {
	DB       *ent.Client
	Server   *echo.Echo
	fixtures *testfixtures.Loader
}

func Setup(t *testing.T) *TestEnv {
	t.Helper()

	_, file, _, _ := runtime.Caller(0)
	backendRoot := filepath.Join(filepath.Dir(file), "..", "..")
	projectRoot := filepath.Join(backendRoot, "..")

	cfg, err := config.Load(projectRoot, "test")
	require.NoError(t, err, "load test config")

	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	require.NoError(t, err, "open sql.DB")
	t.Cleanup(func() { sqlDB.Close() })

	client := db.NewEntClient(sqlDB)
	t.Cleanup(func() { client.Close() })

	require.NoError(t, client.Schema.Create(context.Background()), "migrate schema")

	fixturesDir := filepath.Join(backendRoot, "fixtures")
	loader, err := testfixtures.New(
		testfixtures.Database(sqlDB),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory(fixturesDir),
	)
	require.NoError(t, err, "create fixtures loader")

	e := server.New(cfg, client)

	return &TestEnv{
		DB:       client,
		Server:   e,
		fixtures: loader,
	}
}

func (env *TestEnv) LoadFixtures(t *testing.T) {
	t.Helper()
	require.NoError(t, env.fixtures.Load())
}
