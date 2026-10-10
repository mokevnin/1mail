package site_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/contact"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// Erasure is irreversible, so a plain member is refused and nothing is touched.
func TestSiteContactsEraseIsRefusedForMembers(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	c := env.SiteActor(t, fixtures.MemberMaryEmail)

	res, err := c.SiteContactsDelete(ctx, siteapi.SiteContactsDeleteParams{Slug: fixtures.AcmeSlug, ID: idStr(fixtures.ContactBobID)})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteContactsDeleteForbidden{}, res)

	still, err := env.DB.Contact.Query().Where(contact.ID(fixtures.ContactBobID)).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, still)
}

// An owner's delete is full Erasure: the Contact is gone and its delivery rows are
// anonymized, not left behind.
func TestSiteContactsDeleteByOwnerErases(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	c := env.SiteActor(t, fixtures.OwnerJohnEmail)

	res, err := c.SiteContactsDelete(ctx, siteapi.SiteContactsDeleteParams{Slug: fixtures.AcmeSlug, ID: idStr(fixtures.ContactLiamID)})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteContactsDeleteNoContent{}, res)

	msg, err := env.DB.OutboundMessage.Get(ctx, 100)
	require.NoError(t, err)
	assert.Empty(t, msg.Destination, "the delivery record no longer carries the address")
	assert.Nil(t, msg.ContactID)
}
