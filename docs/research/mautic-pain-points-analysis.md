# Mautic Pain-Point Analysis (for competitive positioning)

Researched July 2026 via web search across the Mautic community forums, GitHub issues,
and third-party review/hosting write-ups (G2, TrustRadius, Research.com, Autoize). Mautic
is the incumbent open-source marketing automation platform; this doc catalogues its
**most-repeated user complaints** and maps each to 1mail's current position, so the roadmap
can target the pains that are genuinely *ours to win*.

Mautic-side claims are cited to forum threads / issues. 1mail-side claims are grounded in
`CONTEXT.md` and `docs/adr/*` (the authoritative model) — where the two disagree, CONTEXT/ADRs win.

**Bottom line:** Mautic's deepest pains are *systemic* (cron architecture, upgrade friction,
heavyweight deployment, dated UI) and 1mail neutralises them **by design** — that is the core
positioning advantage. The two areas where 1mail is *not yet proven* against Mautic are
**scale** (unvalidated) and **send-rate control** (absent). On raw feature breadth Mautic is
still ahead, and that is fine at this stage.

---

## 1. Cron-everything architecture — Mautic's deepest, most-repeated pain

**Complaint.** Mautic drives segment rebuilds, campaign steps, and email sending through
staggered cron jobs. Consequences reported repeatedly:
- Partially-executed campaigns and unsent emails on an underpowered box ([Autoize misconfig
  guide](https://autoize.com/common-mautic-misconfigurations/)).
- ~12.5-minute average latency before a contact gets a system response under the standard
  staggered cron setup ([forum](https://forum.mautic.org/t/solved-cron-job-on-separate-lists-or-slow-down-the-rate-of-sending/17570)).
- "Cron jobs changing themselves" and deliverability fallout ([forum](https://forum.mautic.org/t/deliverability-issues-cron-jobs-chaning-themselves/27102)).
- Cron/queue documentation is unclear, leaving operators guessing at configuration
  ([netcore](https://netcorecloud.com/tutorials/slow-email-sending-speed-in-mautic/)).

**1mail position — solved by design. ✅**
There is no cron layer. Async work runs on **river** (Postgres-backed job queue, retries +
concurrency) and a **watermill** domain-event bus over a transactional outbox; both run
in-process (`docs/design/domain-events.md`; `internal/jobs`). Automation enrollment fires off
the domain-event bus atomically with the state change that triggered it, and wait-steps are
scheduled natively by river — no staggered cron, no "run the cron more often" tuning, no
partial-campaign failure mode from a missed tick.

## 2. Performance / scale on large contact bases

**Complaint.** Mautic degrades badly as the base grows:
- 1.5M+ contacts → back-office browsing is very slow ([#8261](https://github.com/mautic/mautic/issues/8261)).
- Opening the segment editor takes 5+ minutes on large DBs ([#9634](https://github.com/mautic/mautic/issues/9634)).
- ~1M contacts → most pages hang; general "performance issues that cause users to abandon
  Mautic" ([#3358](https://github.com/mautic/mautic/issues/3358)).
- CSV import drops from ~45 contacts/sec on a clean install to ~4/sec at 700k
  ([#2630](https://github.com/mautic/mautic/issues/2630)).
- Campaign conditions that check segment membership re-run the segment query, making them
  unusable on large/slow segments ([forum](https://forum.mautic.org/t/segment-campaign-condition-slow/25268)).

**1mail position — better architecture, but UNPROVEN. ⚠️**
Segments are **live rules compiled to SQL predicates** and membership is *never materialized*
(`CONTEXT.md` — "membership is never materialized and shifts as data changes"; `internal/segments`),
which structurally avoids Mautic's rebuild-the-segment-table pain. Backing store is Postgres,
not MySQL. **However, this is an architectural bet, not a measured result:** the live-count
preview and the send-loop audience resolution have not been load-tested at ~1M contacts. This
is exactly where Mautic's #1 death-by-scale complaint lives, so validating it is the highest-value
roadmap item (see backlog below).

## 3. Slow / uncontrollable send rate

**Complaint.** Users report being unable to exceed ~7 emails/sec despite higher provider
capacity, and generally slow sending ([forum](https://forum.mautic.org/t/mautic-sending-email-is-too-slow/24387));
much of this is downstream of the cron architecture (§1) but also of missing rate control
([Mautic 5 throttling thread](https://forum.mautic.org/t/how-to-slow-down-email-sending-mautic-5/34302)).

**1mail position — cron cause removed, but no send-rate feature yet. ⚠️**
The cron bottleneck is gone (§1), and river gives fixed per-queue concurrency
(`internal/jobs/worker.go` — `QueueBroadcasts: {MaxWorkers: 10}`). But there is **no
configurable send-rate limit (emails/sec), no per-provider throttle, and no IP warmup ramp.**
Large sends against a rate-limited provider (SES sandbox, new IP) will need this. Net-new
roadmap item.

## 4. Painful upgrades & framework churn

**Complaint.** Upgrades are a recurring source of pain:
- 4→5 migration fails at the DB step / `apply --finish`
  ([#13281](https://github.com/mautic/mautic/issues/13281)), Doctrine "table already exists"
  on 5.0.3→5.0.4 ([#13651](https://github.com/mautic/mautic/issues/13651)), 500s after upgrade
  ([forum](https://forum.mautic.org/t/getting-500-error-after-upgrade/31813)).
- Every Symfony version hop forces custom-code rewrites, plugin rewrites, and re-buying paid
  plugins ([forum](https://forum.mautic.org/t/mautic-release-upgrade-strategy-is-misguided/35981)).
- The composer-based update path is "very badly documented."

**1mail position — solved by design. ✅**
Ships as a **single Go binary with the SPA and migrations embedded**; the only runtime
dependency is Postgres (`docs/self-hosting.md`). Migrations are managed by **Atlas**,
forward-only, applied via `./1mail migrate` (or `AUTO_MIGRATE` on a single replica). No
composer, no Symfony version hops, no plugin re-purchase — EE code lives in the same binary,
gated by a license key ([[project_open_core_ee]]). A versioned upgrade runbook + an in-app
"new release available" banner are already planned (ROADMAP → Self-hosting lifecycle); the
Mautic evidence raises their priority.

## 5. Heavyweight deployment for any real scale

**Complaint.** Hosting a large Mautic instance is recommended to involve load balancers,
containers, **Redis** for session storage, and **multi-master MySQL**
([Autoize horizontal scaling](https://autoize.com/hosting-large-instances-of-mautic-with-horizontal-scaling/)).

**1mail position — solved by design. ✅**
Self-contained binary + Postgres only. Queue (river) and pub/sub (watermill) both live *in
Postgres* — **no Redis, no S3, no separate worker process** (`docs/design/domain-events.md`,
`README.md`). Runs single-replica, or multi-replica by running the migrate step once as init.

## 6. Dated UI, clutter, no archiving

**Complaint.** UX was a significant enough pain that Mautic launched a full UI overhaul onto
the Carbon Design System from 2024 ([year-in-review](https://mautic.org/blog/mautics-year-in-review-2025/)),
and there is **no way to archive old/unused resources**, so busy instances become cluttered
and hard to navigate ([roadmap](https://mautic.org/roadmap/)).

**1mail position — modern UI ✅, but archiving is also missing ⚠️.**
Frontend is React 19 + Mantine with responsive layout and light/dark themes from day one
(`CLAUDE.md` frontend conventions) — modern out of the gate, no legacy redesign debt. But
Mautic's *clutter* complaint applies to us too: **there is no archive / soft-delete on
Broadcast, Automation, or Segment** (verified — no `archived_at`/`deleted_at`/soft-delete in
`ent/schema` or `typespec`). Net-new roadmap item.

## 7. Support / community confusion (not an engineering item)

**Complaint.** Growing confusion over where mautic.com subscribers get support
([announcement](https://mautic.org/blog/our-announcement/)). Not applicable to compare — it's
an org/governance issue, not a product capability. Noted for completeness only.

---

## Where Mautic is still ahead (honest gaps)

1mail is an earlier-stage product; on feature *breadth* Mautic leads, echoing the review
critique that Mautic "lacks features for precise/quantitative requirements or larger
companies" ([Research.com](https://research.com/software/reviews/mautic)). Specifically:

- **Automations are linear** — no branching / conditions / goal nodes. The xyflow builder can
  draw branches but drops them at save (ROADMAP Phase 4 note). *Already in the roadmap backlog.*
- **No visual email editor** — MJML body is a textarea. *Already in the roadmap (Phase 3).*
- **No onsite forms/popups.** *Already in the roadmap (Phase 5).*
- **No sending domains (DKIM/SPF), no A/B testing, few provider adapters (SMTP + SES).**
  *Already in the roadmap (Phase 6).* Note sending-domains is a *deliverability* prerequisite,
  not just a feature — it belongs above breadth items in priority.
- **SMS reserved on the schema, not implemented.** *Already noted in the roadmap.*

None of these are new discoveries — they are the known parity backlog. The Mautic lens does
not change *what* is on the roadmap here, only reinforces the *evidence* that these matter.

---

## Net-new roadmap items surfaced by this analysis

Three items are **not** already on the roadmap and are added by this analysis (see
`docs/ROADMAP.md`):

1. **Scale validation (highest value).** Load-test the segment engine (rule → SQL compilation
   + live preview count) *and* the broadcast send-loop audience resolution at ~1M contacts.
   This directly targets Mautic's #1 abandonment cause and is the one place 1mail's core
   architectural bet is currently unproven. Deliverable is a benchmark + any indexes/query
   fixes it exposes — not a vague "prove scale."
2. **Send-rate control & IP warmup.** Configurable per-provider send rate (emails/sec) and a
   warmup ramp for new sending IPs/domains. Lean on river's native rate/concurrency limiting
   rather than a hand-rolled throttler (per the roadmap's "maintained libraries" principle).
   Pairs naturally with the sending-domains (DKIM/SPF) work in Phase 6.
3. **Resource archiving.** Archive (soft-hide) old Broadcasts / Automations / Segments so busy
   workspaces stay navigable — Mautic's clutter complaint applies to us too.
