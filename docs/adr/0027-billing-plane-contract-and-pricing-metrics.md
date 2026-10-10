---
status: accepted
---

# Billing plane contract: pricing on sends, live usage, pull over the Operator surface

ADR 0009 fixed the boundary (the core measures, an external plane prices). This ADR fixes what
crosses it and what the plane charges for. Evidence is in `docs/research/billing-provider-options.md`
and `docs/research/billing-enforcement-prior-art.md`.

## Decisions

- **Price on `emails_sent`; contacts are unlimited.** A base subscription with an included send
  volume and metered overage. `contacts` is still snapshotted (a high-water-mark of identified
  Contacts, ADR 0009) but is not a price dimension. In v1 every Workspace sends through its own
  Integration (SES/SMTP); 1mail does not resell delivery, so the price is for the software only.
  Managed sending, if it comes, is a separate paid add-on.
- **Billing period is the calendar month in UTC, closed by the core**, so the Usage snapshot is
  final before the plane invoices it. Stripe may not bill usage that arrives after an invoice is
  finalized (grace period at most 72 hours).
- **A third read, Live usage**: the open period's figures with a `final` flag and `computed_at`,
  served from a worker or cache, counting in-flight sends. The plane polls it to warn at 80% and
  100% of a free plan's cap (once per period, sent by the plane).
- **The plane pulls; the core never calls it.** The plane reads Usage snapshot and Live usage and
  writes Billing hold through the **Operator surface** (ADR 0008) with a service credential, not
  through a workspace-scoped `/api` token. All of it is EE/SaaS; a self-hosted install has none of
  these endpoints.
- **The plane decides; the core holds no limits.** No plan, price or limit enters the core. The
  plane sets a Billing hold only when dunning has ended (Stripe `unpaid`/`canceled`; `past_due`
  counts as active) or a free plan's hard cap is reached. A paid plan over its included volume is
  billed, never held. Trial end downgrades to the free plan and sets no hold.
- **A hold pauses, it does not fail.** Queued mail waits and is sent when the hold lifts. Each
  freeze cause is stored separately.
- **Contacts get no cap in v1.** Prior art blocks stock limits at creation, not by freezing sends;
  sends and suspension (ADR 0007) bound abuse. A numeric-limit contract would be a new ADR.
- **Where it lives.** The plane, pricing page, Checkout, Customer Portal, Stripe webhooks and the
  Workspace-to-Stripe-customer mapping live in the closed site repository, not here. This repository
  gains no `stripe-go` dependency and no Stripe customer id. The app only shows a hold banner and a
  link to the plane.

## Considered options

- **Price by contact tiers** (rejected): it is the incumbents' model (Mailchimp, Drip, MailerLite,
  Kit); it punishes list growth and rewards nothing a self-hoster does not already get free.
- **Core enforces plan limits and sets holds itself** (rejected): puts prices and limits in the
  AGPL binary, against ADR 0009.
- **Core pushes usage to the plane by webhook** (rejected): the core would need the plane's address
  and delivery guarantees; Notifuse's signed pull is the simpler, verified pattern.
- **Reuse `/api` tokens or a fifth surface for the plane** (rejected): `/api` is Workspace-scoped
  while a hold is set across Workspaces; the Operator surface already exists for exactly that.
- **A grace period in the core** (rejected): every peer waits for the provider's terminal state
  (Plausible, Twenty, Documenso, Formbricks), so the plane owns dunning.

## Consequences

- The core gains Billing period, Live usage and the stored-per-cause freeze check; nothing else
  money-related.
- A lower price than Keila or Plunk is not claimed: peers already price on sends with unlimited
  contacts, so the edge is the Integration model and the open core, not the metric.
- Provider choice (Stripe Billing + Stripe Tax) and the operator's legal entity are the plane's
  concern and are open until the entity exists; the contracts above do not depend on them.
