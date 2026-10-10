package config

import (
	"strings"
	"testing"
)

var strongSecret = strings.Repeat("k7", 16)

func TestConfigLoad_JWTSecret(t *testing.T) {
	cases := []struct {
		name      string
		env       string
		jwtSecret string
		wantErr   bool
	}{
		{name: "production without secret fails", env: "production", jwtSecret: "", wantErr: true},
		{name: "production short secret fails", env: "production", jwtSecret: "s3cret", wantErr: true},
		{name: "production 31 chars fails", env: "production", jwtSecret: strongSecret[:31], wantErr: true},
		{name: "production dev default fails", env: "production", jwtSecret: "dev-secret-change-in-production", wantErr: true},
		{name: "production change-me padded fails", env: "production", jwtSecret: "change-me-change-me-change-me-change-me", wantErr: true},
		{name: "production placeholder case-insensitive fails", env: "production", jwtSecret: "CHANGEME-CHANGEME-CHANGEME-CHANGEME", wantErr: true},
		{name: "production strong secret ok", env: "production", jwtSecret: strongSecret, wantErr: false},
		{name: "staging short secret fails", env: "staging", jwtSecret: "s3cret", wantErr: true},
		{name: "development without secret ok", env: "development", jwtSecret: "", wantErr: false},
		{name: "development placeholder ok", env: "development", jwtSecret: "dev-secret-change-in-production", wantErr: false},
		{name: "test without secret ok", env: "test", jwtSecret: "", wantErr: false},
		{name: "empty env without secret ok", env: "", jwtSecret: "", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://x")
			t.Setenv("JWT_SECRET", tc.jwtSecret)
			_, err := Load(tc.env)
			if tc.wantErr && err == nil {
				t.Fatalf("Load(%q) with secret %q: want error, got nil", tc.env, tc.jwtSecret)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Load(%q) with secret %q: unexpected error: %v", tc.env, tc.jwtSecret, err)
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

func TestConfigLoadMetricsAddr(t *testing.T) {
	cases := []struct {
		name    string
		port    string
		addr    string
		wantErr bool
	}{
		{name: "empty disables the listener", port: "3000", addr: ""},
		{name: "distinct host and port", port: "3000", addr: "127.0.0.1:9090"},
		{name: "all interfaces", port: "3000", addr: "0.0.0.0:9090"},
		{name: "same port as PORT", port: "3000", addr: "127.0.0.1:3000", wantErr: true},
		{name: "same port with another host", port: "3000", addr: "0.0.0.0:3000", wantErr: true},
		{name: "same port with an empty host", port: "3000", addr: ":3000", wantErr: true},
		{name: "missing port", port: "3000", addr: "127.0.0.1", wantErr: true},
		{name: "non-numeric port", port: "3000", addr: "127.0.0.1:metrics", wantErr: true},
		{name: "port out of range", port: "3000", addr: "127.0.0.1:70000", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://x")
			t.Setenv("PORT", tc.port)
			t.Setenv("METRICS_ADDR", tc.addr)
			cfg, err := Load("test")
			if tc.wantErr {
				if err == nil {
					t.Fatalf("METRICS_ADDR=%q PORT=%s: want error, got nil", tc.addr, tc.port)
				}
				return
			}
			if err != nil {
				t.Fatalf("METRICS_ADDR=%q PORT=%s: unexpected error: %v", tc.addr, tc.port, err)
			}
			if cfg.MetricsAddr != tc.addr {
				t.Fatalf("MetricsAddr = %q, want %q", cfg.MetricsAddr, tc.addr)
			}
		})
	}
}
