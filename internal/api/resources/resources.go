package resources

import (
	"strconv"

	"github.com/mokevnin/1mail/ent"
	externalapi "github.com/mokevnin/1mail/gen/external"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/samber/lo"
)

func SiteContact(c *ent.Contact) siteapi.SiteContactResource {
	r := siteapi.SiteContactResource{
		Id:        strconv.FormatInt(c.ID, 10),
		Email:     siteapi.EmailAddress(c.Email),
		Status:    siteapi.SiteContactStatus(c.Status),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		FirstName: c.FirstName,
		LastName:  c.LastName,
		TimeZone:  c.TimeZone,
	}
	if c.CustomFields != nil {
		cf := map[string]string(c.CustomFields)
		r.CustomFields = &cf
	}
	return r
}

func ExternalContact(c *ent.Contact) externalapi.ContactResource {
	r := externalapi.ContactResource{
		Id:        EntityID(c.ID),
		Email:     externalapi.EmailAddress(c.Email),
		Status:    externalapi.ContactStatus(c.Status),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		FirstName: c.FirstName,
		LastName:  c.LastName,
		TimeZone:  c.TimeZone,
	}
	if c.CustomFields != nil {
		cf := map[string]string(c.CustomFields)
		r.CustomFields = &cf
	}
	return r
}

func ExternalTokenInfo(t *ent.ApiToken) externalapi.ApiTokenInfo {
	scopes := lo.Map(t.Scopes, func(s string, _ int) externalapi.ApiTokenScope {
		return externalapi.ApiTokenScope(s)
	})
	return externalapi.ApiTokenInfo{
		Id:         EntityID(t.ID),
		Name:       t.Name,
		Scopes:     scopes,
		ExpiresAt:  t.ExpiresAt,
		RevokedAt:  t.RevokedAt,
		LastUsedAt: t.LastUsedAt,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func EntityID(id int64) string {
	return strconv.FormatInt(id, 10)
}
