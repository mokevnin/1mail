package external

import (
	"net/http"
	"strconv"
	"time"

	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
)

type Handlers struct {
	externalapi.UnimplementedHandler
	ent            *ent.Client
	bootstrapToken string
}

func NewHandlers(client *ent.Client, bootstrapToken string) *Handlers {
	return &Handlers{ent: client, bootstrapToken: bootstrapToken}
}

var _ externalapi.Handler = (*Handlers)(nil)

// problem builds a ProblemDetails value for the given status code and detail.
func problem(code int, detail string) externalapi.ProblemDetails {
	return externalapi.ProblemDetails{
		Status: externalapi.NewOptInt32(int32(code)),
		Title:  externalapi.NewOptString(http.StatusText(code)),
		Detail: externalapi.NewOptString(detail),
	}
}

// optNilStringFromPtr maps a *string into an OptNilString (unset when nil).
func optNilStringFromPtr(v *string) externalapi.OptNilString {
	if v == nil {
		return externalapi.OptNilString{}
	}
	return externalapi.NewOptNilString(*v)
}

// optNilTimeZoneFromPtr maps a *string into an OptNilTimeZoneName (unset when nil).
func optNilTimeZoneFromPtr(v *string) externalapi.OptNilTimeZoneName {
	if v == nil {
		return externalapi.OptNilTimeZoneName{}
	}
	return externalapi.NewOptNilTimeZoneName(externalapi.TimeZoneName(*v))
}

// optNilTimestampFromPtr maps a *time.Time into an OptNilTimestamp (unset when nil).
func optNilTimestampFromPtr(v *time.Time) externalapi.OptNilTimestamp {
	if v == nil {
		return externalapi.OptNilTimestamp{}
	}
	return externalapi.NewOptNilTimestamp(externalapi.Timestamp(*v))
}

// parseEntityID parses an EntityId into an int64.
func parseEntityID(id externalapi.EntityId) (int64, error) {
	return strconv.ParseInt(string(id), 10, 64)
}

// apiTokenInfo maps an ent.ApiToken into an externalapi.ApiTokenInfo.
func apiTokenInfo(t *ent.ApiToken) externalapi.ApiTokenInfo {
	scopes := make([]externalapi.ApiTokenScope, len(t.Scopes))
	for i, s := range t.Scopes {
		scopes[i] = externalapi.ApiTokenScope(s)
	}
	return externalapi.ApiTokenInfo{
		ID:         externalapi.EntityId(strconv.FormatInt(t.ID, 10)),
		Name:       t.Name,
		Scopes:     scopes,
		ExpiresAt:  optNilTimestampFromPtr(t.ExpiresAt),
		RevokedAt:  optNilTimestampFromPtr(t.RevokedAt),
		LastUsedAt: optNilTimestampFromPtr(t.LastUsedAt),
		CreatedAt:  externalapi.Timestamp(t.CreatedAt),
		UpdatedAt:  externalapi.Timestamp(t.UpdatedAt),
	}
}

// contactResource maps an ent.Contact into an externalapi.ContactResource.
func contactResource(c *ent.Contact) externalapi.ContactResource {
	res := externalapi.ContactResource{
		ID:        externalapi.EntityId(strconv.FormatInt(c.ID, 10)),
		Email:     externalapi.EmailAddress(c.Email),
		FirstName: optNilStringFromPtr(c.FirstName),
		LastName:  optNilStringFromPtr(c.LastName),
		TimeZone:  optNilTimeZoneFromPtr(c.TimeZone),
		Status:    externalapi.ContactStatus(string(c.Status)),
		CreatedAt: externalapi.Timestamp(c.CreatedAt),
		UpdatedAt: externalapi.Timestamp(c.UpdatedAt),
	}
	if c.CustomFields != nil {
		res.CustomFields = externalapi.NewOptNilContactResourceCustomFields(
			externalapi.ContactResourceCustomFields(c.CustomFields),
		)
	}
	return res
}
