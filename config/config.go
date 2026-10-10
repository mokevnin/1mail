package config

import (
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/mokevnin/1mail/internal/i18n"
	"github.com/spf13/viper"
)

// BodyLimits are the request body caps in bytes. Collect is the public tracking
// ingestion (its key ships in customer pages, so anyone can post to it); Default
// applies to every other surface.
type BodyLimits struct {
	Default int64
	Collect int64
}

// RateLimits are the per-policy request budgets per minute (ADR 0018). Every limit
// has a default and 0 disables it. They are core, never gated by the EE licence.
type RateLimits struct {
	// Human caps the public human-facing endpoints (signup, invitation accept,
	// consent confirm) per client IP and endpoint.
	Human int
	// APIBurst caps /api and /mcp per Workspace per second (one shared budget).
	APIBurst int
	// APIPerMinute caps /api and /mcp per Workspace per minute, stacked on APIBurst.
	APIPerMinute int
	// FailedAuth caps failed credential checks (bearer token, collect key) per
	// client IP per minute; successful ones are not counted.
	FailedAuth int
	// Tracking caps how many opens and clicks one client IP may have recorded per
	// minute. It never refuses a recipient: over it the event is not recorded.
	Tracking int
}

// DefaultRateLimits are the production budgets.
var DefaultRateLimits = RateLimits{Human: 60, APIBurst: 20, APIPerMinute: 600, FailedAuth: 30, Tracking: 600}

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
	BodyLimits     BodyLimits
	RateLimits     RateLimits
	// IsDev is true for non-production envs (development/test). Used to relax
	// production-only behaviour locally — e.g. the sending-domain DKIM re-check
	// trusts seeded domains instead of hitting real DNS (ADR 0010).
	IsDev bool

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
	// vars (read by the SDK).
	OtelServiceName string

	// MetricsAddr (host:port) is where the opt-in Prometheus listener binds. Empty
	// (the default) means no listener; the public port never serves /metrics
	// (ADR 0018).
	MetricsAddr string

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
	v.SetDefault("MAX_BODY_BYTES", 1<<20)
	v.SetDefault("COLLECT_MAX_BODY_BYTES", 64<<10)
	v.SetDefault("RATE_LIMIT_HUMAN_PER_MINUTE", DefaultRateLimits.Human)
	v.SetDefault("RATE_LIMIT_API_BURST_PER_SECOND", DefaultRateLimits.APIBurst)
	v.SetDefault("RATE_LIMIT_API_PER_MINUTE", DefaultRateLimits.APIPerMinute)
	v.SetDefault("RATE_LIMIT_FAILED_AUTH_PER_MINUTE", DefaultRateLimits.FailedAuth)
	v.SetDefault("RATE_LIMIT_TRACKING_PER_MINUTE", DefaultRateLimits.Tracking)
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

		BodyLimits: BodyLimits{
			Default: v.GetInt64("MAX_BODY_BYTES"),
			Collect: v.GetInt64("COLLECT_MAX_BODY_BYTES"),
		},
		RateLimits: RateLimits{
			Human:        v.GetInt("RATE_LIMIT_HUMAN_PER_MINUTE"),
			APIBurst:     v.GetInt("RATE_LIMIT_API_BURST_PER_SECOND"),
			APIPerMinute: v.GetInt("RATE_LIMIT_API_PER_MINUTE"),
			FailedAuth:   v.GetInt("RATE_LIMIT_FAILED_AUTH_PER_MINUTE"),
			Tracking:     v.GetInt("RATE_LIMIT_TRACKING_PER_MINUTE"),
		},
		IsDev:     isDevEnv(envName),
		Locale:    i18n.Normalize(v.GetString("APP_LOCALE")),
		LogLevel:  v.GetString("LOG_LEVEL"),
		LogFormat: v.GetString("LOG_FORMAT"),

		OtelServiceName: v.GetString("OTEL_SERVICE_NAME"),
		MetricsAddr:     v.GetString("METRICS_ADDR"),

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
	if c.BodyLimits.Default <= 0 {
		return fmt.Errorf("MAX_BODY_BYTES must be positive")
	}
	if c.BodyLimits.Collect <= 0 {
		return fmt.Errorf("COLLECT_MAX_BODY_BYTES must be positive")
	}
	for name, limit := range map[string]int{
		"RATE_LIMIT_HUMAN_PER_MINUTE":       c.RateLimits.Human,
		"RATE_LIMIT_API_BURST_PER_SECOND":   c.RateLimits.APIBurst,
		"RATE_LIMIT_API_PER_MINUTE":         c.RateLimits.APIPerMinute,
		"RATE_LIMIT_FAILED_AUTH_PER_MINUTE": c.RateLimits.FailedAuth,
	} {
		if limit < 0 {
			return fmt.Errorf("%s must not be negative (0 disables)", name)
		}
	}
	if c.RateLimits.Tracking < 0 {
		return fmt.Errorf("RATE_LIMIT_TRACKING_PER_MINUTE must not be negative (0 disables)")
	}
	return c.validateMetricsAddr()
}

// validateMetricsAddr rejects a malformed METRICS_ADDR and one sharing the public
// PORT (the public server binds every interface, so any host collides).
func (c *Config) validateMetricsAddr() error {
	if c.MetricsAddr == "" {
		return nil
	}
	_, portStr, err := net.SplitHostPort(c.MetricsAddr)
	if err != nil {
		return fmt.Errorf("METRICS_ADDR must be host:port: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("METRICS_ADDR port %q is not a valid port", portStr)
	}
	if public, err := strconv.Atoi(c.Port); err == nil && public == port {
		return fmt.Errorf("METRICS_ADDR must not use the public PORT (%d)", port)
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
