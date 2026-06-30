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
			c := &Config{JWTSecret: tc.jwtSecret}
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
