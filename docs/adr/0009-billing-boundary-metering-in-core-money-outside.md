---
status: accepted
---

# Billing boundary: metering in core, money in an external plane

For the planned SaaS offering the core **measures** billable activity but never **prices** it.
Rating, plans, invoices, payment, taxes, and dunning live in a separate billing control-plane
built on a ready provider (Stripe / Lago / Paddle — provider left open); they never enter the
AGPL binary or the core glossary. The core exposes two thin contracts to that plane: it hands
out a finalized **Usage snapshot** (read) and accepts a **Billing hold** (write). This keeps the
open core honest (a self-hoster sees no billing concepts at all) and puts the payment/tax/dunning
complexity behind a managed provider rather than hand-rolled in Go.

This is deliberately scoped to **SaaS subscription billing**. It is unrelated to the **EE license
key** (ADR-defined elsewhere: a runtime key unlocks `ee/` in the same binary); the two are
separate monetization mechanisms and are not unified into one "entitlement" abstraction.

## What the core does

- **Usage snapshot** (in `ee/`): a per-(Workspace, billing period, metric) materialized aggregate,
  finalized (immutable) at period close — the billing-grade figure the plane consumes. It
  materializes the raw Event stream the way a Broadcast recipient's rollup materializes engagement:
  the Events stay the source of truth for audit/dispute, the snapshot is the closed, reproducible
  number. The aggregator runs **in the core process** because that is the only place the raw
  Events (and their retention) live; the *billing-period / finalization* concept is EE, invisible
  to an unlicensed self-hoster.
- **v1 metrics**: `emails_sent` (sum over `email.sent`) and `contacts` (high-water-mark — the peak
  during the period, unrecoverable after the fact if not snapshotted live). Events-ingested, seats,
  and any "active/messaged contacts" metric are deferred until their pricing shape is real.
- **Billing hold**: a reversible Workspace freeze for a *money* reason (non-payment / plan-limit
  breach). It reuses the **same core send chokepoint** as Workspace suspension (ADR 0007) but is a
  **distinct, independent cause** — the send path asks one question ("may this Workspace send
  now?") answered by several freeze reasons, never a repurposing of suspension. It freezes all
  three surfaces (Broadcast, Automation, Transactional) and engages **only after a dunning grace
  period**; it clears by payment (self-service), where suspension clears by appeal. The plane sets
  and lifts it via API; only the freeze check lives in core.

## Considered options

- **Bill by re-scanning the raw Event log live at invoice time** (rejected): not reproducible if
  Events are pruned, expensive per cycle, and not tamper-evident. Every metering engine
  (Stripe Meters, OpenMeter, Metronome, Lago) pre-aggregates into finalized windows instead.
- **Stream raw metering Events to the external plane and aggregate there** (rejected): forces the
  plane to have reliable access to every tenant's full Event stream and duplicates raw-event
  retention/audit on a second side. The core already has the data; aggregate where the data is.
- **Billing engine inside the EE binary, talking to Stripe directly** (rejected): compiles
  payment/tax logic and provider secrets into the artifact shipped to everyone, and couples the
  operator's monetization lifecycle to the product process.
- **Overload Workspace suspension for non-payment** (rejected): breaks the deliberate glossary
  line "suspension is not a billing state" (ADR 0007). A billing freeze and an abuse freeze have
  different lifecycles (payment vs appeal) and messaging; they are two causes on one chokepoint.
- **Freeze only marketing on non-payment, spare transactional** (rejected): leaks billing logic
  into the send path by message type and gives a non-payer a permanent loophole. The grace period,
  not the message type, is what protects a tenant's password-reset mail.
- **Unify EE license key and SaaS subscription into one "entitlement"** (rejected): premature
  abstraction — an offline runtime key and an external money contract have different lifecycles and
  failure modes.

## Consequences

- Self-hosted deployments see **none** of this: no plans, no Billing hold, no Usage snapshot
  (an EE concept). The AGPL core gains only a generalized "is sending frozen, and why" check that
  already existed for suspension.
- The core↔plane coupling is exactly two contracts: read `Usage snapshot`, write `Billing hold`.
  No prices, currencies, or tax rules cross into the core.
- The billing control-plane is a separate service/repo; picking the provider (Stripe managed vs
  Lago self-hosted) is deferred to implementation.
- Pricing model is not fixed here — the core snapshots both `emails_sent` and `contacts` so the
  plane can charge on either (or both) without a core change.
