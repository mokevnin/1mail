package config

import "testing"

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
