package config

import (
	"testing"
	"time"
)

func TestConfigValidate_JWTSecret(t *testing.T) {
	cases := []struct {
		name      string
		env       string
		jwtSecret string
		wantErr   bool
	}{
		{name: "production without secret fails", env: "production", jwtSecret: "", wantErr: true},
		{name: "production with secret ok", env: "production", jwtSecret: "s3cret", wantErr: false},
		{name: "staging without secret fails", env: "staging", jwtSecret: "", wantErr: true},
		{name: "development without secret ok", env: "development", jwtSecret: "", wantErr: false},
		{name: "test without secret ok", env: "test", jwtSecret: "", wantErr: false},
		{name: "empty env without secret ok", env: "", jwtSecret: "", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{JWTSecret: tc.jwtSecret, BodyLimits: BodyLimits{Default: 1, Collect: 1}, DBPool: DBPool{MaxOpenConns: 1, ConnMaxLifetime: 1, PGXMaxConns: 1}}
			err := c.validate(tc.env)
			if tc.wantErr && err == nil {
				t.Fatalf("validate(%q) with secret %q: want error, got nil", tc.env, tc.jwtSecret)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validate(%q) with secret %q: unexpected error: %v", tc.env, tc.jwtSecret, err)
			}
		})
	}
}

func TestConfigLoadDBPoolDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	cfg, err := Load("test")
	if err != nil {
		t.Fatal(err)
	}
	want := DBPool{MaxOpenConns: 15, MaxIdleConns: 15, ConnMaxLifetime: 30 * time.Minute, PGXMaxConns: 25}
	if cfg.DBPool != want {
		t.Fatalf("defaults = %+v, want %+v", cfg.DBPool, want)
	}
}

func TestConfigLoadDBPoolFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("DB_MAX_OPEN_CONNS", "7")
	t.Setenv("DB_MAX_IDLE_CONNS", "3")
	t.Setenv("DB_CONN_MAX_LIFETIME", "5m")
	t.Setenv("PGX_MAX_CONNS", "11")
	cfg, err := Load("test")
	if err != nil {
		t.Fatal(err)
	}
	want := DBPool{MaxOpenConns: 7, MaxIdleConns: 3, ConnMaxLifetime: 5 * time.Minute, PGXMaxConns: 11}
	if cfg.DBPool != want {
		t.Fatalf("pool = %+v, want %+v", cfg.DBPool, want)
	}
}

func TestConfigLoadDBPoolIdleFollowsOpen(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("DB_MAX_OPEN_CONNS", "9")
	cfg, err := Load("test")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBPool.MaxIdleConns != 9 {
		t.Fatalf("idle = %d, want 9", cfg.DBPool.MaxIdleConns)
	}
}

func TestConfigLoadDBPoolRejectsInvalid(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"zero open":     {"DB_MAX_OPEN_CONNS": "0"},
		"idle > open":   {"DB_MAX_OPEN_CONNS": "2", "DB_MAX_IDLE_CONNS": "3"},
		"zero lifetime": {"DB_CONN_MAX_LIFETIME": "0s"},
		"zero pgx":      {"PGX_MAX_CONNS": "0"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://x")
			for k, v := range env {
				t.Setenv(k, v)
			}
			if _, err := Load("test"); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestConfigLoadBodyLimitDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	cfg, err := Load("test")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BodyLimits != (BodyLimits{Default: 1 << 20, Collect: 64 << 10}) {
		t.Fatalf("defaults = %+v", cfg.BodyLimits)
	}
}
