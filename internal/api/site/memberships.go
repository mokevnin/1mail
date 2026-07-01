package site

import (
	"context"
	"net/http"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/membership"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/i18n"
)

// membershipResource projects a Membership (with its User edge loaded) into the
// site DTO — a join view of the member's identity and role.
func membershipResource(m *ent.Membership) siteapi.SiteMembershipResource {
	u := m.Edges.User
	return siteapi.SiteMembershipResource{
		ID:        siteapi.EntityId(strconv.FormatInt(m.ID, 10)),
		UserId:    siteapi.EntityId(strconv.FormatInt(m.UserID, 10)),
		Email:     siteapi.EmailAddress(u.Email),
		Name:      u.Name,
		Role:      siteapi.SiteMembershipRole(m.Role),
		CreatedAt: siteapi.Timestamp(m.CreatedAt),
	}
}

// canManageMembers is the coarse core gate: owner and admin manage members and
// invites; member cannot. Fine-grained per-permission RBAC is an EE feature.
func canManageMembers(role membership.Role) bool {
	return role == membership.RoleOwner || role == membership.RoleAdmin
}

// SiteMembershipsList returns the workspace's members. Any member may view them.
func (h *Handlers) SiteMembershipsList(ctx context.Context, params siteapi.SiteMembershipsListParams) (siteapi.SiteMembershipsListRes, error) {
	ws, _, err := h.membershipFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := problem(http.StatusNotFound, "workspace not found")
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	members, err := h.ent.Membership.Query().
		Where(membership.WorkspaceID(ws)).
		WithUser().
		Order(ent.Asc(membership.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	items := make(siteapi.SiteMembershipsListOKApplicationJSON, len(members))
	for i, m := range members {
		items[i] = membershipResource(m)
	}
	return &items, nil
}

// SiteMembershipsUpdate changes a member's role. Owner/admin only; only an owner
// may grant the owner role, and the last owner can never be demoted.
func (h *Handlers) SiteMembershipsUpdate(ctx context.Context, req *siteapi.SiteUpdateMembershipInput, params siteapi.SiteMembershipsUpdateParams) (siteapi.SiteMembershipsUpdateRes, error) {
	ws, callerRole, err := h.membershipFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteMembershipsUpdateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	if !canManageMembers(callerRole) {
		v := siteapi.SiteMembershipsUpdateForbidden(problem(http.StatusForbidden, "insufficient role"))
		return &v, nil
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteMembershipsUpdateNotFound(problem(http.StatusNotFound, "member not found"))
		return &v, nil
	}

	target, err := h.ent.Membership.Query().
		Where(membership.ID(id), membership.WorkspaceID(ws)).
		WithUser().
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteMembershipsUpdateNotFound(problem(http.StatusNotFound, "member not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	desired := membership.Role(req.Role)
	// Only an owner may grant/transfer the owner role.
	if desired == membership.RoleOwner && callerRole != membership.RoleOwner {
		v := siteapi.SiteMembershipsUpdateForbidden(problem(http.StatusForbidden, "only an owner may grant the owner role"))
		return &v, nil
	}
	// Only an owner may modify another owner (an admin cannot demote a co-owner).
	if target.Role == membership.RoleOwner && callerRole != membership.RoleOwner {
		v := siteapi.SiteMembershipsUpdateForbidden(problem(http.StatusForbidden, "only an owner may change an owner's role"))
		return &v, nil
	}
	// The last owner cannot be demoted, or the workspace would be ownerless.
	if target.Role == membership.RoleOwner && desired != membership.RoleOwner {
		owners, err := h.ent.Membership.Query().
			Where(membership.WorkspaceID(ws), membership.RoleEQ(membership.RoleOwner)).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		if owners <= 1 {
			v := siteapi.SiteMembershipsUpdateUnprocessableEntity(problemWithErrors(
				http.StatusUnprocessableEntity,
				"cannot demote the last owner",
				map[string][]string{"role": {i18n.T("errors.keep_one_owner", nil)}},
			))
			return &v, nil
		}
	}

	updated, err := target.Update().SetRole(desired).Save(ctx)
	if err != nil {
		return nil, err
	}
	updated.Edges.User = target.Edges.User
	resource := membershipResource(updated)
	return &resource, nil
}

// SiteMembershipsDelete removes a member. Owner/admin only; the last owner cannot
// be removed.
func (h *Handlers) SiteMembershipsDelete(ctx context.Context, params siteapi.SiteMembershipsDeleteParams) (siteapi.SiteMembershipsDeleteRes, error) {
	ws, callerRole, err := h.membershipFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteMembershipsDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	if !canManageMembers(callerRole) {
		v := siteapi.SiteMembershipsDeleteForbidden(problem(http.StatusForbidden, "insufficient role"))
		return &v, nil
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteMembershipsDeleteNotFound(problem(http.StatusNotFound, "member not found"))
		return &v, nil
	}

	target, err := h.ent.Membership.Query().
		Where(membership.ID(id), membership.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteMembershipsDeleteNotFound(problem(http.StatusNotFound, "member not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	// Only an owner may remove another owner (an admin cannot remove a co-owner).
	if target.Role == membership.RoleOwner && callerRole != membership.RoleOwner {
		v := siteapi.SiteMembershipsDeleteForbidden(problem(http.StatusForbidden, "only an owner may remove an owner"))
		return &v, nil
	}

	if target.Role == membership.RoleOwner {
		owners, err := h.ent.Membership.Query().
			Where(membership.WorkspaceID(ws), membership.RoleEQ(membership.RoleOwner)).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		if owners <= 1 {
			v := siteapi.SiteMembershipsDeleteUnprocessableEntity(problemWithErrors(
				http.StatusUnprocessableEntity,
				"cannot remove the last owner",
				map[string][]string{"member": {i18n.T("errors.keep_one_owner", nil)}},
			))
			return &v, nil
		}
	}

	if err := h.ent.Membership.DeleteOneID(target.ID).Exec(ctx); err != nil {
		return nil, err
	}
	return &siteapi.SiteMembershipsDeleteNoContent{}, nil
}
