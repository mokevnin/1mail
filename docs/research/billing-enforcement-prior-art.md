# Billing enforcement prior art: plans, usage limits and over-limit behavior in open-core SaaS

Researched October 2026 from primary sources: source code cloned at each repo's default branch
(shallow clone, commit SHAs below) and the repos' own license files. No runtime testing; everything
is static reading. Permalinks use the short commit SHA so line numbers stay valid. Every claim is
**verified** (permalink given), or explicitly marked **inference** / **not verified**. Provider
comparison (Stripe / Lago / Paddle) is out of scope: see
[billing-provider-options.md](./billing-provider-options.md).

| Repo                  | Branch  | Commit                                     | Billing code in the open repo?         |
| --------------------- | ------- | ------------------------------------------ | -------------------------------------- |
| plausible/analytics   | master  | `bc2c7b395caddf7bb4eec2548be7958824d07d4a` | Yes (grace period, locker, quota)      |
| PostHog/posthog       | master  | `1f18623dc7273e1cfd992944b848004b669af48b` | Quota limiter only; plans are external |
| pentacent/keila       | main    | `2308bebd2b54edc8ac8d484be5c9b4e972aed7cc` | Credit ledger in core; Paddle in extra |
| useplunk/plunk        | next    | `12f9f20f7c991b94063c51b7ad03c875e08d90c9` | Yes (limits, Stripe, disable)          |
| twentyhq/twenty       | main    | `2e8b29b836a366e7a6b4e068902c64d07db56928` | Yes (Stripe webhook, suspension)       |
| formbricks/formbricks | main    | `2c5c3b104bfccabc44c2460e2488eac14d8910c9` | Yes (`modules/ee/billing`)             |
| documenso/documenso   | main    | `38ecb217effcc53a7164d7123c51336f636bd3b2` | Yes (`packages/ee/server-only/limits`) |
| chatwoot/chatwoot     | develop | `e212425e7b1814614bdf8b0beea42ce689baad5c` | Yes (`enterprise/`)                    |
| Infisical/infisical   | main    | `c40d5fd5bb834ddd3cb28ce025a395a86c55dc8a` | Plan lookup + meter emit; plans remote |
| Notifuse/notifuse     | main    | `7516397cc5c1eadc2a6aa67c733261043a630a62` | Read-side usage contract only          |
| calcom/cal.com        | main    | `54343aa685ae8f33159d2f485ec4a57bad5c574a` | Schema only; logic not in the repo     |
| postalserver/postal   | main    | `d96eddbadeda600595d6247486f9ceb8e99cbc15` | No billing; send-limit mechanism       |
| getlago/lago-api      | main    | `4c4f02df9a6dba69fb28e22c67f70338656af0b9` | It is the provider (plane-side only)   |
| dittofeed/dittofeed   | main    | `fe89657bbaad3bf25fa3571288287022f12a8347` | No                                     |
| knadh/listmonk        | master  | `70ae5162f3bf4d435facc7f33d989361602fd7db` | No                                     |
| mautic/mautic         | 7.x     | `bc46b64883255608e5b14a41987c207de9535616` | No                                     |

Base URL form: `https://github.com/<org>/<repo>/blob/<short-sha>/<path>#L<n>` (short SHAs are the
first 7 characters of the commits above). Branch names are the default branch at clone time; only
the SHA matters.

## Question

ADR 0009 fixes the boundary for 1mail's SaaS: the core meters (a finalized **Usage snapshot**: `emails_sent`
sum, `contacts` high-water-mark, per Workspace and period) and accepts a **Billing hold** (a reversible
freeze for non-payment or plan-limit breach, engaged only after a dunning grace period, same chokepoint
as suspension in ADR 0007 but an independent cause). Plans, prices, invoices and dunning live in an external
plane. How do mature open-source products implement plans, usage limits and over-limit behavior for their
hosted offering, and how do they keep it out of the self-hosted build? This note reports facts, then
labels the synthesis and recommendations as inference. It does not change ADR 0009.

## Summary of findings

Verified facts:

- **Plans live in one of four places.** An external billing service the open code only caches (PostHog,
  Infisical), the payment provider itself with a mirror in the app DB (Formbricks `billing.limits`,
  Chatwoot `InstallationConfig` + `Account.limits`, Plausible static `priv/plans_v*.json` + Paddle ids),
  per-tenant environment variables pushed by a control plane (Notifuse `PLAN_MAX_*`), or a prepaid ledger
  with no "plan" concept in core (Keila credits).
- **Metering is computed live from the product tables far more often than it is snapshotted.** Plausible
  sums pageviews + custom events over billing-cycle windows from ClickHouse; Formbricks and Documenso
  `COUNT(*)` rows; Plunk counts `Email` rows with a 5-minute Redis cache; Postal reads a rolling
  60-minute live-stats window. Only Notifuse stores a monthly snapshot (`monthly_usage`, recomputed by a
  worker, exposed with `computed_at`). Cal.com's schema has seat high-water-mark columns but the writer is
  not in the repo.
- **Over-limit on a paid plan is almost never an instant hard stop.** Plausible: two consecutive cycles
  over the limit plus a 10% margin, then a **7-day grace**, then dashboard lock, while **ingestion keeps
  running**. Formbricks: overage billed via Stripe meter events, "no hard cap". PostHog: trust-score grace
  of 0 to 5 days, then events dropped at capture until the period ends. The hard cap is reserved for the
  **free tier** (Plunk 1000 emails/month, Documenso 5 documents/month) and for customer-set spend caps
  (Plunk `billingLimit*`).
- **Non-payment is almost always handled at the provider's terminal state, not on the first failed
  charge.** Plausible treats `past_due` as active. Twenty suspends only on Stripe `canceled`/`unpaid`.
  Documenso keeps `PAST_DUE` fully functional and zeroes limits only at `INACTIVE`. Keila simply stops
  refilling credits. Plunk is the outlier: it disables the project on the first `invoice.payment_failed`
  of a recurring invoice.
- **Abuse and money are separated in only one product, and badly.** Plunk has one `disabledReason`
  enum column (`PAYMENT_FAILED`, `EMAIL_REPUTATION`, `PHISHING_DETECTED`, `CARD_VERIFICATION_FAILED`,
  `MANUAL`) over one `disabled` boolean. Twenty reuses `SUSPENDED` for non-payment and for workspace
  deletion. Chatwoot has an independent `Account.status` (`active`/`suspended`) next to Stripe state.
  Postal has server-level and organization-level `suspended_at` and no billing.
- **Self-host hiding takes five forms**: compile-time macros (Plausible `on_ee`, Keila `if_cloud`),
  directory presence (Chatwoot `enterprise/`), runtime env flag (Documenso, Twenty, Plunk, Formbricks,
  Plausible `SELFHOST`), a license-gated directory (PostHog `ee/`, Infisical `ee/`, Documenso
  `packages/ee`), and a null-object default (Twenty `NoUsageLimitEntitlementProvider`, Documenso
  `SELFHOSTED_PLAN_LIMITS = Infinity`, Notifuse `0 = unlimited`).
- **No billing code at the pinned commits** in Listmonk, Mautic and Dittofeed (details in section M).

Inference (labelled where used below): ADR 0009's model (decide out of band, the hot path checks one
cached fact) matches PostHog's and Plausible's architecture, and the read contract matches Notifuse's,
but a finalized-only snapshot cannot drive warnings or grace on its own.

## A. Plausible (AGPL core + proprietary `extra/`, Elixir)

Closest prior art for over-limit behavior and for the hosted/self-host split. The billing code is in
the AGPL `lib/` tree and is compiled out of self-host builds, not kept in `extra/`.

- **(a) Plans and limits.** Plan details are static JSON files loaded at compile time, pricing comes from
  Paddle ([lib/plausible/billing/plans.ex#L1-L30](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/plans.ex#L1-L30)).
  Enterprise plans are DB rows (`EnterprisePlan`). Trial site limit is a constant
  ([lib/plausible/teams/billing.ex#L185](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams/billing.ex#L185)).
- **(b) What is metered, how.** Pageviews + custom events summed over three billing-cycle windows
  (`:current_cycle`, `:last_cycle`, `:penultimate_cycle`) derived from the subscription's `last_bill_date`,
  queried live from ClickHouse, **sum, not high-water-mark**
  ([teams/billing.ex#L366](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams/billing.ex#L366),
  [#L417-L437](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams/billing.ex#L417-L437)).
  Without a billing cycle (trial) it uses a rolling last-30-days window.
  There is no stored snapshot.
- **(c) Enforcement and what it does.** Two separate mechanisms:
  1. _Over-limit._ A daily Oban job (`0 14 * * *`) selects only subscribers whose billing day-of-month
     was yesterday, so each team is checked once per cycle
     ([workers/check_usage.ex#L61-L66](https://github.com/plausible/analytics/blob/bc2c7b3/lib/workers/check_usage.ex#L61-L66),
     [config/runtime.exs#L849](https://github.com/plausible/analytics/blob/bc2c7b3/config/runtime.exs#L849)).
     It starts a grace period only if **both** the last and penultimate cycles exceed the limit
     ([check_usage.ex#L163-L168](https://github.com/plausible/analytics/blob/bc2c7b3/lib/workers/check_usage.ex#L163-L168),
     [quota.ex#L135](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/qouta/quota.ex#L135))
     with a **10% margin** ([limits.ex#L7-L12](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/qouta/limits.ex#L7-L12)),
     emails owners and billing members, and sets a **7-day** `grace_period`
     ([teams/grace_period.ex#L6](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams/grace_period.ex#L6),
     [#L36](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams/grace_period.ex#L36)).
     Enterprise plans get a "manual lock" grace with no end date, controlled from an internal CRM
     ([grace_period.ex#L50-L56](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams/grace_period.ex#L50-L56),
     [check_usage.ex#L132-L161](https://github.com/plausible/analytics/blob/bc2c7b3/lib/workers/check_usage.ex#L132-L161)).
     A daily `LockSites` job (`0 0 * * *`) calls `SiteLocker.update_for`: when the grace has ended and usage is
     still over, it sets `teams.locked = true` and emails once
     ([billing/site_locker.ex#L20-L45](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/site_locker.ex#L20-L45),
     [runtime.exs#L853](https://github.com/plausible/analytics/blob/bc2c7b3/config/runtime.exs#L853)).
     The lock clears automatically when usage drops within the limit
     ([teams/billing.ex#L120-L146](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams/billing.ex#L120-L146),
     [check_usage.ex#L103-L111](https://github.com/plausible/analytics/blob/bc2c7b3/lib/workers/check_usage.ex#L103-L111)).
     **Lock scope:** the dashboard and the stats API return "site locked"
     ([stats_controller.ex#L58](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible_web/controllers/stats_controller.ex#L58),
     [authorize_public_api.ex#L248](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible_web/plugs/authorize_public_api.ex#L248)),
     with a super-admin bypass. The ingestion code path has no `locked` check (grep of `lib/plausible/ingestion`),
     and the pricing page says so in product copy: "your dashboards will be temporarily locked, but your
     stats will continue to be collected"
     ([live/choose_plan.ex#L232](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible_web/live/choose_plan.ex#L232)).
     **Warning thresholds:** the in-app notice has a pure-function priority list including
     `:pageview_approaching_limit` at **90%** of the base limit (no margin) and `:traffic_exceeded_*`
     states ([quota.ex#L194](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/qouta/quota.ex#L194),
     [#L219](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/qouta/quota.ex#L219),
     [#L282](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/qouta/quota.ex#L282)).
  2. _No subscription / lapsed._ Ingestion has its own gate: `GateKeeper` returns `:payment_required` once
     `team.accept_traffic_until` has passed
     ([site/gate_keeper.ex#L44-L49](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/site/gate_keeper.ex#L44-L49)).
     That date is trial end + **14 days** or next bill date + **30 days**
     ([teams.ex#L279-L296](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams.ex#L279-L296),
     [teams/team.ex#L19-L20](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams/team.ex#L19-L20)).
     So dropping data is a long-tail backstop, separate from the dashboard lock.
- **(d) Non-payment.** A Paddle subscription in `past_due` still counts as active
  ([billing/subscriptions.ex#L8](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/subscriptions.ex#L8)),
  and a cancelled one is active until `next_bill_date`
  ([#L10-L13](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/subscriptions.ex#L10-L13)).
  Dunning itself is Paddle's (inference: no retry logic in the app). The app's `locked` flag is one boolean computed from "no active trial or
  subscription" or "grace ended" ([site_locker.ex#L20-L45](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/site_locker.ex#L20-L45)),
  so over-limit and lapsed payment converge on one flag. Abuse is a different mechanism (not read).
- **(e) Hidden from self-hosters.** Two layers. Compile time: `on_ee`/`ee?` expand per `Mix.env`
  (`ce`, `ce_dev`, `ce_test` are the community builds)
  ([lib/plausible.ex#L6-L47](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible.ex#L6-L47)).
  For example, `on_trial?` is `always(true)` and `site_limit` is `:unlimited` in CE builds
  ([teams.ex#L55-L65](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams.ex#L55-L65),
  [teams/billing.ex#L184-L211](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams/billing.ex#L184-L211)).
  Runtime: `SELFHOST` (default true) removes the cloud cron entries and queues (`check_usage`, `lock_sites`, ...)
  ([runtime.exs#L296](https://github.com/plausible/analytics/blob/bc2c7b3/config/runtime.exs#L296),
  [#L876](https://github.com/plausible/analytics/blob/bc2c7b3/config/runtime.exs#L876),
  [#L912](https://github.com/plausible/analytics/blob/bc2c7b3/config/runtime.exs#L912)).
  `extra/` carries a separate license and a runtime license check (`extra/lib/license.ex`).

## B. PostHog (MIT core, proprietary `ee/`, Python + Node + Rust)

Closest prior art for ADR 0009's "decide out of band, hot path asks a cached fact" shape.

- **(a) Plans and limits.** In an external billing service. The open code fetches the result from
  `BILLING_SERVICE_URL` ([ee/billing/billing_manager.py#L430](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/billing_manager.py#L430))
  and stores a per-resource `usage` summary on the `Organization` row; the limit is read from that summary
  ([ee/billing/quota_limiting.py#L442-L457](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L442-L457)).
  The plan catalogue and prices are not in the repo.
- **(b) Metering.** Per resource (events, recordings, exceptions, feature-flag requests, rows synced, ...)
  `usage + todays_usage` against the billing period `usage["period"]`: the closed service supplies the
  closed days, the app adds today's count ([quota_limiting.py#L456](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L456),
  [#L519-L520](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L519-L520)). Sum, not peak.
- **(c) Enforcement.** A Temporal schedule runs the limiter at minutes 10/25/40/55 of every hour, i.e.
  every 15 minutes ([posthog/temporal/schedule.py#L165-L174](https://github.com/PostHog/posthog/blob/1f18623/posthog/temporal/schedule.py#L165-L174);
  the `update_all_orgs_billing_quotas` docstring says "every 30 minutes", the schedule is authoritative).
  Result: a Redis sorted set of limited team tokens scored by `limited_until`
  ([nodejs/src/common/services/quota-limiting.service.ts#L103](https://github.com/PostHog/posthog/blob/1f18623/nodejs/src/common/services/quota-limiting.service.ts#L103)).
  The Rust capture service reads it and **drops** the over-quota events at ingestion
  ([rust/capture/src/quota_limiters.rs#L155-L165](https://github.com/PostHog/posthog/blob/1f18623/rust/capture/src/quota_limiters.rs#L155-L165),
  global limiter applied last, [#L182-L188](https://github.com/PostHog/posthog/blob/1f18623/rust/capture/src/quota_limiters.rs#L182-L188)).
  Only a resource's own events are dropped for scoped limiters (surveys, exceptions, LLM events); the global
  one drops everything. Over-limit is `usage >= limit + OVERAGE_BUFFER` (0 for most, 1000 for recordings)
  ([quota_limiting.py#L108](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L108),
  [#L518](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L518)).
  **Grace** is by customer trust score: score 3 or unknown = limit immediately, 7 = 1 day, 10 = 3 days,
  15 = 5 days; some resources are exempt from any grace
  ([#L59-L72](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L59-L72),
  [#L129](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L129),
  [#L643-L693](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L643-L693)). The limit
  lasts until the **billing period end** (`quota_limited_until = billing_period_end`). Opt-outs: org
  `never_drop_data` ([#L561](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L561))
  and a feature flag ([#L609](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L609)).
  Dashboard access is not part of this mechanism (not read: no lock-out of the UI found in these files).
- **(d) Non-payment.** Not in the open repo (billing service is closed). Not verified.
- **(e) Hidden from self-hosters.** `ee/` is under the PostHog Enterprise license
  ([ee/LICENSE#L1](https://github.com/PostHog/posthog/blob/1f18623/ee/LICENSE#L1)); the quota path needs the closed billing
  service to populate `organization.usage`, so without it the limiter has nothing to compare (inference:
  `if not organization.usage: return None`, [#L448-L449](https://github.com/PostHog/posthog/blob/1f18623/ee/billing/quota_limiting.py#L448-L449)).

## C. Keila (AGPL core + non-AGPL `extra/`, Elixir)

Closest prior art for "limits in the AGPL core, money in a closed part", and for a **prepaid** model.

- **(a) Plans and limits.** Core has no plan concept, only an `Account` credit ledger (`CreditTransaction`,
  positive and negative rows with `expires_at`/`valid_from`)
  ([lib/keila/accounts/accounts.ex#L252](https://github.com/pentacent/keila/blob/2308beb/lib/keila/accounts/accounts.ex#L252),
  consume: [#L282](https://github.com/pentacent/keila/blob/2308beb/lib/keila/accounts/accounts.ex#L282)). Plans (`monthly_credits`,
  Paddle plan ids) are in the non-AGPL `extra/keila_cloud/billing`.
- **(b) Metered.** Emails, one credit per recipient (transactional: 1 + cc + bcc)
  ([lib/keila/mailings/transactional_message.ex#L165-L174](https://github.com/pentacent/keila/blob/2308beb/lib/keila/mailings/transactional_message.ex#L165-L174)).
  A paid month adds `monthly_credits` expiring at `next_billed_on`; an annual plan adds twelve monthly
  tranches with `valid_from` windows
  ([extra/keila_cloud/billing/billing.ex#L94-L120](https://github.com/pentacent/keila/blob/2308beb/extra/keila_cloud/billing/billing.ex#L94-L120)).
- **(c) Enforcement.** On the send path, before anything is sent: a Campaign debits credits for the
  **entire audience inside the delivery transaction**, and `Repo.rollback(:insufficient_credits)` aborts
  the whole delivery ([lib/keila/mailings/mailings.ex#L592](https://github.com/pentacent/keila/blob/2308beb/lib/keila/mailings/mailings.ex#L592),
  [#L631-L645](https://github.com/pentacent/keila/blob/2308beb/lib/keila/mailings/mailings.ex#L631-L645)). Hard block, no grace, no overage.
  Features can also depend on credits (`double_opt_in`, `welcome_email` are unavailable with zero credits)
  ([extra/keila_cloud/billing/features.ex#L15-L43](https://github.com/pentacent/keila/blob/2308beb/extra/keila_cloud/billing/features.ex#L15-L43)).
- **(d) Non-payment.** The Paddle webhook controller: `subscription_payment_succeeded` adds credits,
  `subscription_payment_failed` only updates the subscription row, `subscription_cancelled` cancels it
  ([extra/keila_cloud_web/controllers/cloud_paddle_webhook_controller.ex#L40-L65](https://github.com/pentacent/keila/blob/2308beb/extra/keila_cloud_web/controllers/cloud_paddle_webhook_controller.ex#L40-L65)).
  So non-payment means credits expire and are not refilled: **no explicit hold, no dunning state in the app**.
  A cloud-only `account.status != :active` check also aborts campaign delivery
  ([mailings.ex#L635-L638](https://github.com/pentacent/keila/blob/2308beb/lib/keila/mailings/mailings.ex#L635-L638)); the
  meaning of that status (abuse vs onboarding) was not traced.
- **(e) Hidden from self-hosters.** The ledger is in core but off by default (`credits_enabled: false`,
  enabled by `ENABLE_QUOTAS`; every ledger function returns `:ok`/`0` when off)
  ([config/config.exs#L78](https://github.com/pentacent/keila/blob/2308beb/config/config.exs#L78),
  [config/runtime.exs#L374](https://github.com/pentacent/keila/blob/2308beb/config/runtime.exs#L374)). Cloud code compiles in only if `KeilaCloud` is
  present, via `Keila.if_extra`/`if_cloud` macros
  ([lib/keila.ex#L7](https://github.com/pentacent/keila/blob/2308beb/lib/keila.ex#L7), [#L33](https://github.com/pentacent/keila/blob/2308beb/lib/keila.ex#L33)); `extra/` is
  excluded from AGPL ([extra/README.md#L6](https://github.com/pentacent/keila/blob/2308beb/extra/README.md#L6)).

## D. Plunk (AGPL, TypeScript; the only email sender here with Stripe in the open repo)

Closest prior art for send-path enforcement, free-tier cap, warnings, and (as a counter-example) the
abuse/payment conflation.

- **(a) Plans and limits.** No plan table. A project is free-tier when it has no `subscription` and no
  custom limit; the free cap is a code constant `FREE_TIER_TOTAL_LIMIT = 1000` emails/month across all
  sources ([services/BillingLimitService.ts#L49](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/BillingLimitService.ts#L49),
  [#L222](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/BillingLimitService.ts#L222)). Paid projects are pay-per-email on a Stripe
  metered price. Customers can set **their own** per-source monthly caps `billingLimitWorkflows|Campaigns|Transactional|Inbound`
  ([packages/db/prisma/schema.prisma#L63](https://github.com/useplunk/plunk/blob/12f9f20/packages/db/prisma/schema.prisma#L63)), a spend guard rather than a plan limit.
- **(b) Metering.** Usage = `Email` rows this **calendar month** with `status != FAILED`, "mirrors what Stripe
  is actually metered for"; `PENDING` and `SENDING` **count**, so in-flight mail holds quota
  ([BillingLimitService.ts#L24-L34](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/BillingLimitService.ts#L24-L34),
  [#L66](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/BillingLimitService.ts#L66)). Redis cache with a 5-minute TTL plus an
  increment after each send ([#L47](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/BillingLimitService.ts#L47),
  [#L141-L155](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/BillingLimitService.ts#L141-L155)); on error the check returns 0 (fail open,
  [#L75-L78](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/BillingLimitService.ts#L75-L78)). The billing event is a Stripe meter
  event queued through BullMQ with an idempotency `identifier`
  ([jobs/meter-processor.ts#L21-L28](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/jobs/meter-processor.ts#L21-L28)).
- **(c) Enforcement.** On the send path of all three sources (`checkLimit` before creating the email row),
  throwing HTTP **429** for transactional
  ([services/EmailService.ts#L77-L81](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/EmailService.ts#L77-L81)); a campaign is checked **once
  for the whole batch** (`usage + recipientCount > limit` then HTTP 403)
  ([services/CampaignService.ts#L528-L545](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/CampaignService.ts#L528-L545)).
  **Warning at 80%** and **block at 100%**, each with an email to project members deduped by a Redis `SET NX`
  key that expires at month end ("only once per month")
  ([BillingLimitService.ts#L48](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/BillingLimitService.ts#L48),
  [#L250-L262](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/BillingLimitService.ts#L250-L262),
  [#L615-L640](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/services/BillingLimitService.ts#L615-L640)).
  Hard block, no grace, no overage for the capped cases. The check is wrapped in `STRIPE_ENABLED` for the free-tier branch.
- **(d) Non-payment.** The Stripe webhook `invoice.payment_failed` sets `disabled = true,
  disabledReason = 'PAYMENT_FAILED'` on the **first** failure of a recurring invoice (first-time
  `subscription_create` failures are skipped) and emails members
  ([controllers/Webhooks.ts#L716-L768](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/controllers/Webhooks.ts#L716-L768)).
  `invoice.paid` re-enables **only if** `disabledReason === 'PAYMENT_FAILED'`
  ([#L685-L708](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/controllers/Webhooks.ts#L685-L708)). The enum has five causes in one column:
  `PAYMENT_FAILED`, `EMAIL_REPUTATION`, `PHISHING_DETECTED`, `CARD_VERIFICATION_FAILED`, `MANUAL`
  ([schema.prisma#L716-L722](https://github.com/useplunk/plunk/blob/12f9f20/packages/db/prisma/schema.prisma#L716-L722)). The `payment_failed` handler writes
  the reason without reading the existing one (verified,
  [Webhooks.ts#L741-L745](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/controllers/Webhooks.ts#L741-L745)).
  **Inference:** a project disabled for `EMAIL_REPUTATION` whose invoice later fails gets its reason overwritten
  to `PAYMENT_FAILED`, after which a successful payment re-enables it. This is a concrete example of why
  ADR 0009 keeps billing and abuse as independent causes.
- **What a disabled project can still do.** API-key middleware rejects only write methods with 403, reads still work
  ([middleware/auth.ts#L131-L141](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/middleware/auth.ts#L131-L141)).
  Queued emails are **cancelled as `FAILED`** and campaigns are finalized with a partial `sentCount`, not paused
  ([jobs/email-processor.ts#L149-L156](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/jobs/email-processor.ts#L149-L156)). Contrast with 1mail's
  resumable `Held` outcome (ADR 0007 amendment).
- **(e) Hidden from self-hosters.** `STRIPE_ENABLED = STRIPE_SK !== '' && STRIPE_WEBHOOK_SECRET !== ''`: runtime
  keys, no build split; the billing code ships in the AGPL image
  ([app/constants.ts#L92](https://github.com/useplunk/plunk/blob/12f9f20/apps/api/src/app/constants.ts#L92)).

## E. Twenty (AGPL + `@license Enterprise` files, TypeScript)

- **(a) Plans and limits.** Stripe products/prices synced into DB (`billing-sync-plans-data` command) and
  entitlements read from the subscription (e.g. `BillingEntitlementKey.USAGE_LIMIT`). Trial lengths are
  config: 30 days with a card, 7 days without
  ([twenty-config/config-variables.ts#L970](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/twenty-config/config-variables.ts#L970),
  [#L980](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/twenty-config/config-variables.ts#L980)). Per-workspace
  usage limits are a `usageLimit` table (resource, spender, period, kind).
- **(b) Metering.** Credits (AI, workflow) consumed against a quota with a period (`usage-limit` module:
  quota definition, consumption and period services). Quota arithmetic and the Stripe-side metering were not
  traced.
- **(c) Enforcement.** Two mechanisms: a credit/usage-limit service per resource, and a workspace-wide switch
  (below). The latter is a NestJS guard plus a GraphQL middleware that **refuses every request** from a
  suspended workspace unless the endpoint is annotated `@AllowSuspendedWorkspace`
  ([engine/middlewares/graphql-refuse-suspended-workspace.middleware.ts#L17](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/middlewares/graphql-refuse-suspended-workspace.middleware.ts#L17),
  [engine/guards/workspace-not-suspended.guard.ts#L37](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/guards/workspace-not-suspended.guard.ts#L37)).
  Reminders: a daily cron at 08:00 UTC sends trial-ending (1 day before without card, 7 days with card) and renewal
  (7 days before) emails, deduped by a stored boundary date
  ([billing/reminders/constants/billing-reminder.cron-pattern.constant.ts#L1](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/billing/reminders/constants/billing-reminder.cron-pattern.constant.ts#L1),
  [config-variables.ts#L1011](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/twenty-config/config-variables.ts#L1011),
  [#L1022](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/twenty-config/config-variables.ts#L1022)).
- **(d) Non-payment.** `shouldSuspendWorkspace` is true when **all** live subscriptions are `Canceled` or `Unpaid`,
  or `PastDue` within 24 hours after a trial ended; a plain `past_due` does **not** suspend. So suspension waits
  for Stripe's own retry schedule to run out. `Active`/`Trialing` reactivate
  ([billing-webhook/services/billing-webhook-subscription.service.ts#L211-L217](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/billing-webhook/services/billing-webhook-subscription.service.ts#L211-L217),
  [#L276-L302](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/billing-webhook/services/billing-webhook-subscription.service.ts#L276-L302)).
  **Not separated from other causes:** the same `suspendWorkspace` sets `SUSPENDED` in the billing webhook, and is
  also called while deleting a workspace
  ([workspace/workspace.service.ts#L632](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/workspace/services/workspace.service.ts#L632),
  [workspace.resolver.ts#L254](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/workspace/workspace.resolver.ts#L254),
  [user/services/user.service.ts#L349](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/user/services/user.service.ts#L349)).
  No separate abuse suspension was found (not verified beyond a grep).
- **(e) Hidden from self-hosters.** `IS_BILLING_ENABLED` env flag defaulting to `false`
  ([config-variables.ts#L952](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/twenty-config/config-variables.ts#L952)).
  The seam is an abstract provider in core with a no-op default: `UsageLimitEntitlementProvider` has a
  `NoUsageLimitEntitlementProvider` for self-host
  ([usage-limit/services/no-usage-limit-entitlement-provider.service.ts#L3](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/usage-limit/services/no-usage-limit-entitlement-provider.service.ts#L3)),
  and the billing implementation carries a `/* @license Enterprise */` header
  ([billing/services/billing-usage-limit-entitlement-provider.service.ts#L1](https://github.com/twentyhq/twenty/blob/2e8b29b/packages/twenty-server/src/engine/core-modules/billing/services/billing-usage-limit-entitlement-provider.service.ts#L1)).
  This is the same interface-in-core, implementation-in-`ee/` shape 1mail already uses (`site.AuditLog`, `events.Consumer`).

## F. Formbricks (AGPL + `ee/` license, TypeScript)

- **(a) Plans and limits.** Stripe is the catalogue; the app mirrors it into the organization row
  `billing.limits = {workspaces, monthly: {responses, workflowRuns}}` (null = unlimited), default
  3 workspaces and **1500 responses**, and `billing.stripe.{plan, subscriptionStatus, trialEnd, paymentAttemptError, ...}`
  ([packages/types/organizations.ts#L8-L77](https://github.com/formbricks/formbricks/blob/2c5c3b1/packages/types/organizations.ts#L8-L77)).
- **(b) Metering.** Live `COUNT(*)` of responses over a billing **usage cycle window** anchored on
  `usageCycleAnchor` ([apps/web/lib/organization/service.ts#L384-L400](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/lib/organization/service.ts#L384-L400)).
  Billing side: one Stripe meter event per created response/workflow run with idempotency `identifier`
  `response_created:<id>`, fire-and-forget, failure only logged
  ([modules/ee/billing/lib/metering.ts#L6-L40](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/modules/ee/billing/lib/metering.ts#L6-L40)).
- **(c) Enforcement.** **Soft.** The file says "Stripe aggregates events ... and bills the overage beyond the plan's
  included volume. There is no hard cap here"
  ([metering.ts#L38-L45](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/modules/ee/billing/lib/metering.ts#L38-L45)). The only in-app reaction found is a
  dismissible "limits reached" banner when the live count reaches the limit
  ([modules/ui/components/limits-reached-banner/index.tsx#L17](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/modules/ui/components/limits-reached-banner/index.tsx#L17)).
  (A grep of the response-creation routes found no limit check; not exhaustively verified.)
- **(d) Non-payment.** Statuses `trialing|active|past_due|unpaid|paused` all count as active
  ([modules/ee/billing/lib/organization-billing.ts#L58](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/modules/ee/billing/lib/organization-billing.ts#L58));
  a failed payment attempt is recorded as `paymentAttemptError` and surfaced on the billing page
  ([components/pricing-table.tsx#L643](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/modules/ee/billing/components/pricing-table.tsx#L643)). No lock found. Stripe
  dunning is the only clock. **Inference:** non-payment degrades to Stripe cancelling the subscription, after which
  the account falls back to the free "Hobby" plan (`subscription.deleted` handling is reflected in the downgrade
  code around [#L1152-L1172](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/modules/ee/billing/lib/organization-billing.ts#L1152-L1172)).
- **(e) Hidden from self-hosters.** `IS_FORMBRICKS_CLOUD === "1"` runtime flag gates metering and the usage reads
  ([apps/web/lib/constants.ts#L7](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/lib/constants.ts#L7), `metering.ts#L11`);
  `ENTERPRISE_LICENSE_KEY` is the separate EE license mechanism
  ([constants.ts#L239](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/lib/constants.ts#L239)), analogous to the EE license/SaaS split in ADR 0009.
- **Trial.** Pro trial of 14 days (7 when shortened) created with `missing_payment_method: "cancel"`; for orgs
  with no card the end behavior is overridden to `create_invoice` so the trial ends into an active **$0 Hobby**
  rather than a cancelled subscription
  ([organization-billing.ts#L578-L580](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/modules/ee/billing/lib/organization-billing.ts#L578-L580),
  [#L616](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/modules/ee/billing/lib/organization-billing.ts#L616),
  [#L1161-L1170](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/modules/ee/billing/lib/organization-billing.ts#L1161-L1170)).

## G. Documenso (AGPL + `packages/ee` commercial license, TypeScript)

- **(a) Plans and limits.** Constants per state: free `{documents: 5, recipients: 10, directTemplates: 3}`, inactive all
  zero, paid and self-hosted `Infinity`
  ([packages/ee/server-only/limits/constants.ts#L3-L25](https://github.com/documenso/documenso/blob/38ecb21/packages/ee/server-only/limits/constants.ts#L3-L25));
  paid limits come from an `organisationClaim` row with flags.
- **(b) Metering.** `getServerLimits` is computed **live on each call**: `COUNT(*)` of envelopes since
  `startOf('month')` (UTC calendar month, not the billing period), minus the quota
  ([limits/server.ts#L47-L130](https://github.com/documenso/documenso/blob/38ecb21/packages/ee/server-only/limits/server.ts#L47-L130)).
- **(c) Enforcement.** On document/envelope creation routes (`create-document`, `create-envelope`, `use-envelope`,
  template routes) via `remaining` ([trpc/server/document-router/create-document.ts#L61](https://github.com/documenso/documenso/blob/38ecb21/packages/trpc/server/document-router/create-document.ts#L61)).
  Hard block of **creation**; existing documents stay accessible (inference from the calls being on create paths only).
- **(d) Non-payment.** Stripe `past_due` maps to `PAST_DUE`, anything not active/trialing/past_due maps to
  `INACTIVE` ([stripe/sync-stripe-customer-subscription.ts#L165-L169](https://github.com/documenso/documenso/blob/38ecb21/packages/ee/server-only/stripe/sync-stripe-customer-subscription.ts#L165-L169)).
  Only `INACTIVE` (and an org awaiting its first payment) gets zero limits
  ([limits/server.ts#L65-L79](https://github.com/documenso/documenso/blob/38ecb21/packages/ee/server-only/limits/server.ts#L65-L79)); `PAST_DUE` keeps the plan. So
  the provider's dunning is the grace period. An `ENTERPRISE` claim bypasses limits even when expired
  ([#L55-L62](https://github.com/documenso/documenso/blob/38ecb21/packages/ee/server-only/limits/server.ts#L55-L62)).
- **(e) Hidden from self-hosters.** Runtime `NEXT_PUBLIC_FEATURE_BILLING_ENABLED`; when off, `getServerLimits` returns
  `SELFHOSTED_PLAN_LIMITS` first thing
  ([lib/constants/app.ts#L62](https://github.com/documenso/documenso/blob/38ecb21/packages/lib/constants/app.ts#L62),
  [server.ts#L47-L53](https://github.com/documenso/documenso/blob/38ecb21/packages/ee/server-only/limits/server.ts#L47-L53)). `packages/ee` has a commercial license
  ([packages/ee/LICENSE#L1](https://github.com/documenso/documenso/blob/38ecb21/packages/ee/LICENSE#L1)).

## H. Chatwoot (MIT core + `enterprise/` license, Ruby)

- **(a) Plans and limits.** In the DB as `InstallationConfig` rows editable from the super-admin console
  (`CHATWOOT_CLOUD_PLANS`, `ACCOUNT_EMAILS_PLAN_LIMITS`, `CAPTAIN_CLOUD_PLAN_LIMITS`), plus a per-account
  `limits` JSON override ([enterprise/app/services/enterprise/billing/handle_stripe_event_service.rb#L4](https://github.com/chatwoot/chatwoot/blob/e212425/enterprise/app/services/enterprise/billing/handle_stripe_event_service.rb#L4),
  [enterprise/app/models/enterprise/account/plan_usage_and_limits.rb#L94](https://github.com/chatwoot/chatwoot/blob/e212425/enterprise/app/models/enterprise/account/plan_usage_and_limits.rb#L94)).
- **(b) Metering.** Counters in `custom_attributes` incremented inline and reset on billing-period renewal
  (`captain_responses_usage`) ([plan_usage_and_limits.rb#L18-L26](https://github.com/chatwoot/chatwoot/blob/e212425/enterprise/app/models/enterprise/account/plan_usage_and_limits.rb#L18-L26),
  [handle_stripe_event_service.rb#L45-L53](https://github.com/chatwoot/chatwoot/blob/e212425/enterprise/app/services/enterprise/billing/handle_stripe_event_service.rb#L45-L53)).
  Outbound email: a Redis **daily** counter (25 h TTL) in a core concern
  ([app/models/concerns/account_email_rate_limitable.rb#L4-L30](https://github.com/chatwoot/chatwoot/blob/e212425/app/models/concerns/account_email_rate_limitable.rb#L4-L30)).
- **(c) Enforcement.** `within_email_rate_limit?` is defined in core and returns true unless `chatwoot_cloud?`
  (its call sites were not traced).
  The limit precedence is account override, then plan, then global, then unlimited
  ([enterprise/.../plan_usage_and_limits.rb#L41-L43](https://github.com/chatwoot/chatwoot/blob/e212425/enterprise/app/models/enterprise/account/plan_usage_and_limits.rb#L41-L43),
  [account_email_rate_limitable.rb#L77-L87](https://github.com/chatwoot/chatwoot/blob/e212425/app/models/concerns/account_email_rate_limitable.rb#L77-L87)). The enterprise
  module overrides one method; core keeps the check. Seat and inbox limits are exposed as `usage_limits`.
- **(d) Non-payment.** Only `customer.subscription.updated` and `customer.subscription.deleted` are handled
  ([handle_stripe_event_service.rb#L13-L24](https://github.com/chatwoot/chatwoot/blob/e212425/enterprise/app/services/enterprise/billing/handle_stripe_event_service.rb#L13-L24)):
  deletion recreates the default (free) plan and zeroes credits, i.e. a **downgrade, not a lock**
  ([#L107-L120](https://github.com/chatwoot/chatwoot/blob/e212425/enterprise/app/services/enterprise/billing/handle_stripe_event_service.rb#L107-L120)). `Account.status` is an
  independent `active|suspended` enum ([app/models/account.rb#L113](https://github.com/chatwoot/chatwoot/blob/e212425/app/models/account.rb#L113)); no billing path
  setting it was found (grep of `app`, `enterprise`, `lib`), the super-admin dashboard filters on it.
- **(e) Hidden from self-hosters.** Directory presence plus env: `enterprise?` is true only when `enterprise/`
  exists and `DISABLE_ENTERPRISE` is unset; `chatwoot_cloud?` also needs `DEPLOYMENT_ENV == 'cloud'`
  ([lib/chatwoot_app.rb#L14-L22](https://github.com/chatwoot/chatwoot/blob/e212425/lib/chatwoot_app.rb#L14-L22)). The `enterprise/` directory has its own license
  ([enterprise/LICENSE#L1](https://github.com/chatwoot/chatwoot/blob/e212425/enterprise/LICENSE#L1)).

## I. Infisical (MIT core + `ee/` license, TypeScript)

- **(a) Plans and limits.** A remote **license server**: in cloud mode (`LICENSE_SERVER_V2_SERVICE_KEY`) every org's feature
  set is fetched from it, cached in Redis for **15 minutes** (with jitter) and a 24-hour "last known" copy
  ([ee/services/license/license-service.ts#L95](https://github.com/Infisical/infisical/blob/c40d5fd/backend/src/ee/services/license/license-service.ts#L95),
  [#L242-L277](https://github.com/Infisical/infisical/blob/c40d5fd/backend/src/ee/services/license/license-service.ts#L242-L277),
  [keystore/keystore.ts#L277-L278](https://github.com/Infisical/infisical/blob/c40d5fd/backend/src/keystore/keystore.ts#L277-L278)). A billing mutation flags the org so the
  cached plan is served while a background refresh runs ([#L266-L268](https://github.com/Infisical/infisical/blob/c40d5fd/backend/src/ee/services/license/license-service.ts#L266-L268)).
- **(b) Metering.** A `usageMeteringService` emits meters and `reconcile(orgId)` is triggered from `getPlan`, throttled by a
  Redis marker and only for paid plans (`plan.slug` non-null)
  ([license-service.ts#L253-L261](https://github.com/Infisical/infisical/blob/c40d5fd/backend/src/ee/services/license/license-service.ts#L253-L261)). Metering is pushed to the closed side.
- **(c) Enforcement.** Per-feature checks `plan.<feature>` at each call site (e.g. `plan.secretApproval`) and a
  `getEnforcedIdentityLimit(plan)` for SSO/LDAP provisioning; this version has no over-limit freeze (seat limits are in
  the plan type, [ee/services/license/license-types.ts#L53-L54](https://github.com/Infisical/infisical/blob/c40d5fd/backend/src/ee/services/license/license-types.ts#L53-L54)).
- **(d) Non-payment.** Not in the repo (license server is closed). Not verified. On a license-server error `getPlan` falls
  back to the on-prem feature set and flags the cache entry as fallback
  ([license-service.ts#L279-L291](https://github.com/Infisical/infisical/blob/c40d5fd/backend/src/ee/services/license/license-service.ts#L279-L291)).
  Whether that fallback fails open or closed for a given feature was not traced.
- **(e) Hidden from self-hosters.** `ee/` is under a separate license ([backend/src/ee/LICENSE.md#L1](https://github.com/Infisical/infisical/blob/c40d5fd/backend/src/ee/LICENSE.md#L1));
  self-hosted deployments use an offline or online license key rather than a plan lookup
  ([license-service.ts#L104-L112](https://github.com/Infisical/infisical/blob/c40d5fd/backend/src/ee/services/license/license-service.ts#L104-L112)).

## J. Notifuse (BSL 1.1 at this commit; control-plane pull model)

Closest prior art for ADR 0009's **read contract**, but the enforcement half is not implemented in the repo.

- **(a) Plans and limits.** Pushed by "the Notifuse Cloud control plane" as process environment variables
  `PLAN_MAX_ACTIVE_CONTACTS|STORED_CONTACTS|MONTHLY_EVENTS|MONTHLY_PAGEVIEWS` and `PLAN_DATA_RETENTION_MONTHS`;
  `0 = unlimited`, so "a self-hosted install that sets none of them is unaffected"
  ([config/config.go#L520-L529](https://github.com/Notifuse/notifuse/blob/7516397/config/config.go#L520-L529),
  [#L890-L897](https://github.com/Notifuse/notifuse/blob/7516397/config/config.go#L890-L897)).
- **(b) Metering.** A `monthly_usage` table, one row per UTC month with pageviews and timeline entries. The row is "a
  recomputed snapshot, not an incremental ledger": a maintenance worker recounts the open and previous month, so
  the ingest hot path is untouched
  ([internal/database/schema/usage_tables.go#L1-L28](https://github.com/Notifuse/notifuse/blob/7516397/internal/database/schema/usage_tables.go#L1-L28)).
- **(c) Enforcement.** **None found.** A grep of all Go sources for the `Plan.Max*` fields finds only the config
  struct and a startup log line ("Plan limits resolved: ... the only way to confirm that a redeploy actually delivered
  a quota change", [internal/app/app.go#L1972-L1980](https://github.com/Notifuse/notifuse/blob/7516397/internal/app/app.go#L1972-L1980)). At this commit the limits
  are plumbing without an enforcement site (or the enforcement is in code not in this repo).
- **The read contract.** `GET/POST /api/usage.get` on the bare mux, **pull-only** ("a self-hosted installation never
  initiates an outbound call"), authenticated by an HMAC-SHA256 signature over `{timestamp}.{path}` with a key derived
  from `SECRET_KEY` (domain-separated label), 5-minute max skew, `v1,` version prefix
  ([internal/http/usage_handler.go#L16-L50](https://github.com/Notifuse/notifuse/blob/7516397/internal/http/usage_handler.go#L16-L50),
  [internal/domain/usage.go#L30-L66](https://github.com/Notifuse/notifuse/blob/7516397/internal/domain/usage.go#L30-L66)). It returns stored snapshots summed across
  workspaces and reports the **oldest** `computed_at` so the reader can see staleness; it never recounts on read
  ([internal/service/usage_service.go#L15-L41](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/usage_service.go#L15-L41),
  [internal/domain/usage.go#L113-L126](https://github.com/Notifuse/notifuse/blob/7516397/internal/domain/usage.go#L113-L126)).
- **(d) Non-payment.** Not in the repo. The license service (a separate `EntitlementProvider` for EE features) is
  documented as "never blocks a send, never refuses a login and never deletes anything"
  ([internal/service/license_service.go#L143](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/license_service.go#L143)).
- **(e) Hidden from self-hosters.** Absence of configuration (all zeros) plus a pull-only endpoint; no outbound code.
  The repo is now BSL 1.1 (see [outbound-send-prior-art.md](./outbound-send-prior-art.md)).

## K. Postal (MIT; no billing; a send-limit mechanism worth copying)

Postal has no plans or billing. Its per-server send limit is the cleanest open example of live-usage warnings
plus a send-path hold.

- **Metering and thresholds.** `send_limit` is an operator-set integer per server. Usage is a **rolling 60-minute**
  live-stats window (`live_stats.total(60)`, max 60 minutes by construction), "approaching" at **90%**, "exceeded"
  at 100% ([app/models/server.rb#L190-L205](https://github.com/postalserver/postal/blob/d96edda/app/models/server.rb#L190-L205),
  [lib/postal/message_db/live_stats.rb#L24-L29](https://github.com/postalserver/postal/blob/d96edda/lib/postal/message_db/live_stats.rb#L24-L29)).
- **Enforcement.** One step in the outgoing pipeline (`check_send_limits`): exceeded means create a `Held` delivery
  "because send limit has been reached", remove the message from the queue and stop processing; approaching only marks
  the server and continues
  ([app/lib/message_dequeuer/outgoing_message_processor.rb#L17](https://github.com/postalserver/postal/blob/d96edda/app/lib/message_dequeuer/outgoing_message_processor.rb#L17),
  [#L118-L132](https://github.com/postalserver/postal/blob/d96edda/app/lib/message_dequeuer/outgoing_message_processor.rb#L118-L132)). Release of held messages was not traced.
- **Warnings.** A scheduled task every minute emails the organization's notification addresses and fires a webhook
  (`SendLimitApproaching`/`SendLimitExceeded`), deduplicated with `*_notified_at` columns
  ([scheduled_tasks/send_notifications_scheduled_task.rb#L5-L11](https://github.com/postalserver/postal/blob/d96edda/app/scheduled_tasks/send_notifications_scheduled_task.rb#L5-L11),
  [server.rb#L206](https://github.com/postalserver/postal/blob/d96edda/app/models/server.rb#L206)).
- **Suspension.** Separate `suspended_at` columns on server and organization;
  `suspended? = suspended_at.present? || organization.suspended?` with a stored reason
  ([server.rb#L110-L122](https://github.com/postalserver/postal/blob/d96edda/app/models/server.rb#L110-L122)). The limit and the suspension are independent conditions.

## L. Cal.com (MIT; billing logic not in this repo) and Lago

- **Cal.com.** Plan state lives in Stripe. The repo's Prisma schema has `TeamBilling` / `OrganizationBilling` with
  `status`, `billingPeriod`, `billingMode` and **seat high-water-mark** columns `highWaterMark` +
  `highWaterMarkPeriodStart`, with a migration that seeds a `hwm-seating` feature flag described as "charges for peak
  seats used during billing period" ([packages/prisma/schema.prisma#L2607-L2630](https://github.com/calcom/cal.com/blob/54343aa/packages/prisma/schema.prisma#L2607-L2630),
  [migrations/20260130100000_add_high_water_mark_fields/migration.sql](https://github.com/calcom/cal.com/blob/54343aa/packages/prisma/migrations/20260130100000_add_high_water_mark_fields/migration.sql)).
  A grep of all `.ts`/`.tsx` for `highWaterMark` finds no writer: **the code that maintains the peak is not in the
  public tree**. `CreditBalance` has `limitReachedAt` and `warningSentAt` (the same "dedupe the warning in the row"
  idea as Postal) ([schema.prisma#L652-L664](https://github.com/calcom/cal.com/blob/54343aa/packages/prisma/schema.prisma#L652-L664)). Gate:
  `IS_TEAM_BILLING_ENABLED = IS_STRIPE_ENABLED && HOSTED_CAL_FEATURES`, where `HOSTED_CAL_FEATURES` defaults to
  `!IS_SELF_HOSTED` ([packages/lib/constants.ts#L82](https://github.com/calcom/cal.com/blob/54343aa/packages/lib/constants.ts#L82),
  [#L130](https://github.com/calcom/cal.com/blob/54343aa/packages/lib/constants.ts#L130)). Treat as schema-only evidence.
- **No product among the clones uses Lago** (grepping the other clones for `lago` finds only i18n strings, a lockfile and
  a yarn bundle). Lago is itself a candidate plane; its plane-side surfaces relevant here are a `current_usage` and
  `projected_usage` read per customer/subscription
  ([config/routes/shared_api.rb#L22-L23](https://github.com/getlago/lago-api/blob/4c4f02d/config/routes/shared_api.rb#L22-L23)), `UsageMonitoring::Alert` thresholds
  (current usage, lifetime usage, wallet balance; [app/models/usage_monitoring/alert.rb](https://github.com/getlago/lago-api/blob/4c4f02d/app/models/usage_monitoring/alert.rb)) and
  `DunningCampaign` with `max_attempts` and `days_between_attempts`
  ([app/models/dunning_campaign.rb#L23-L24](https://github.com/getlago/lago-api/blob/4c4f02d/app/models/dunning_campaign.rb#L23-L24)). Not evaluated further here.

## M. No billing code: Listmonk, Mautic, Dittofeed

- **Listmonk:** a grep of Go and SQL for `stripe|billing|paddle|quota` finds nothing in `cmd/`, `internal/`, `schema.sql`.
- **Mautic:** no billing/Stripe/Paddle references in `app/bundles` or `plugins` (only mailbox-quota parsing in the bounce
  processor and an emoji map). No quota/limit concept for hosted plans.
- **Dittofeed:** `stripe` appears only in README, an example README and the OpenAPI JSON; no billing code under `packages`.
  A `WorkspaceStatusDbEnum` with `Paused` exists
  ([packages/isomorphic-lib/src/types.ts#L2102](https://github.com/dittofeed/dittofeed/blob/fe89657/packages/isomorphic-lib/src/types.ts#L2102)); it is not a billing state.

## N. Comparison

| Product    | Plans/limits live in                    | Metric and window                          | Over-limit (paid)                              | Over-limit (free)             | Non-payment trigger                       | Frozen surface                                  | Abuse vs money                | Self-host hiding                       |
| ---------- | --------------------------------------- | ------------------------------------------ | ---------------------------------------------- | ----------------------------- | ----------------------------------------- | ----------------------------------------------- | ----------------------------- | -------------------------------------- |
| Plausible  | JSON + Paddle + `EnterprisePlan` rows   | sum, billing cycle, live ClickHouse        | 2 cycles + 10%, 7-day grace, dashboard lock    | trial/lapse: ingest gate      | `past_due` still active                   | dashboard + stats API; ingestion kept           | One `locked` flag for both    | `on_ee` build + `SELFHOST` cron/queues |
| PostHog    | external billing service                | sum, billing period, +today live           | trust-score grace 0-5 d, then drop at capture  | same                          | closed side                               | ingestion (drop) per resource                   | not read                      | `ee/` license + closed service         |
| Keila      | ledger in core, plans in `extra/`       | prepaid credits, expiry = next bill        | n/a (hard block at 0)                          | hard block                    | no refill; no explicit hold               | campaign delivery (+ features)                  | `account.status` (not traced) | `ENABLE_QUOTAS` + `if_cloud` macros    |
| Plunk      | code const + Stripe                     | calendar month rows, 5-min cache           | Stripe usage billing; per-source customer caps | 1000/month hard cap, 80% warn | first `invoice.payment_failed`            | writes 403, queued mail `FAILED`, reads allowed | one enum column, overwritten  | `STRIPE_ENABLED` keys                  |
| Twenty     | Stripe synced to DB                     | credits + usage-limit table                | credit allowance                               | trial limits                  | `unpaid`/`canceled` (not `past_due`)      | all requests unless allowlisted                 | same `SUSPENDED` state        | `IS_BILLING_ENABLED` + null provider   |
| Formbricks | Stripe mirror in org row                | live COUNT over usage cycle                | billed overage via meter, banner               | Hobby limits, banner          | Stripe dunning only                       | none found                                      | n/a                           | `IS_FORMBRICKS_CLOUD`                  |
| Documenso  | constants + claim rows                  | live COUNT, UTC calendar month             | claim flags                                    | 5 docs hard block (create)    | `INACTIVE` zeroes limits; `past_due` kept | creation only                                   | n/a                           | runtime flag, `Infinity` limits        |
| Chatwoot   | `InstallationConfig` + `Account.limits` | inline counters; daily Redis email counter | per-account / plan / global email cap          | default plan                  | deletion = downgrade                      | email sending (cloud)                           | independent `Account.status`  | `enterprise/` dir + `DEPLOYMENT_ENV`   |
| Infisical  | remote license server, 15-min cache     | meter emit + reconcile                     | not read                                       | not read                      | closed side                               | per-feature `plan.*`                            | n/a                           | `ee/` license, license key             |
| Notifuse   | env vars pushed by control plane        | monthly snapshot, recomputed               | no enforcement site                            | no enforcement site           | closed side                               | none                                            | license never blocks          | zeros = unlimited, pull-only           |
| Postal     | operator-set `send_limit`               | rolling 60 min                             | `Held` + 90% warn + email/webhook              | same                          | n/a                                       | outbound pipeline step                          | separate `suspended_at`       | n/a (no billing)                       |

## Synthesis (inference unless a permalink is given)

### What the prior art says about the two contracts

1. **Hold = out-of-band decision, hot path reads a cached fact.** PostHog (Redis zset of limited tokens, set by a
   15-minute job, read by the capture edge) and Plausible (a `locked` boolean set by daily jobs, read at the
   dashboard/API) both keep limit arithmetic off the request path. Plunk and Keila do the arithmetic **in** the
   send path (live count + threshold in `BillingLimitService`, ledger debit in the delivery transaction). That works,
   but it puts plan numbers into the open binary, which ADR 0009 rules out for 1mail. The ADR's Billing hold matches the
   first group.
2. **Usage snapshot read = pull, signed, with a staleness field.** Notifuse is the only open example of the read side,
   and it matches ADR 0009's shape: aggregate where the data lives, expose to the plane by pull, never push, never
   recount on read, report `computed_at`. Its snapshot is recomputed (not an incremental ledger), which keeps
   the ingest path untouched. ADR 0009 wants the snapshot **finalized at period close**; Notifuse's open month is
   re-countable until it closes.
3. **A finalized-only snapshot cannot drive warnings.** Every product that warns or locks does so on **open-period**
   usage (Plausible 90% on `:current_cycle`, Plunk live count, PostHog `usage + todays_usage`, Postal live window).
   Plausible's _enforcement_ looks at **closed** cycles (last and penultimate), which is the one design that works from
   finalized data alone, at the price of reacting up to a cycle late.
4. **Non-payment follows the provider's terminal state, not the first failure.** Plausible (`past_due` = active),
   Twenty (`unpaid`/`canceled`), Documenso (`INACTIVE`), Formbricks (no lock), Keila (no refill) all let the
   provider's retry schedule be the grace period. Only Plunk reacts to the first failure. That agrees with ADR
   0009's "engages only after a dunning grace period": the plane owns the clock and calls the hold once at the end.
5. **Independent causes need independent storage.** Plunk's single-column design lets one cause overwrite another
   (see D). Twenty reuses one state. Postal and Chatwoot keep suspension separate from limits. ADR 0009/0007's
   "two causes on one chokepoint" is the right shape; the prior art suggests storing **one row or flag per cause**, so
   that lifting one cannot lift another.
6. **Self-host hiding is mostly "absent config = no limits".** Notifuse `0 = unlimited`, Documenso
   `SELFHOSTED_PLAN_LIMITS`, Twenty's null provider, Keila's ledger functions returning `:ok` when off. In 1mail, "no
   Billing hold ever set" is already that default: there is nothing in core to hide except the (EE) snapshot.

### Recommendations for 1mail

**Live-usage read (recommend).**

- Make the read contract return the **open (non-final) period** as well as finalized ones, tagged `final: bool` and
  `computed_at`, so the plane can warn at 80% and cap without waiting for period close. Finalized periods stay the
  billing-grade, immutable figure ADR 0009 asks for.
- Back the open-period number with the same recount-and-store worker approach as Notifuse, or a cached live count
  (Plunk: 5-minute cache, increment on send). Do **not** recount on each read.
- Count in-flight sends against the open period (Plunk counts `PENDING`/`SENDING`, which prevents two broadcasts racing
  past one cap).
- Authenticate as Notifuse does (HMAC over timestamp + path, short skew, derived key); pull-only keeps self-host free of
  outbound calls. `contacts` as high-water mark must be maintained **live** (Cal.com's schema has the same field and
  the same reason, but the maintaining code is not public; so no prior art for the write logic).

**Grace period and thresholds (recommend).**

- _Non-payment:_ no core-side grace at all. The plane sets the hold only when its dunning (Stripe retries or Lago
  `DunningCampaign` `max_attempts` x `days_between_attempts`) is exhausted, as Twenty/Documenso/Plausible do. A
  fixed grace number in core would duplicate the provider's clock. The retry window itself is a provider setting
  (see `billing-provider-options.md`); I do not quote a number from memory.
- _Over-limit on a paid plan:_ prefer **metered overage, not a hold** (Formbricks, Plausible: "an occasional spike
  won't trigger extra charges"). If a hold is wanted for a hard-capped plan, copy Plausible's _shape_: tolerate a margin
  (10%), require persistence (2 cycles) or a time-boxed grace (Plausible: 7 days; PostHog: 0 to 5 days by trust), and
  auto-clear when usage falls back or the period rolls (PostHog: limit until period end; Plausible: unlock when the
  last cycle is under the limit).
- _Warnings:_ the prior art converges on **80%** (Plunk) to **90%** (Plausible, Postal) for "approaching" and 100% for
  "reached", **deduplicated per period** in the data row (Plunk `SET NX` until month end, Postal `*_notified_at`,
  Cal.com `warningSentAt`). Recommend 80% and 100% (and 90% only if one extra step is wanted), sent by the **plane**
  from the open-period read, not by core, to keep plan numbers out of the AGPL binary.

**Hold scope (recommend, partly contradicting ADR 0009's wording).**

- Freeze **outbound sending only** and keep login, dashboard, reads, billing pages and `/collect` (Plausible keeps
  ingestion and only locks the dashboard; Plunk disables writes but keeps reads; ADR 0007 already chose this for abuse).
  Twenty's "refuse everything" suspension is the heavier option, and it is exactly what ADR 0007 rejected.
- Make a held send **pause and resume** (ADR 0007's `Held`), not fail. Plunk's `FAILED` conversion loses the queued
  mail and leaves campaigns with a partial `sentCount`.
- All three surfaces frozen by the same chokepoint is consistent with the prior art; no product spared transactional
  mail on a billing hold (Plunk blocks all writes, Keila meters transactional too). ADR 0009's reasoning ("the grace
  period, not the message type, protects password-reset mail") is not contradicted.
- **Open question for ADR 0009:** it lumps "plan-limit breach" into the hold. For **stock** limits such as `contacts`,
  the prior art blocks the _creation_ (Plausible `ensure_can_add_new_site`
  [teams/billing.ex#L161](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/teams/billing.ex#L161) with enterprise exempt to avoid disruption,
  Documenso on create routes, Twenty `usageLimit` stock service) rather than freezing sends. A contact cap would need
  a second thin write contract (a numeric limit pushed to core), which ADR 0009 does not have, or the plane must treat
  `contacts` as billed-in-arrears with no cap.
- Store holds **per cause** and let each be cleared only by its own source (payment clears the money cause, appeal
  clears suspension), to avoid the Plunk overwrite.

**Free tier and trial (recommend).**

- Prior art treats the free tier as **a plan with a hard cap** (Plunk 1000/month with warn at 80%, Documenso 5
  documents) and treats trial expiry as a **downgrade, not a lock** (Formbricks overrides to a $0 Hobby with
  `create_invoice`; Chatwoot recreates the default plan on `subscription.deleted`). Plausible is the exception: no
  subscription after trial locks the dashboard and later stops ingestion (trial + 14 days).
- For 1mail: plan, trial and free tier all live in the plane. A free tier is a plan whose cap the plane enforces by setting
  a plan-limit hold at 100% and lifting it at period roll. Trial end should change the plane's plan to free, not set a hold.
  Warn before the trial ends (Twenty: 7 days before with card, 1 day without; Plausible: 7, 1 and 0 days).
- Self-hosters are unaffected by construction: no plane, no hold, no snapshot.

## Gaps / unverified

- **Notifuse:** `PLAN_MAX_*` has no enforcement site in this repo; whether the cloud product enforces elsewhere is unknown.
  The usage endpoint meters pageviews and timeline entries, not emails.
- **Cal.com:** the high-water-mark writer and all billing logic are outside the public tree; schema and flag only.
- **PostHog:** non-payment handling and the plan catalogue are in the closed billing service. Whether the UI is locked
  on quota limits was not found in the files read. The capture-edge drop was read; other consumers (recordings, logs)
  were not.
- **Twenty:** the usage-limit and credit-allowance services were identified, but quota arithmetic and period handling
  were not read in detail; no abuse-specific suspension was searched beyond a grep.
- **Formbricks:** "no hard cap" is the code comment at `metering.ts#L38-L45`; response-creation routes were grepped for
  limit checks but not exhaustively traced. The `subscription.deleted` downgrade was inferred from the downgrade code.
- **Infisical:** fail-open vs fail-closed behavior of the fallback plan was not traced; over-limit enforcement beyond
  feature flags (`plan.*`) and identity limits was not found.
- **Keila:** the meaning of `account.status` values (onboarding vs abuse) was not traced; no dunning exists in the app.
- **Postal:** release of `Held` messages was not traced.
- **Chatwoot:** the call sites of `within_email_rate_limit?` were not traced to the send path; billing never sets
  `Account.status = suspended` per a grep, not a proof.
- **Documenso:** whether existing documents remain usable under `INACTIVE` is inferred from the limits being checked only on
  create paths.
- **Stripe/Paddle/Lago behavior:** dunning windows, retry counts and `past_due` -> `unpaid` timing are provider settings;
  not re-verified here (see `billing-provider-options.md`).
- No runtime testing of any codebase; all findings are static reading at the commits listed. Line numbers are valid only
  at those SHAs.
