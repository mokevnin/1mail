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
			c := &Config{JWTSecret: tc.jwtSecret, BodyLimits: BodyLimits{Default: 1, Collect: 1}}
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

func TestConfigLoadOutboxFloor(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	cfg, err := Load("test")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OutboxFloor != 7*24*time.Hour {
		t.Fatalf("default OutboxFloor = %v, want 7 days", cfg.OutboxFloor)
	}

	t.Setenv("OUTBOX_RETENTION_FLOOR_DAYS", "14")
	cfg, err = Load("test")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OutboxFloor != 14*24*time.Hour {
		t.Fatalf("OutboxFloor = %v, want 14 days", cfg.OutboxFloor)
	}

	t.Setenv("OUTBOX_RETENTION_FLOOR_DAYS", "-1")
	if _, err := Load("test"); err == nil {
		t.Fatal("a negative floor must be rejected")
	}
}
