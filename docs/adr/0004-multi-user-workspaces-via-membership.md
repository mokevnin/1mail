---
status: accepted
---

# Workspaces are multi-user via Membership, not single-owner

A Workspace is reached through **Memberships** — a `User × Workspace × Role` join — rather than
owned by a single `User`. A User can belong to many Workspaces and a Workspace to many Users,
each Membership carrying a Role (owner / admin / member; exact set is a follow-up).

This replaces the single-owner model (`Workspace.user_id`). It is laid in now, before any
team feature is built, because a multi-user SaaS is on the roadmap and retrofitting access
control onto a single-owner schema is expensive: every ownership check, every "who can see
this workspace" query, and the auth layer would have to change at once. Adding the join now is
cheap on a clean rebuild.

## Consequences

- Authorization is "does this User have a Membership on this Workspace (with a sufficient
  Role)?", not "is this User the workspace owner?".
- The exact Role set and per-Role permissions are deliberately deferred; only the structural
  decision (access is role-scoped per Membership) is fixed here.
- `Workspace.user_id` is removed in favor of Membership rows.
