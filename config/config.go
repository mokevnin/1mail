package config

import (
	"fmt"
	"path/filepath"
	"runtime"

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
}

func Load(envName string) (*Config, error) {
	_, file, _, _ := runtime.Caller(0)
	rootDir := filepath.Join(filepath.Dir(file), "..")

	v := viper.New()
	v.SetDefault("PORT", "3000")
	v.SetDefault("APP_URL", "http://localhost:3000")
	v.SetDefault("SMTP_PORT", 1025)

	for _, file := range envFiles(rootDir, envName) {
		sub := viper.New()
		sub.SetConfigFile(file)
		sub.SetConfigType("env")
		if err := sub.ReadInConfig(); err == nil {
			v.MergeConfigMap(sub.AllSettings())
		}
	}

	v.AutomaticEnv()

	if !v.IsSet("DATABASE_URL") {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return &Config{
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
	}, nil
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
