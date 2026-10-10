package accounts_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/membership"
	"github.com/mokevnin/1mail/internal/accounts"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

func TestRoleDecisionsAreNamedAndSeparate(t *testing.T) {
	cases := []struct {
		role                   membership.Role
		members, tokens, erase bool
	}{
		{membership.RoleOwner, true, true, true},
		{membership.RoleAdmin, true, true, true},
		{membership.RoleMember, false, false, false},
	}
	for _, c := range cases {
		assert.Equal(t, c.members, accounts.CanManageMembers(c.role), "manage members: %s", c.role)
		assert.Equal(t, c.tokens, accounts.CanManageTokens(c.role), "manage tokens: %s", c.role)
		assert.Equal(t, c.erase, accounts.CanErase(c.role), "erase: %s", c.role)
	}
}

func TestChangeMembershipRoleInvariants(t *testing.T) {
	env := testhelper.Setup(t)
	acc := accounts.New(env.DB, env.Bus)
	ctx := context.Background()
	s := env.DB.Scoped(fixtures.InitechID)
	actor := events.Actor{Kind: events.ActorUser}

	roleOf := func(id int64) membership.Role {
		return env.DB.Membership.GetX(ctx, id).Role
	}

	t.Run("an admin cannot grant the owner Role", func(t *testing.T) {
		_, err := acc.ChangeMembershipRole(ctx, s, actor, membership.RoleAdmin, fixtures.InitechMemberMembershipID, membership.RoleOwner)
		require.ErrorIs(t, err, accounts.ErrOwnerOnly)
		assert.Equal(t, membership.RoleMember, roleOf(fixtures.InitechMemberMembershipID))
	})

	t.Run("an admin cannot alter an owner", func(t *testing.T) {
		_, err := acc.ChangeMembershipRole(ctx, s, actor, membership.RoleAdmin, fixtures.InitechCoOwnerMembershipID, membership.RoleMember)
		require.ErrorIs(t, err, accounts.ErrOwnerOnly)
		assert.Equal(t, membership.RoleOwner, roleOf(fixtures.InitechCoOwnerMembershipID))
	})

	t.Run("an owner grants the owner Role", func(t *testing.T) {
		m, err := acc.ChangeMembershipRole(ctx, s, actor, membership.RoleOwner, fixtures.InitechAdminMembershipID, membership.RoleOwner)
		require.NoError(t, err)
		assert.Equal(t, membership.RoleOwner, m.Role)
		require.NotNil(t, m.Edges.User, "the User edge comes loaded for the response")
		assert.Equal(t, fixtures.InitechAdminAdaName, m.Edges.User.Name)
	})

	t.Run("a co-owner can be demoted while another owner remains", func(t *testing.T) {
		m, err := acc.ChangeMembershipRole(ctx, s, actor, membership.RoleOwner, fixtures.InitechCoOwnerMembershipID, membership.RoleAdmin)
		require.NoError(t, err)
		assert.Equal(t, membership.RoleAdmin, m.Role)
	})

	t.Run("the last owner cannot be demoted", func(t *testing.T) {
		// The admin promoted above and the owner Olga remain; demote the former, then Olga is last.
		_, err := acc.ChangeMembershipRole(ctx, s, actor, membership.RoleOwner, fixtures.InitechAdminMembershipID, membership.RoleMember)
		require.NoError(t, err)
		_, err = acc.ChangeMembershipRole(ctx, s, actor, membership.RoleOwner, fixtures.InitechOwnerMembershipID, membership.RoleAdmin)
		require.ErrorIs(t, err, accounts.ErrLastOwner)
		assert.Equal(t, membership.RoleOwner, roleOf(fixtures.InitechOwnerMembershipID))
	})

	t.Run("a Membership of another Workspace is not found", func(t *testing.T) {
		_, err := acc.ChangeMembershipRole(ctx, s, actor, membership.RoleOwner, fixtures.GlobexOwnerMembershipID, membership.RoleMember)
		require.Error(t, err)
		assert.True(t, ent.IsNotFound(err))
	})
}

func TestRemoveMembershipInvariants(t *testing.T) {
	env := testhelper.Setup(t)
	acc := accounts.New(env.DB, env.Bus)
	ctx := context.Background()
	s := env.DB.Scoped(fixtures.InitechID)

	exists := func(id int64) bool {
		ok, err := env.DB.Membership.Query().Where(membership.ID(id)).Exist(ctx)
		require.NoError(t, err)
		return ok
	}

	t.Run("an admin cannot remove an owner", func(t *testing.T) {
		err := acc.RemoveMembership(ctx, s, membership.RoleAdmin, fixtures.InitechCoOwnerMembershipID)
		require.ErrorIs(t, err, accounts.ErrOwnerOnly)
		assert.True(t, exists(fixtures.InitechCoOwnerMembershipID))
	})

	t.Run("an admin removes a member", func(t *testing.T) {
		require.NoError(t, acc.RemoveMembership(ctx, s, membership.RoleAdmin, fixtures.InitechMemberMembershipID))
		assert.False(t, exists(fixtures.InitechMemberMembershipID))
	})

	t.Run("an owner removes a co-owner, then the last owner cannot be removed", func(t *testing.T) {
		require.NoError(t, acc.RemoveMembership(ctx, s, membership.RoleOwner, fixtures.InitechCoOwnerMembershipID))
		err := acc.RemoveMembership(ctx, s, membership.RoleOwner, fixtures.InitechOwnerMembershipID)
		require.ErrorIs(t, err, accounts.ErrLastOwner)
		assert.True(t, exists(fixtures.InitechOwnerMembershipID))
	})

	t.Run("a Membership of another Workspace is not found", func(t *testing.T) {
		err := acc.RemoveMembership(ctx, s, membership.RoleOwner, fixtures.GlobexOwnerMembershipID)
		require.Error(t, err)
		assert.True(t, ent.IsNotFound(err))
		assert.True(t, exists(fixtures.GlobexOwnerMembershipID))
	})
}
