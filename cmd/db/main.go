package main

import (
	"database/sql"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mokevnin/1mail/config"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: db <create|drop>")
	}

	envName := os.Getenv("APP_ENV")
	if envName == "" {
		envName = "development"
	}

	cfg, err := config.Load(envName)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	u, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("parse DATABASE_URL: %v", err)
	}

	dbName := strings.TrimPrefix(u.Path, "/")
	u.Path = "/postgres"

	adminDB, err := sql.Open("pgx", u.String())
	if err != nil {
		log.Fatalf("open admin db: %v", err)
	}
	defer func() { _ = adminDB.Close() }()

	switch os.Args[1] {
	case "create":
		_, err = adminDB.Exec(`CREATE DATABASE "` + dbName + `"`)
		if err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "42P04" {
				log.Printf("database %q already exists, skipping", dbName)
				return
			}
			log.Fatalf("create database: %v", err)
		}
		log.Printf("database %q created", dbName)
	case "drop":
		_, err = adminDB.Exec(`DROP DATABASE IF EXISTS "` + dbName + `"`)
		if err != nil {
			log.Fatalf("drop database: %v", err)
		}
		log.Printf("database %q dropped", dbName)
	default:
		log.Fatalf("unknown command %q, expected create or drop", os.Args[1])
	}
}
