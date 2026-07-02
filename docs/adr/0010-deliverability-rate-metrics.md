---
status: accepted
---

# Deliverability rate metrics: complaint & bounce rate in core, thresholds in EE

The automated abuse detector designed in ADR 0007 (auto-suspend a sender torching the shared
reputation) has **nothing to read** until a complaint/bounce *rate* exists. This ADR defines
that rate: what it measures, where it is computed, and at what grain — the prerequisite metric,
not the detector policy on top of it.

## Decisions

### Rate computation is core and user-facing; thresholds are EE

A deliverability *number* is useful to everyone — a self-hoster on their own domain needs it to
avoid torching their own reputation — so the **rate computation lives in the AGPL core** and is
shown on the workspace dashboard. It is a *deliverability* signal, **not** a billing metric, so
it does **not** enter the EE metering plane (which per ADR 0009 *billing-boundary* carries only
billing-grade Usage snapshots). What is EE is the **auto-suspension policy** — the numeric thresholds and the
minimum-volume floor that turn a rate into a freeze decision (ADR 0007's detector). A
self-hosted workspace with one tenant does not police itself, so that policy is SaaS-only.

Clean layering: **core exposes the number, EE decides what to do with it.** This grill covers
only the rate derived from the Events we already ingest (`email.sent` / `email.bounced` /
`email.complained`). Google Postmaster / FBL (ARF) ingestion — *new* data sources — stay out of
scope, in the separate Deliverability-observability backlog item.

### Complaint rate = `complaints / (sent − hard bounces)`

The load-bearing decision. Mailbox providers (Gmail/Yahoo 2024+) define complaint rate over
**delivered** mail and expect it held **< 0.3%**. But 1mail records only `email.sent`
(*accepted by provider*, not delivered — see CONTEXT *Sent vs delivered*), and the SES→SNS hook
**discards** `Delivery` notifications. A rate over raw `sent` therefore **systematically
understates** versus the provider's own figure — and worst for exactly the senders the detector
must catch (a 20%-bounce sender delivers ≈ 0.8×sent, so `complaints/sent` reads ~20% low, quietly
making the industry 0.3% threshold too lenient).

We subtract hard bounces from the denominator as a **cheap proxy for delivered**: it needs zero
new ingestion (hard bounces are already ingested and suppress), and `(sent − hard bounces)` is a
good approximation of delivered. Ingesting true `Delivery` events (which would roughly double
SES→SNS volume for a second-decimal-place gain) is deferred to Deliverability-observability.

The `≈ delivered` claim is a *cohort* argument (of the messages sent, the non-bounced ones were
delivered) but the metric is a **flow rate** (see below): the denominator is actually
`COUNT(sent in window) − COUNT(hard bounces in window)` — two different message cohorts. This
holds while a send's bounces are still in-window (steady state / active sending) and degrades in
the tail (a blast's bounces linger after its sends age out). So the denominator is **clamped at
≥ 0**, and a ≤ 0 or tiny denominator falls below the volume floor → "insufficient data" (never
`complaints / negative`). The volume floor, not the proxy's tail precision, is what keeps the
number honest.

### Bounce rate = `permanent_bounces / sent`

The sibling metric, cleaner (you sent it, it bounced — the denominator is plain `sent`). Only
**permanent** (hard) bounces count; **transient** (soft) bounces are excluded from the headline
(at most a secondary counter), because transient bounces are temporary noise, not a reputation
signal. One hard bounce is used three coherent ways: it suppresses, it subtracts from the
complaint-rate denominator, and it is the bounce-rate numerator.

### Grain = per-(Workspace, Sending domain)

Mailbox providers attribute reputation to the **DKIM signing domain** — which in 1mail is the
**Sending domain** (ADR 0009 *sending-domains*). So the metric is grained per-(Workspace, Sending domain), not a
workspace-wide average: an average would mask one dirty domain behind a clean one — precisely the
sender the detector must catch. Per-workspace is just the roll-up of the per-domain rates for the
dashboard headline.

**Sequencing dependency:** Sending domains (ADR 0009 *sending-domains*) are not built yet, and today's delivery
Events do **not** carry the sending domain (they record the *recipient*). This metric therefore
ships **with or after** ADR 0009, and requires two ingestion changes:
- the send path stamps the Sending domain onto `email.sent`;
- the bounce/complaint hook stamps it onto the failure Event (SES puts the From address in the
  notification's `mail.source`).

Reputation isolation is achieved by **multiple Sending domains**, never by sub-dividing the
Workspace — `workspace = company` stays the sole tenant (no sub-accounts).

### Flow rate, by each event's own timestamp

Complaints and bounces arrive *after* the send (FBL complaints can lag days). The rate is a
**flow rate**: each Event (`sent` / `bounce` / `complaint`) counts by its own `occurred_at`
within the trailing window — never linked back to the originating send. This needs zero
send↔feedback correlation (the `EmailDeliveryFailure` Event carries no reference to its send
anyway), is always current, and reacts fast to a live spike — the ESP-standard, Postmaster-style
approach. The known distortion (feedback for old sends counted against newer sends) only bites
during sharp volume ramps and washes out at steady state.

### The metric is a triple, never a bare scalar

Core always exposes `(numerator, denominator, rate)` per (Workspace, Sending domain, window);
the **rate is undefined when the denominator is zero**. The dashboard shows the denominator
beside the rate ("0.4% of 12,000 sent") and "insufficient data" below a small display floor. The
EE detector applies its **own** volume floor to that same denominator — the floor *value* is EE
policy, but it is only possible because core exposes the denominator, not just the rate.

### Live query now; materialized rollup later

Unlike ADR 0009 *billing-boundary*'s Usage snapshot (immutable, reproducible, tamper-evident),
this is a **disposable health monitor** — recent and cheap matters, immutability does not. That
ADR's "no live re-scan of raw Events" rejection therefore **does not bind** here. v1 computes the rate
by a **live windowed query over `events`** (plus an index like
`(workspace_id, sending_domain, action, occurred_at)`) — no new table. Because Events are the
source of truth, a **materialized daily rollup maintained by a periodic river job** can drop in
**later** (gated on real high-volume senders) as a pure internal optimization — no re-grain, no
semantic change. Deferred until product scale demands it.

## Considered options

- **Rate over raw `sent`** (rejected): free, but systematically lenient exactly where it must not
  be (see complaint-rate reasoning).
- **Rate over ingested `Delivery` events** (deferred): provider-exact, but ~doubles SES→SNS
  volume for a marginal gain — belongs to Deliverability-observability.
- **Per-workspace-only grain** (rejected): masks a dirty domain behind a clean one; the provider
  judges the domain, so must we.
- **Sub-accounts for reputation isolation** (rejected): reputation isolation is already delivered
  by multiple Sending domains; a sub-account is tenancy-within-a-tenant (agency/reseller), a
  large separate concern with no target market today.
- **Cohort/vintage rate** (rejected): more accurate per send-cohort but reacts slower, and
  impossible without adding a send↔feedback linkage the Events lack.
- **Per-event incremental rollup** (rejected for the eventual rollup): a hot counter row per
  (workspace, domain, day) contends under exactly the high-volume senders that matter — a
  periodic batch job avoids it.
- **The metric in the EE metering plane** (rejected): it is a deliverability signal, not a
  billing figure; self-hosters need it, so it is core.

## Consequences

- Self-hosted deployments **see the numbers** (their own reputation health) but get **no**
  auto-suspension — the thresholds/floor and the detector are EE, and a single-tenant install
  does not police itself.
- The EE detector (ADR 0007) now has a defined input: the `(numerator, denominator, rate)` triple
  per (Workspace, Sending domain), over a window and denominator it chooses.
- This feature is **blocked on Sending domains** (ADR 0009 *sending-domains*) and on two ingestion changes that
  stamp the Sending domain onto the send and failure Events.
- A shared-IP-pool reputation view (the SaaS case where one tenant poisons a shared IP) is a
  future EE aggregation over this per-domain metric, deferred with the IP-pool model.
