package site_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/mokevnin/1mail/ent/membership"
	entuser "github.com/mokevnin/1mail/ent/user"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addMember creates a real User with the given email and joins them to workspace
// 1 (Acme) with the given role, returning the new membership id. Inline (txdb
// rolls back) — a one-off actor for a specific access scenario.
func addMember(t *testing.T, env *testhelper.TestEnv, email string, role membership.Role) int64 {
	t.Helper()
	ctx := context.Background()
	u, err := env.DB.User.Create().SetName(email).SetEmail(email).Save(ctx)
	require.NoError(t, err)
	m, err := env.DB.Membership.Create().SetUserID(u.ID).SetWorkspaceID(1).SetRole(role).Save(ctx)
	require.NoError(t, err)
	return m.ID
}

func TestSiteInvitationsCreateAndList(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com") // owner
	ctx := context.Background()

	res, err := c.SiteInvitationsCreate(ctx,
		&siteapi.SiteCreateInvitationInput{Email: "newbie@acme.test", Role: siteapi.SiteInvitableRoleMember},
		siteapi.SiteInvitationsCreateParams{Slug: "acme"})
	require.NoError(t, err)
	created, ok := res.(*siteapi.SiteCreateInvitationResponse)
	require.Truef(t, ok, "got %T", res)
	assert.Contains(t, created.InviteUrl, "/invitations/", "copy-link is returned")
	assert.Equal(t, siteapi.EmailAddress("newbie@acme.test"), created.Resource.Email)

	list, err := c.SiteInvitationsList(ctx, siteapi.SiteInvitationsListParams{Slug: "acme"})
	require.NoError(t, err)
	items, ok := list.(*siteapi.SiteInvitationsListOKApplicationJSON)
	require.Truef(t, ok, "got %T", list)
	emails := make([]string, 0, len(*items))
	for _, inv := range *items {
		emails = append(emails, string(inv.Email))
	}
	assert.Contains(t, emails, "newbie@acme.test")
}

func TestSiteInvitationsForbiddenForMember(t *testing.T) {
	env := testhelper.Setup(t)
	addMember(t, env, "plainmember@acme.test", membership.RoleMember)
	c := siteClient(t, env, "plainmember@acme.test")

	res, err := c.SiteInvitationsCreate(context.Background(),
		&siteapi.SiteCreateInvitationInput{Email: "x@acme.test", Role: siteapi.SiteInvitableRoleMember},
		siteapi.SiteInvitationsCreateParams{Slug: "acme"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteInvitationsCreateForbidden{}, res)
}

func TestSiteInvitationAcceptNewUser(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	// The fixture invite (raw token below) targets an email with no account yet.
	lookup, err := c.SitePublicInvitationsLookup(ctx, siteapi.SitePublicInvitationsLookupParams{Token: "inv_fixture_token_acme"})
	require.NoError(t, err)
	found, ok := lookup.(*siteapi.SiteInvitationLookupResult)
	require.Truef(t, ok, "got %T", lookup)
	assert.Equal(t, "Acme", found.WorkspaceName)
	assert.False(t, found.HasAccount)

	accept, err := c.SitePublicInvitationsAccept(ctx,
		&siteapi.SiteAcceptInvitationInput{Name: siteapi.NewOptString("New Bie"), Password: siteapi.NewOptString("s3cret-pass")},
		siteapi.SitePublicInvitationsAcceptParams{Token: "inv_fixture_token_acme"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SitePublicInvitationsAcceptOK{}, accept)

	// A verified User now exists and is a member of Acme.
	u, err := env.DB.User.Query().Where(entuser.Email("invited@acme.test")).Only(ctx)
	require.NoError(t, err)
	assert.NotNil(t, u.EmailVerifiedAt, "invite-link acceptance verifies the email")
	m, err := env.DB.Membership.Query().Where(membership.UserID(u.ID), membership.WorkspaceID(1)).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, membership.RoleMember, m.Role)

	// The invite is consumed — no longer pending.
	list, err := c.SiteInvitationsList(ctx, siteapi.SiteInvitationsListParams{Slug: "acme"})
	require.NoError(t, err)
	items := list.(*siteapi.SiteInvitationsListOKApplicationJSON)
	for _, inv := range *items {
		assert.NotEqual(t, siteapi.EmailAddress("invited@acme.test"), inv.Email)
	}
}

func TestSiteInvitationAcceptExistingUser(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	// An account exists but is not yet a member.
	existing, err := env.DB.User.Create().SetName("Ext").SetEmail("existing@acme.test").Save(ctx)
	require.NoError(t, err)

	res, err := c.SiteInvitationsCreate(ctx,
		&siteapi.SiteCreateInvitationInput{Email: "existing@acme.test", Role: siteapi.SiteInvitableRoleAdmin},
		siteapi.SiteInvitationsCreateParams{Slug: "acme"})
	require.NoError(t, err)
	created := res.(*siteapi.SiteCreateInvitationResponse)
	token := created.InviteUrl[strings.LastIndex(created.InviteUrl, "/")+1:]

	// No name/password needed — the account already exists.
	accept, err := c.SitePublicInvitationsAccept(ctx, &siteapi.SiteAcceptInvitationInput{}, siteapi.SitePublicInvitationsAcceptParams{Token: token})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SitePublicInvitationsAcceptOK{}, accept)

	m, err := env.DB.Membership.Query().Where(membership.UserID(existing.ID), membership.WorkspaceID(1)).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, membership.RoleAdmin, m.Role)
}

func TestSiteInvitationExpiredRejected(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	lookup, err := c.SitePublicInvitationsLookup(ctx, siteapi.SitePublicInvitationsLookupParams{Token: "inv_fixture_token_expired"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.ProblemDetails{}, lookup)

	accept, err := c.SitePublicInvitationsAccept(ctx, &siteapi.SiteAcceptInvitationInput{}, siteapi.SitePublicInvitationsAcceptParams{Token: "inv_fixture_token_expired"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SitePublicInvitationsAcceptNotFound{}, accept)
}

func TestSiteMembershipsList(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	list, err := c.SiteMembershipsList(context.Background(), siteapi.SiteMembershipsListParams{Slug: "acme"})
	require.NoError(t, err)
	items, ok := list.(*siteapi.SiteMembershipsListOKApplicationJSON)
	require.Truef(t, ok, "got %T", list)
	require.Len(t, *items, 1)
	assert.Equal(t, siteapi.SiteMembershipRoleOwner, (*items)[0].Role)
	assert.Equal(t, siteapi.EmailAddress("info@1mail.com"), (*items)[0].Email)
}

func TestSiteMembershipsLastOwnerGuard(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	// membership id 1 is the sole owner (fixture).
	del, err := c.SiteMembershipsDelete(ctx, siteapi.SiteMembershipsDeleteParams{Slug: "acme", ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteMembershipsDeleteUnprocessableEntity{}, del)

	upd, err := c.SiteMembershipsUpdate(ctx,
		&siteapi.SiteUpdateMembershipInput{Role: siteapi.SiteMembershipRoleMember},
		siteapi.SiteMembershipsUpdateParams{Slug: "acme", ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteMembershipsUpdateUnprocessableEntity{}, upd)
}

func TestSiteMembershipsRoleManagement(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	memberID := addMember(t, env, "promote@acme.test", membership.RoleMember)

	// Owner promotes a member to admin.
	owner := siteClient(t, env, "info@1mail.com")
	upd, err := owner.SiteMembershipsUpdate(ctx,
		&siteapi.SiteUpdateMembershipInput{Role: siteapi.SiteMembershipRoleAdmin},
		siteapi.SiteMembershipsUpdateParams{Slug: "acme", ID: idStr(memberID)})
	require.NoError(t, err)
	got, ok := upd.(*siteapi.SiteMembershipResource)
	require.Truef(t, ok, "got %T", upd)
	assert.Equal(t, siteapi.SiteMembershipRoleAdmin, got.Role)

	// A plain member cannot manage roles.
	addMember(t, env, "nosy@acme.test", membership.RoleMember)
	member := siteClient(t, env, "nosy@acme.test")
	forbidden, err := member.SiteMembershipsUpdate(ctx,
		&siteapi.SiteUpdateMembershipInput{Role: siteapi.SiteMembershipRoleAdmin},
		siteapi.SiteMembershipsUpdateParams{Slug: "acme", ID: idStr(memberID)})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteMembershipsUpdateForbidden{}, forbidden)
}

// An admin manages ordinary members but may not touch an owner — only an owner
// can modify or remove another owner.
func TestSiteMembershipsAdminCannotTouchOwner(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	addMember(t, env, "boss@acme.test", membership.RoleAdmin)
	targetID := addMember(t, env, "grunt@acme.test", membership.RoleMember)

	admin := siteClient(t, env, "boss@acme.test")

	// Admin can promote an ordinary member.
	upd, err := admin.SiteMembershipsUpdate(ctx,
		&siteapi.SiteUpdateMembershipInput{Role: siteapi.SiteMembershipRoleAdmin},
		siteapi.SiteMembershipsUpdateParams{Slug: "acme", ID: idStr(targetID)})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteMembershipResource{}, upd)

	// Admin cannot demote the owner (membership id 1)...
	demote, err := admin.SiteMembershipsUpdate(ctx,
		&siteapi.SiteUpdateMembershipInput{Role: siteapi.SiteMembershipRoleMember},
		siteapi.SiteMembershipsUpdateParams{Slug: "acme", ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteMembershipsUpdateForbidden{}, demote)

	// ...nor remove them.
	del, err := admin.SiteMembershipsDelete(ctx, siteapi.SiteMembershipsDeleteParams{Slug: "acme", ID: "1"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteMembershipsDeleteForbidden{}, del)
}

func idStr(id int64) siteapi.EntityId {
	return siteapi.EntityId(strconv.FormatInt(id, 10))
}
