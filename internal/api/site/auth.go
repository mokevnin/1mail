package site

import (
	"context"
	"strconv"
	"strings"

	"github.com/mokevnin/1mail/ent"
	entuser "github.com/mokevnin/1mail/ent/user"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/pubsub"
	"github.com/mokevnin/1mail/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handlers) SiteAuthRegister(ctx context.Context, req siteapi.SiteAuthRegisterRequestObject) (siteapi.SiteAuthRegisterResponseObject, error) {
	name := strings.TrimSpace(req.Body.Name)
	email := strings.TrimSpace(string(req.Body.Email))
	password := req.Body.Password

	if name == "" || email == "" || password == "" {
		return siteapi.SiteAuthRegister422ApplicationProblemPlusJSONResponse(unprocessable("name, email and password are required")), nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	u, err := h.ent.User.Create().
		SetName(name).
		SetEmail(email).
		SetPasswordHash(string(hash)).
		Save(ctx)
	if service.IsUniqueViolation(err) {
		return siteapi.SiteAuthRegister409ApplicationProblemPlusJSONResponse(conflict("email already exists")), nil
	}
	if err != nil {
		return nil, err
	}

	_ = pubsub.Publish(h.pubsub, pubsub.TopicUserRegistered, pubsub.UserRegisteredEvent{
		UserID: u.ID,
		Name:   u.Name,
		Email:  u.Email,
	})

	return siteapi.SiteAuthRegister201JSONResponse{
		Id:        strconv.FormatInt(u.ID, 10),
		Name:      u.Name,
		Email:     siteapi.EmailAddress(u.Email),
		CreatedAt: u.CreatedAt,
	}, nil
}

func unprocessable(detail string) siteapi.ProblemDetails {
	return problem(422, "Unprocessable Entity", detail)
}

// CredChecker validates email/password against the users table for go-pkgz/auth.
type CredChecker struct {
	ent *ent.Client
}

func NewCredChecker(client *ent.Client) *CredChecker {
	return &CredChecker{ent: client}
}

func (c *CredChecker) Check(user, password string) (bool, error) {
	u, err := c.ent.User.Query().Where(entuser.Email(user)).Only(context.Background())
	if ent.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if u.PasswordHash == "" {
		return false, nil
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil, nil
}
