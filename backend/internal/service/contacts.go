package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mokevnin/1mail/backend/ent"
)

const pgUniqueViolation = "23505"

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

type CreateContactInput struct {
	Email       string
	FirstName   *string
	LastName    *string
	TimeZone    *string
	CustomFields map[string]string
}

type UpdateContactInput struct {
	FirstName    *string
	LastName     *string
	TimeZone     *string
	CustomFields map[string]string
	HasCustomFields bool
}

func CreateContact(ctx context.Context, client *ent.Client, input CreateContactInput) (*ent.Contact, error) {
	q := client.Contact.Create().
		SetEmail(input.Email).
		SetNillableFirstName(input.FirstName).
		SetNillableLastName(input.LastName).
		SetNillableTimeZone(input.TimeZone)
	if input.CustomFields != nil {
		q = q.SetCustomFields(input.CustomFields)
	}
	return q.Save(ctx)
}

func UpdateContact(ctx context.Context, client *ent.Client, id int64, input UpdateContactInput) (*ent.Contact, error) {
	q := client.Contact.UpdateOneID(id).
		SetNillableFirstName(input.FirstName).
		SetNillableLastName(input.LastName).
		SetNillableTimeZone(input.TimeZone)
	if input.HasCustomFields {
		q = q.SetCustomFields(input.CustomFields)
	}
	return q.Save(ctx)
}
