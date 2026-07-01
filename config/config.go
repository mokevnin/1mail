package config

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/mokevnin/1mail/internal/i18n"
	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL    string
	Port           string
	CollectSiteKey string
	BootstrapToken string
	CORSOrigins    []string
	JWTSecret      string
	AppURL         string
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPass       string
	SMTPFrom       string
	EncryptionKey  string
	AutoMigrate    bool

	// Locale is the instance-wide UI/email language (a SaaS deployment runs per
	// country). One of the supported locales; anything else is coerced to "en".
	// Drives both the SPA (injected into index.html) and the backend's own
	// system emails and validation messages (internal/i18n).
	Locale string

	// Logging: level is one of debug|info|warn|error; format is text|json.
	// Dev defaults to a human-readable coloured text handler, prod to JSON.
	LogLevel  string
	LogFormat string

	// OtelServiceName is the service.name reported on OTel traces/metrics. The
	// OTLP export target itself comes from the standard OTEL_EXPORTER_OTLP_* env
	// vars (read by the SDK); the /metrics Prometheus endpoint is always on.
	OtelServiceName string

	// System (platform) transactional email — 1mail's OWN sender, distinct from a
	// customer's per-workspace integration. Dev uses smtp → mailpit (the SMTP_*
	// values); prod uses ses (the SES_* values). Sent via the same messaging
	// Catalog, not a separate sender.
	SystemEmailProvider string // "smtp" | "ses"
	SystemEmailFrom     string
	SESRegion           string
	SESAccessKeyID      string
	SESSecretAccessKey  string
}

func Load(envName string) (*Config, error) {
	_, file, _, _ := runtime.Caller(0)
	rootDir := filepath.Join(filepath.Dir(file), "..")

	v := viper.New()
	v.SetDefault("PORT", "3000")
	v.SetDefault("APP_URL", "http://localhost:3000")
	v.SetDefault("SMTP_PORT", 1025)
	v.SetDefault("SYSTEM_EMAIL_PROVIDER", "smtp")
	v.SetDefault("SYSTEM_EMAIL_FROM", "noreply@1mail.localhost")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("OTEL_SERVICE_NAME", "1mail")
	v.SetDefault("APP_LOCALE", "en")
	// Human-readable logs in dev, structured JSON everywhere else.
	if isDevEnv(envName) {
		v.SetDefault("LOG_FORMAT", "text")
	} else {
		v.SetDefault("LOG_FORMAT", "json")
	}

	for _, file := range envFiles(rootDir, envName) {
		sub := viper.New()
		sub.SetConfigFile(file)
		sub.SetConfigType("env")
		if err := sub.ReadInConfig(); err == nil {
			if err := v.MergeConfigMap(sub.AllSettings()); err != nil {
				return nil, fmt.Errorf("merge config %s: %w", file, err)
			}
		}
	}

	v.AutomaticEnv()

	if !v.IsSet("DATABASE_URL") {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	cfg := &Config{
		DatabaseURL:    v.GetString("DATABASE_URL"),
		Port:           v.GetString("PORT"),
		CollectSiteKey: v.GetString("COLLECT_SITE_KEY"),
		BootstrapToken: v.GetString("BOOTSTRAP_TOKEN"),
		CORSOrigins:    v.GetStringSlice("CORS_ORIGINS"),
		JWTSecret:      v.GetString("JWT_SECRET"),
		AppURL:         v.GetString("APP_URL"),
		SMTPHost:       v.GetString("SMTP_HOST"),
		SMTPPort:       v.GetInt("SMTP_PORT"),
		SMTPUser:       v.GetString("SMTP_USER"),
		SMTPPass:       v.GetString("SMTP_PASS"),
		SMTPFrom:       v.GetString("SMTP_FROM"),
		EncryptionKey:  v.GetString("ENCRYPTION_KEY"),
		AutoMigrate:    v.GetBool("AUTO_MIGRATE"),
		Locale:         i18n.Normalize(v.GetString("APP_LOCALE")),
		LogLevel:       v.GetString("LOG_LEVEL"),
		LogFormat:      v.GetString("LOG_FORMAT"),

		OtelServiceName: v.GetString("OTEL_SERVICE_NAME"),

		SystemEmailProvider: v.GetString("SYSTEM_EMAIL_PROVIDER"),
		SystemEmailFrom:     v.GetString("SYSTEM_EMAIL_FROM"),
		SESRegion:           v.GetString("SES_REGION"),
		SESAccessKeyID:      v.GetString("SES_ACCESS_KEY_ID"),
		SESSecretAccessKey:  v.GetString("SES_SECRET_ACCESS_KEY"),
	}

	if err := cfg.validate(envName); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate enforces invariants that depend on the deployment environment.
func (c *Config) validate(envName string) error {
	// Outside development/test, an empty JWT_SECRET silently signs auth tokens
	// with an empty key — refuse to boot rather than ship that footgun.
	if !isDevEnv(envName) && c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required outside development")
	}
	return nil
}

// isDevEnv reports whether the env is a non-production one where missing
// security secrets are tolerated (so local dev and tests boot without ceremony).
func isDevEnv(envName string) bool {
	return envName == "" || envName == "development" || envName == "test"
}

func envFiles(rootDir, envName string) []string {
	files := []string{
		filepath.Join(rootDir, ".env"),
		filepath.Join(rootDir, ".env."+envName),
	}
	if envName != "test" {
		files = append(files, filepath.Join(rootDir, ".env.local"))
	}
	files = append(files, filepath.Join(rootDir, ".env."+envName+".local"))
	return files
}
