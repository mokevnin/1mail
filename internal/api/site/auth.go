package site

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/pubsub"
	"github.com/mokevnin/1mail/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handlers) SiteAuthRegister(ctx context.Context, req *siteapi.SiteRegisterInput) (siteapi.SiteAuthRegisterRes, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(string(req.Email))
	password := req.Password

	if name == "" || email == "" || password == "" {
		fieldErrors := map[string][]string{}
		if name == "" {
			fieldErrors["name"] = []string{"name is required"}
		}
		if email == "" {
			fieldErrors["email"] = []string{"email is required"}
		}
		if password == "" {
			fieldErrors["password"] = []string{"password is required"}
		}
		v := siteapi.SiteAuthRegisterUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity,
			"name, email and password are required",
			fieldErrors,
		))
		return &v, nil
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
		v := siteapi.SiteAuthRegisterConflict(problemWithErrors(
			http.StatusConflict,
			"email already exists",
			map[string][]string{"email": {"email already exists"}},
		))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	if _, err := h.createDefaultWorkspace(ctx, u.ID, name); err != nil {
		return nil, err
	}

	_ = pubsub.Publish(h.pubsub, pubsub.TopicUserRegistered, pubsub.UserRegisteredEvent{
		UserID: u.ID,
		Name:   u.Name,
		Email:  u.Email,
	})

	return &siteapi.SiteRegisterResult{
		ID:        siteapi.EntityId(strconv.FormatInt(u.ID, 10)),
		Name:      u.Name,
		Email:     siteapi.EmailAddress(u.Email),
		CreatedAt: siteapi.Timestamp(u.CreatedAt),
	}, nil
}

func (h *Handlers) SiteAuthDirectLogin(_ context.Context, _ *siteapi.SiteDirectLoginInput) (siteapi.SiteAuthDirectLoginRes, error) {
	v := problem(http.StatusBadRequest, "direct login is handled by auth provider")
	return &v, nil
}
