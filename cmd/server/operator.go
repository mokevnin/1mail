package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/mokevnin/sphericon/ee/operator"
)

// operatorOps is what the `operator` commands need from the app. The app implements
// it (internal/app); tests use a fake.
type operatorOps interface {
	CreateOperator(ctx context.Context, email string) (password string, err error)
}

const operatorUsage = "usage: sphericon operator create <email>"

// runOperator executes `sphericon operator create <email>` (ADR 0026): the only way a
// platform Operator comes to exist, since there is no signup. It prints the one-time
// password; the Operator enrols a TOTP at first login.
func runOperator(ctx context.Context, ops operatorOps, args []string, out io.Writer) error {
	if len(args) != 2 {
		return errors.New(operatorUsage)
	}
	email := args[1]
	switch args[0] {
	case "create":
		password, err := ops.CreateOperator(ctx, email)
		switch {
		case errors.Is(err, operator.ErrDuplicate):
			return fmt.Errorf("an operator with email %q already exists; nothing changed", email)
		case errors.Is(err, operator.ErrNotLicensed):
			return errors.New("this instance has no operator license (feature `operator`)")
		case err != nil:
			return err
		}
		_, _ = fmt.Fprintf(out, "operator %q created\npassword (shown once): %s\nthe operator enrols a TOTP at first login\n", email, password)
	default:
		return fmt.Errorf("unknown operator command %q\n%s", args[0], operatorUsage)
	}
	return nil
}
