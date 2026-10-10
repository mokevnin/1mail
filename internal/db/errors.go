package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const pgUniqueViolation = "23505"

// IsUniqueViolation reports whether err is a Postgres unique-constraint violation.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}
