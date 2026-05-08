package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL    string
	Port           string
	CollectSiteKey string
	BootstrapToken string
	CORSOrigins    []string
}

func Load(rootDir, envName string) (*Config, error) {
	v := viper.New()
	v.SetDefault("PORT", "3000")

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
