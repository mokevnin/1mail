package main

import (
	"database/sql"
	"log"

	"github.com/go-testfixtures/testfixtures/v3"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mokevnin/1mail/config"
)

func main() {
	cfg, err := config.Load("development")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	loader, err := testfixtures.New(
		testfixtures.Database(sqlDB),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory("fixtures"),
		testfixtures.DangerousSkipTestDatabaseCheck(),
	)
	if err != nil {
		log.Fatalf("create fixtures loader: %v", err)
	}

	if err := loader.Load(); err != nil {
		log.Fatalf("load fixtures: %v", err)
	}

	log.Println("Fixtures loaded successfully")
}
