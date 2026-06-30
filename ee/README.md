# 1mail Enterprise Edition (`ee/`)

This directory holds **1mail Enterprise** features. Everything under `ee/` is
**source-available but not open source** — it is governed by [`ee/LICENSE`](./LICENSE),
*not* the AGPL-3.0 that covers the rest of the repository. See
[`../LICENSING.md`](../LICENSING.md) for the boundary.

## What lives here

Commercial, enterprise-targeted capabilities. Planned:

- **RBAC / roles** — fine-grained, role-based access control over a workspace
  (builds on the Membership model, ADR 0004).
- **SSO / SAML / OIDC** — enterprise single sign-on.

## How it ships

Enterprise code compiles into the **same single binary** as the open core. The
features stay locked until a valid **license key** is present at runtime — so we
distribute one artifact to everyone and unlock Enterprise functionality per
subscription (the Cal.com / GitLab model). There is no separate "enterprise build".

> No Enterprise feature is implemented yet — this directory and license establish
> the boundary so closed features can land without restructuring the repo later.
