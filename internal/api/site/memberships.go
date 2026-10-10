package site

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/membership"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/accounts"
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

// SiteMembershipsList returns the workspace's members. Any member may view them.
func (h *Handlers) SiteMembershipsList(ctx context.Context, params siteapi.SiteMembershipsListParams) (siteapi.SiteMembershipsListRes, error) {
	s, _, err := h.scopedWithRoleFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := problem(http.StatusNotFound, "workspace not found")
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	members, err := s.Membership().Query().
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
	s, callerRole, err := h.scopedWithRoleFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteMembershipsUpdateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	if !accounts.CanManageMembers(callerRole) {
		v := siteapi.SiteMembershipsUpdateForbidden(problem(http.StatusForbidden, "insufficient role"))
		return &v, nil
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteMembershipsUpdateNotFound(problem(http.StatusNotFound, "member not found"))
		return &v, nil
	}

	updated, err := h.accounts.ChangeMembershipRole(ctx, s, h.actor(ctx), callerRole, id, membership.Role(req.Role))
	switch {
	case ent.IsNotFound(err):
		v := siteapi.SiteMembershipsUpdateNotFound(problem(http.StatusNotFound, "member not found"))
		return &v, nil
	case errors.Is(err, accounts.ErrOwnerOnly):
		v := siteapi.SiteMembershipsUpdateForbidden(problem(http.StatusForbidden, "only an owner may grant the owner role or change an owner's role"))
		return &v, nil
	case errors.Is(err, accounts.ErrLastOwner):
		v := siteapi.SiteMembershipsUpdateUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity,
			"cannot demote the last owner",
			map[string][]string{"role": {i18n.T("errors.keep_one_owner", nil)}},
		))
		return &v, nil
	case err != nil:
		return nil, err
	}
	resource := membershipResource(updated)
	return &resource, nil
}

// SiteMembershipsDelete removes a member. Owner/admin only; the last owner cannot
// be removed.
func (h *Handlers) SiteMembershipsDelete(ctx context.Context, params siteapi.SiteMembershipsDeleteParams) (siteapi.SiteMembershipsDeleteRes, error) {
	s, callerRole, err := h.scopedWithRoleFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteMembershipsDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	if !accounts.CanManageMembers(callerRole) {
		v := siteapi.SiteMembershipsDeleteForbidden(problem(http.StatusForbidden, "insufficient role"))
		return &v, nil
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteMembershipsDeleteNotFound(problem(http.StatusNotFound, "member not found"))
		return &v, nil
	}

	err = h.accounts.RemoveMembership(ctx, s, callerRole, id)
	switch {
	case ent.IsNotFound(err):
		v := siteapi.SiteMembershipsDeleteNotFound(problem(http.StatusNotFound, "member not found"))
		return &v, nil
	case errors.Is(err, accounts.ErrOwnerOnly):
		v := siteapi.SiteMembershipsDeleteForbidden(problem(http.StatusForbidden, "only an owner may remove an owner"))
		return &v, nil
	case errors.Is(err, accounts.ErrLastOwner):
		v := siteapi.SiteMembershipsDeleteUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity,
			"cannot remove the last owner",
			map[string][]string{"member": {i18n.T("errors.keep_one_owner", nil)}},
		))
		return &v, nil
	case err != nil:
		return nil, err
	}
	return &siteapi.SiteMembershipsDeleteNoContent{}, nil
}
