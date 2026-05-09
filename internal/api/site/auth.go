package site

import (
	"context"
	"strconv"
	"strings"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/api/problems"
	"github.com/mokevnin/1mail/internal/pubsub"
	"github.com/mokevnin/1mail/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handlers) SiteAuthRegister(ctx context.Context, req siteapi.SiteAuthRegisterRequestObject) (siteapi.SiteAuthRegisterResponseObject, error) {
	name := strings.TrimSpace(req.Body.Name)
	email := strings.TrimSpace(string(req.Body.Email))
	password := req.Body.Password

	if name == "" || email == "" || password == "" {
		errors := problems.FieldErrors{}
		if name == "" {
			errors["name"] = []string{"name is required"}
		}
		if email == "" {
			errors["email"] = []string{"email is required"}
		}
		if password == "" {
			errors["password"] = []string{"password is required"}
		}
		return siteapi.SiteAuthRegister422ApplicationProblemPlusJSONResponse(problems.UnprocessableWithErrors("name, email and password are required", errors).Site()), nil
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
		return siteapi.SiteAuthRegister409ApplicationProblemPlusJSONResponse(problems.ConflictWithErrors("email already exists", problems.FieldErrors{
			"email": {"email already exists"},
		}).Site()), nil
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

func (h *Handlers) SiteAuthDirectLogin(_ context.Context, _ siteapi.SiteAuthDirectLoginRequestObject) (siteapi.SiteAuthDirectLoginResponseObject, error) {
	return siteapi.SiteAuthDirectLogin400ApplicationProblemPlusJSONResponse(problems.BadRequest("direct login is handled by auth provider").Site()), nil
}
