# Billing provider options for the SaaS control-plane (ADR 0009)

Researched October 2026 from primary sources: source code cloned at each repo's default branch
(shallow clone, commit SHAs below) and the providers' own docs, pricing pages and policy pages
(URLs inline). Permalinks use the commit SHA so line numbers stay valid. Every claim is either
**verified** (URL or permalink given) or explicitly marked **inference** / **not verified**.

Verification pass 2026-10-10: most former "not verified" items were closed against first-party
pages (marked "verified 2026-10-10" inline); the five decision-critical claims were re-fetched
(see "Re-verification log" at the end). The Chrome browser tools were not connected in that
session, so JS-rendered pricing pages were read from the pages' own embedded data or from the
vendor's machine-readable pricing files instead (method stated per row).

| Repo                  | Branch  | Commit                                     |
| --------------------- | ------- | ------------------------------------------ |
| plausible/analytics   | master  | `bc2c7b395caddf7bb4eec2548be7958824d07d4a` |
| pentacent/keila       | main    | `2308bebd2b54edc8ac8d484be5c9b4e972aed7cc` |
| formbricks/formbricks | main    | `2c5c3b104bfccabc44c2460e2488eac14d8910c9` |
| twentyhq/twenty       | main    | `6e36af2e7d425aad7ae038381cd018752684f6f6` |
| documenso/documenso   | main    | `38ecb217effcc53a7164d7123c51336f636bd3b2` |
| chatwoot/chatwoot     | develop | `e212425e7b1814614bdf8b0beea42ce689baad5c` |
| calcom/cal.com        | main    | `54343aa685ae8f33159d2f485ec4a57bad5c574a` |
| dubinc/dub            | main    | `682ab2881b87f344b0178fce4f1438e4dbed94bb` |
| getlago/lago-api      | main    | `4c4f02df9a6dba69fb28e22c67f70338656af0b9` |
| Notifuse/notifuse     | main    | `7516397cc5c1eadc2a6aa67c733261043a630a62` |
| knadh/listmonk        | master  | `d2a0e120c1784c42e0fdf089617985c98b0f244b` |

Base URLs: `https://github.com/<org>/<repo>/blob/<sha>/<path>#L<n>` (branch names above are the
default branch at clone time; only the SHA matters).

## Question

ADR 0009 fixes the boundary: the core meters (Usage snapshot: `emails_sent` sum, `contacts`
high-water-mark, finalized per Workspace and period) and accepts a Billing hold; rating, plans,
invoices, tax and dunning live in an external control-plane on a ready provider, left open between
Stripe, Lago and Paddle. Which provider (or stack) should the control-plane use, given a small
operator team, usage-based pricing on emails sent and/or contacts, EUR and USD, a global audience
that includes Russian-speaking customers, and a product that is itself email marketing? This note
reports facts and a labelled recommendation; it does not change ADR 0009.

## Summary of findings

- **Category acceptance is the deciding constraint, not fees.** The merchant-of-record (MoR)
  providers' acceptable-use lists are written against marketing and outreach products. Paddle
  prohibits "Mass Marketing Products, including but not limited to SMS marketing and Message App
  marketing"; Polar prohibits "Advertising and unsolicited marketing services" (including "bulk
  SMS and automated outreach") and puts "Marketing services" and "Marketing & outreach tools" under
  closer review; Lemon Squeezy bans "Services of any kind (including marketing...)". None of the
  three names email marketing. "Including but not limited to" means silence is **not** clearance:
  treat every MoR option as **blocked until the provider confirms in writing**.
- **Stripe's prohibited and restricted lists do not mention email marketing or lead generation**
  (verified against the page text below). That is a negative result on a list, not an underwriting
  decision; Stripe reviews each account.
- **Open-source SaaS peers that meter usage use Stripe Billing Meters** (Formbricks, Twenty). The
  one email-newsletter peer whose cloud billing code is public (Keila) uses Paddle, specifically
  Paddle Classic, with credit plans, not usage metering.
- **Stripe Billing Meters have no `max` aggregation (sum, count, last only), and that does not
  matter under ADR 0009**: the core already finalizes the contacts high-water-mark, so the plane
  pushes one value per period and uses `last`. What does matter is timing: usage must reach Stripe
  before the subscription's invoice finalizes.
- **Stripe now steers new usage-based integrations to Metronome** and says Billing Meters fit
  pay-as-you-go and "flat fee and overages" models. That covers the planned pricing shape.
- **Lago is AGPLv3, not a PSP and not an MoR.** `max` aggregation is in the open code, but dunning
  campaigns, invoice grace period and Avalara tax are gated by `License.premium?`. It adds a
  service to operate and still needs a PSP and a tax answer.
- **Paddle Billing has no native metering**: usage is custom items added to transactions by your
  own app.
- **Lemon Squeezy and Polar both have native `max` aggregation** but fall under the category risk
  above, and both exclude Russia as a customer country and neither lists Russia as a payout country (Polar and
  Lemon Squeezy verified 2026-10-10; Paddle also lists Russia as an unsupported seller location).
- **Payout country is an open dependency.** Stripe's supported business locations and Managed
  Payments locations exclude Russia. The operator's entity country is unknown and decides which
  options exist at all.

## A. How open-core and open-source peers bill their hosted SaaS

Method: shallow clone, grep for `stripe|paddle|lemonsqueezy|getlago|openmeter|chargebee`, then
read the hit. A repo's code shows what it integrates, not necessarily how its price list is
structured; pricing pages were not verified for these peers.

| Product                      | Provider evidence (verified)                                                                                                                                                                                                                                                                                                                                        | What the code shows                                                                                                                                                        |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Formbricks (survey, AGPL+EE) | Stripe Billing Meters: [`metering.ts` L19](https://github.com/formbricks/formbricks/blob/2c5c3b1/apps/web/modules/ee/billing/lib/metering.ts#L19)                                                                                                                                                                                                                   | `stripeClient.billing.meterEvents.create` per response, idempotent `identifier`, cloud-only guard                                                                          |
| Twenty (CRM, AGPL)           | Stripe SDK `stripe@20.4.1` ([package.json L165](https://github.com/twentyhq/twenty/blob/6e36af2/packages/twenty-server/package.json#L165)), meter entity and [`billing-webhook-price.service.ts` L18](https://github.com/twentyhq/twenty/blob/6e36af2/packages/twenty-server/src/engine/core-modules/billing-webhook/services/billing-webhook-price.service.ts#L18) | A `billingMeter` table mirrored from Stripe via webhooks                                                                                                                   |
| Keila (newsletters, AGPL)    | Paddle Classic: [`paddle/api.ex` L82](https://github.com/pentacent/keila/blob/2308beb/extra/keila_cloud/paddle/api.ex#L82) (`vendors.paddle.com/api`), [`paddle/transaction.ex` L9](https://github.com/pentacent/keila/blob/2308beb/extra/keila_cloud/paddle/transaction.ex#L9) (`classic.paddle.com` spec)                                                         | Plans with `monthly_credits` in [`billing/schemas/plan.ex`](https://github.com/pentacent/keila/blob/2308beb/extra/keila_cloud/billing/schemas/plan.ex); not usage metering |
| Plausible (analytics, AGPL)  | Paddle: [`paddle_api.ex`](https://github.com/plausible/analytics/blob/bc2c7b3/lib/plausible/billing/paddle_api.ex#L161) (`checkout.paddle.com`), `PADDLE_VENDOR_ID` in [`runtime.exs` L253](https://github.com/plausible/analytics/blob/bc2c7b3/config/runtime.exs#L253)                                                                                            | Subscription tiers by plan id (`paddle_plan_id`), vendors-API style (Classic)                                                                                              |
| Documenso (AGPL)             | Stripe: [`.env.example` L153](https://github.com/documenso/documenso/blob/38ecb21/.env.example#L153)                                                                                                                                                                                                                                                                | Stripe API key and webhook secret                                                                                                                                          |
| Chatwoot (open core + EE)    | Stripe: [`Gemfile` L173](https://github.com/chatwoot/chatwoot/blob/e212425/Gemfile#L173), `enterprise/` account billing models                                                                                                                                                                                                                                      | `stripe` gem; plan usage and limits in `enterprise/`                                                                                                                       |
| Cal.com (MIT core + EE)      | Stripe plan prices: [`.env.example` L193](https://github.com/calcom/cal.com/blob/54343aa/.env.example#L193) (`NEXT_PUBLIC_STRIPE_*_PLAN_PRICE`)                                                                                                                                                                                                                     | Stripe plan price ids; the app also ships Stripe as a customer-facing integration                                                                                          |
| Dub                          | Stripe: [`apps/web/package.json` L136](https://github.com/dubinc/dub/blob/682ab28/apps/web/package.json#L136), `lib/stripe/cancel-subscription.ts`                                                                                                                                                                                                                  | Subscription management; Stripe is also a customer-facing product feature                                                                                                  |
| Listmonk (AGPL)              | No billing code found (self-host only)                                                                                                                                                                                                                                                                                                                              | n/a                                                                                                                                                                        |
| Notifuse (BSL 1.1 now)       | Not SaaS billing: a Stripe mention only in a code comment in [`license_service.go` L25](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/license_service.go#L25) about license-key grace                                                                                                                                                          | Licence keys; the comment asserts Stripe retry timing, not verified here                                                                                                   |

Not checked (listed in the request, no primary evidence gathered): Mautic/Acquia, GitLab,
Supabase, Sentry, Infisical, n8n, Appsmith, PostHog, BillionMail. A grep of the Mautic repo found
no Stripe or Paddle integration (matches were CSS "stripe"); PostHog's `hogli.yaml` only mentions
Stripe billing in comments, which is not enough to cite.

Reading (inference): Stripe is the default for open-core SaaS (7 of 9 hits above); the two Paddle
users are the analytics and newsletter products. Neither Keila's nor Plausible's public code shows
Paddle's acceptance decision for an email-marketing product today, and Keila uses Paddle Classic,
not the current Paddle Billing product.

## B. Candidates

### B1. Stripe Billing + Stripe Tax (Stripe is PSP, not MoR)

- **Fees (verified).** Billing is 0.7% of Billing volume pay-as-you-go (or from $620 per month on
  a one-year contract); basic usage metering is included up to 100M events per month; card
  processing is 2.9% + 30c per successful card charge
  ([stripe.com/billing/pricing](https://stripe.com/billing/pricing)). Stripe Tax Basic is 0.5% per
  transaction on no-code integrations such as Billing, where you are registered to collect tax, or
  50c per transaction via API ([stripe.com/tax/pricing](https://stripe.com/tax/pricing)). Dispute
  received fee $15 ([stripe.com/pricing](https://stripe.com/pricing)).
- **Tax (verified).** Stripe Tax calculates and collects; **registration and filing stay with the
  operator** (Tax Complete tiers bundle registrations, with extra fees outside the US). This is
  the cost of not using an MoR.
- **Metering (verified).** Meter aggregation is `sum`, `count` or `last`; ingestion is `raw` or
  `pre-aggregated` (latest event per hour or day wins); a meter cannot be edited after
  configuration except its display name
  ([meters/configure](https://docs.stripe.com/billing/subscriptions/usage-based/meters/configure)).
  Event timestamps must be within the past 35 days; usage must be reported before the
  subscription's invoice finalizes or it may appear on no invoice; the meter endpoint allows
  1000 calls per second per account and one concurrent call per customer per meter; `identifier`
  gives idempotency
  ([recording-usage-api](https://docs.stripe.com/billing/subscriptions/usage-based/recording-usage-api)).
  Canceling events works only within 24 hours and never changes a finalized invoice (configure
  page).
- **Stripe's own steer (verified).** Metronome is "Stripe's primary usage-based billing platform
  for all new integrations"; basic Billing Meters "work best for ... pay-as-you-go pricing" and
  also support flat fee plus overages; prepaid credit drawdown, commits and real-time usage need
  Metronome ([compare-metronome](https://docs.stripe.com/billing/subscriptions/usage-based/compare-metronome),
  [recording-usage](https://docs.stripe.com/billing/subscriptions/usage-based/recording-usage)).
- **Acceptable use (verified on page text).** The restricted-businesses page lists adult content,
  multilevel marketing, telemarketing and "Sales of online traffic or engagement", and does not
  list email marketing, marketing services, mailing lists or lead generation
  ([stripe.com/legal/restricted-businesses](https://stripe.com/legal/restricted-businesses)).
  Account-level underwriting is separate and not publicly specified.
- **Go SDK (verified).** `stripe/stripe-go`, MIT, requires Go 1.24+
  ([github.com/stripe/stripe-go](https://github.com/stripe/stripe-go)).
- **Smart Retries (verified 2026-10-10).** "The recommended default setting is 8 tries within 2
  weeks"; the policy can be 1 week, 2 weeks, 3 weeks, 1 month or 2 months with a set number of
  retries; a custom schedule allows up to three retries, each N days after the previous. When
  recovery fails the subscription is cancelled, marked `unpaid`, left `past_due` or paused, per
  settings, after the maximum number of days of the retry schedule. Hard declines (for example
  `stolen_card`, `authentication_required`) are not retried until a new payment method exists.
  Failures are signalled by `invoice.payment_failed` (`attempt_count`, `next_payment_attempt`; with
  Billing automations `next_payment_attempt` moves to `invoice.updated`)
  ([smart-retries](https://docs.stripe.com/billing/revenue-recovery/smart-retries)). Implication
  for ADR 0009: the dunning grace period before the Billing hold should be at least the retry
  window (default 2 weeks) or the plane must read the subscription status instead.
- **Invoice finalization grace period for metered usage (verified 2026-10-10).** "All invoices
  have a default finalization grace period of 1 hour"; usage for the ended period can still be
  reported in that window and is included only on subscription-cycle (and schedule phase
  transition) invoices with metered items; the delay is configurable **up to 72 hours**, and rules
  can target "Has a metered price" invoices; late usage "we don't add that usage to the finalized
  invoice or the next invoice"; a mid-cycle price change drops grace-period usage; the first
  invoice of a charge-automatically subscription finalizes immediately
  ([configure-grace-period](https://docs.stripe.com/billing/subscriptions/usage-based/configure-grace-period)).
  So core period close must push the snapshot within 1 hour by default, or within up to 72 hours
  if the rule is set.
- **Business location (verified, re-fetched 2026-10-10).** [stripe.com/global](https://stripe.com/global)
  lists 51 countries/regions (Australia, Austria, Belgium, Brazil, Bulgaria, Canada, Croatia,
  Cyprus, Czech Republic, Denmark, Estonia, Finland, France, Germany, Gibraltar, Greece, Hong Kong,
  Hungary, Ireland, Italy, Japan, Latvia, Liechtenstein, Lithuania, Luxembourg, Malaysia, Malta,
  Mexico, Netherlands, New Zealand, Norway, Poland, Portugal, Romania, Singapore, Slovakia,
  Slovenia, Spain, Sweden, Switzerland, Thailand, UAE, UK, US; India and Indonesia as Preview;
  Cote d'Ivoire, Ghana, Kenya, Nigeria and South Africa as "Extended network"). **Russia is not
  listed.**
- **No legal entity yet (verified 2026-10-10).** Stripe: "You can use Stripe to sell a product if
  you have not established a separate business entity to do so"; at signup you pick **sole
  proprietorship** ("If you do not have a separate entity and operate by yourself") or partnership
  ([support.stripe.com](https://support.stripe.com/questions/selling-on-stripe-without-a-separate-business-entity)).
  That page names no countries; Stripe's US help pages add US-specific rules (SSN/ITIN/EIN, an
  unregistered business needs an owner physically in the US), and requirements elsewhere vary by
  country (not verified per country). Managed Payments is separate and lists only business
  locations, not entity types, but excludes platform and Express accounts.

### B2. Stripe Managed Payments (Stripe as MoR)

- **Verified.** Stripe is merchant of record, handles sales tax, VAT and GST in more than 80
  countries, plus fraud, disputes and transaction support; works with Stripe Billing subscriptions
  and **only via Checkout or Payment Links**. It does not support invoice items on a Customer
  attached to a Managed Payments subscription, one-off invoices outside the billing period,
  subscriptions created outside Checkout, Connect, third-party tax integrations or custom
  checkout domains. Eligible products are digital: SaaS has tax code `txcd_10103001`; unsupported
  categories include "Professional services, such as consulting, marketing, design, development,
  or tech support". Business must be in a supported location (US, CA, most of Europe, AU, HK, JP,
  SG); customers in Russia, China and several sanctioned places are excluded; Stripe may review
  business type and geography, and keeps a dispute-rate requirement
  ([overview](https://docs.stripe.com/payments/managed-payments),
  [how it works](https://docs.stripe.com/payments/managed-payments/how-it-works),
  [eligibility](https://docs.stripe.com/payments/managed-payments/eligibility)). Fee:
  **3.5% per successful Managed Payments transaction in addition to Payments fees**
  ([stripe.com/pricing](https://stripe.com/pricing)).
- **Metered prices under Managed Payments: still not verified, now narrowed (2026-10-10).** Read
  again: overview, how-it-works, eligibility, set-up and update-checkout. None mentions meters,
  usage-based prices or meter events as supported or unsupported. The overview table says support
  for subscriptions is "Available with Billing", and the unsupported list covers invoice items on
  a `Customer`, one-off invoices outside the billing period, subscriptions created outside
  Checkout or Payment Links, Connect and third-party tax. Metered usage via meter events uses
  none of those mechanisms, so it is plausible but undocumented; the removed Checkout parameters
  include `subscription_data.invoice_settings`, which is not what the grace period uses. Needs a
  Stripe confirmation (support or sales) before Managed Payments is considered for usage billing.
- **Inference.** 1mail sells software (SaaS), not a marketing service, so the "marketing"
  exclusion should not apply, but that is for Stripe to confirm. A third-party site quotes
  "5% + 50c" for Managed Payments; Stripe's own pricing page says 3.5% on top of processing.
  Use Stripe's figure.

### B3. Paddle Billing (MoR)

- **Fees (verified).** 5% + 50c per Checkout transaction, includes global tax and regulatory
  compliance, fraud, chargebacks and buyer support; products under $10 need bespoke pricing
  ([paddle.com/pricing](https://www.paddle.com/pricing)).
- **Acceptable use (verified).** The "what am I not allowed to sell" page prohibits category 7
  "Advertising and marketing", including "Mass Marketing Products, including but not limited to
  SMS marketing and Message App marketing" and "Marketing Services and Schemes"; email marketing
  and lead generation are not named
  ([paddle.com/help/.../what-am-i-not-allowed-to-sell-on-paddle](https://paddle.com/help/start/intro-to-paddle/what-am-i-not-allowed-to-sell-on-paddle)).
  **Treat as blocked pending written confirmation.**
- **Usage billing (verified).** No native metering: "track usage in your app using a metering
  library like OpenMeter, then bill for it in Paddle"; usage is a custom line item added to the
  upcoming transaction; pure usage billing is possible through a zero-dollar subscription
  ([developer.paddle.com/llms-full.txt](https://developer.paddle.com/llms-full.txt), AI companies
  and SaaS pages). Under ADR 0009 that is workable (the plane adds the snapshot figure as a
  custom item) but the app owns metering semantics and idempotency.
- **Go SDK (verified).** Official `github.com/PaddleHQ/paddle-go-sdk/v5`, Apache-2.0
  ([docs](https://developer.paddle.com/sdks/libraries/go.md),
  [repo](https://github.com/PaddleHQ/paddle-go-sdk)).
- **Seller locations (verified 2026-10-10).** Paddle has no positive payout-country list. Its
  help page says "Paddle works with software businesses anywhere in the world with the exception
  of the unsupported countries listed below" and "Paddle is unable to support suppliers operating
  from the below countries": Afghanistan, Antarctica, Belarus, Burma (Myanmar), Central African
  Republic, Cuba, Crimea, DR Congo, Donetsk, Haiti, Iran, Iraq, Kherson, Libya, Luhansk, Mali,
  Netherlands Antilles, Nicaragua, North Korea, **Russia**, Somalia, South Sudan, Sudan, Syria,
  Venezuela, Yemen, Zaporizhzhia, Zimbabwe; buyers in these regions are blocked too; the list "may
  change"
  ([which-countries-are-supported-by-paddle](https://www.paddle.com/help/start/intro-to-paddle/which-countries-are-supported-by-paddle)).
  Search snippets mention a payout-currency list (USD, EUR, GBP, AUD, CAD, with a later FAQ
  listing more); the two pages disagree and were not reconciled.
- **Not verified:** dunning schedule, webhook signature details.

### B4. Lemon Squeezy (MoR, now tied to Stripe)

- **Fees (verified).** 5% + 50c, MoR with tax liability taken on, "Transactions made outside of
  the US may encounter small additional fees"
  ([lemonsqueezy.com/pricing](https://www.lemonsqueezy.com/pricing)). The page carries a
  "2026 Update: Lemon Squeezy + Stripe Managed Payments" banner whose text I could not read; the
  roadmap of the product is therefore **not verified** and should be checked with them.
- **Usage billing (verified).** Native usage records with aggregation options `sum`, most recent
  in period, most recent ever and **maximum usage during period**; billing is retrospective at
  renewal ([usage-based-billing](https://docs.lemonsqueezy.com/help/products/usage-based-billing)).
  This is the closest native match to a high-water-mark metric.
- **Acceptable use (verified).** Prohibits "Services of any kind (including marketing, design,
  web development, consulting or other related services)" and "Advertising in newsletters"; email
  marketing software is not named
  ([prohibited products](https://docs.lemonsqueezy.com/help/getting-started/prohibited-products)).
  **Blocked pending written confirmation.**
- **Geography (verified 2026-10-10).** Bank payouts in 99 listed countries (EU/EEA members, UK,
  US, Canada, Switzerland, Turkey, UAE, Israel, Singapore, Japan, Korea, India with a caveat,
  many others), plus PayPal payouts in 200+ countries and regions; "can offer services to
  merchants and affiliates who can receive bank or PayPal payouts". **Russia is not in the bank
  list** and is on the unsupported customer countries list (with Central African Republic, Cuba,
  North Korea, DR Congo, Eritrea, Guinea-Bissau, Iran, Iraq, Lebanon, Libya, Mali, Somalia, South
  Sudan, Sudan, Syria, Yemen). Whether a Russian-based seller could be paid by PayPal was not
  verified ([supported-countries](https://docs.lemonsqueezy.com/help/getting-started/supported-countries)).

### B5. Polar (MoR)

- **Fees (verified).** Starter 5% + 50c; Pro $20 per month 3.8% + 40c; Growth $100 3.6% + 35c;
  Scale $400 3.4% + 30c; +1.5% for international cards; payouts via Stripe Connect Express with a
  $2 monthly payout fee, 0.25% + $0.25 per payout and 0.25% to 1% cross-border
  ([polar.sh/docs/merchant-of-record/fees](https://polar.sh/docs/merchant-of-record/fees)).
- **Usage billing (verified).** Meters with count, sum, avg, min, max, unique
  ([meters](https://polar.sh/docs/features/usage-based-billing/meters)).
- **Acceptable use (verified).** Prohibits "Advertising and unsolicited marketing services",
  including "bulk SMS and automated outreach"; lists "Marketing services" and "Marketing &
  outreach tools" as restricted and reviewed
  ([acceptable-use](https://polar.sh/docs/merchant-of-record/acceptable-use)). **Blocked pending
  written confirmation**, with the highest risk of the MoR set because "outreach tools" is named.
- **Geography (verified).** Payouts through Stripe Connect Express in supported countries;
  Russia is excluded as a payer and not a payout country
  ([supported countries](https://polar.sh/docs/merchant-of-record/supported-countries); quote
  re-fetched 2026-10-10: "We support payments globally except from countries with US sanctions
  (Cuba, Russia, Iran, North Korea, and Syria)"). Go SDK not verified.

### B6. Lago (rating layer; AGPLv3; self-hosted or managed)

- **License (verified).** Platform AGPLv3; agent SDKs, MCP server and CLI MIT
  ([github.com/getlago/lago](https://github.com/getlago/lago)). Same license family as 1mail.
- **Not a PSP or MoR (verified).** "Lago is not a payment service provider"; it integrates
  Stripe Payments, GoCardless and Adyen, or emits webhooks to any PSP (Paddle, Razorpay, ...)
  ([payments overview](https://github.com/getlago/lago-docs/blob/502a9e7/docs/guide/11_payments/overview.md);
  note that docs repo is stale, July 2023, so only this licensing and architecture statement is
  used from it). Current `lago-api` also has Cashfree, Flutterwave, GoCardless, Adyen, MoneyHash
  and Stripe payment-provider services under
  [`app/services/payment_providers`](https://github.com/getlago/lago-api/tree/4c4f02d/app/services/payment_providers).
- **Aggregation (verified in code).** `max_agg` exists in the open code
  ([`billable_metric.rb` L32](https://github.com/getlago/lago-api/blob/4c4f02d/app/models/billable_metric.rb#L32)).
- **Premium gating (verified in code).** Dunning campaigns return early without a premium licence
  ([`process_dunning_campaigns_job.rb` L8](https://github.com/getlago/lago-api/blob/4c4f02d/app/jobs/clock/process_dunning_campaigns_job.rb#L8)),
  invoice grace period is premium
  ([`organizations/update_service.rb` L60](https://github.com/getlago/lago-api/blob/4c4f02d/app/services/organizations/update_service.rb#L60)),
  payment requests are premium
  ([`payment_requests/create_service.rb` L72](https://github.com/getlago/lago-api/blob/4c4f02d/app/services/payment_requests/create_service.rb#L72)),
  and the pricing page lists automatic dunning and tax integrations as add-ons "on demand"
  ([getlago.com/pricing](https://www.getlago.com/pricing); no prices published, "Contact us").
  I did not find a gate on plain payment retries in a grep of `app/services`; not exhaustively
  verified. Tax: only Avalara is named as an integration
  ([github.com/getlago/lago](https://github.com/getlago/lago)).
- **Reading (inference).** Lago adds one more service to run (Rails, Postgres, Redis, workers),
  does not remove the PSP or the tax problem, and puts dunning behind a commercial licence. It
  becomes attractive only if the operator needs a PSP Stripe will not serve (for example local
  payment rails) or wants billing logic fully under its own control.

### B7. OpenMeter, Orb, Metronome (metering engines)

- **OpenMeter (verified).** Apache-2.0, beta ("Releases are beta and can include breaking
  changes"); runtime needs Kafka, ClickHouse and Postgres; Compose and Helm are labelled
  development only ([github.com/openmeterio/openmeter](https://github.com/openmeterio/openmeter)).
  Wrong weight class for a team that already has the aggregate in Postgres.
- **Metronome (verified).** Stripe's recommended engine for new usage-based integrations;
  pricing is custom ([compare-metronome](https://docs.stripe.com/billing/subscriptions/usage-based/compare-metronome),
  [stripe.com/billing/pricing](https://stripe.com/billing/pricing)).
- **Orb (verified 2026-10-10).** Billing platform for usage-based, seat-based and hybrid pricing.
  Metrics: the "how Orb works" page lists a **MAX** metric type ("Peak value during the period",
  example "storage high-water mark"); the no-code builder offers count, unique, sum, max, min and
  average per Orb's support article, and custom SQL metrics for anything else. Pricing is not
  published: Core, Advanced and Enterprise tiers are all "Custom pricing", based on "billings and
  events" plus a platform fee on the upper tiers
  ([docs.withorb.com/how-orb-works](https://docs.withorb.com/how-orb-works),
  [support.withorb.com](https://support.withorb.com/articles/7129471641-understanding-billable-metrics-in-orb),
  [withorb.com/pricing](https://www.withorb.com/pricing)). Orb is a rating/metering layer, not a
  PSP or MoR; it was not researched for its tax or payment-provider coverage. Same reading as
  Metronome: it duplicates a snapshot the core has already finalized.
- **Reading (inference).** ADR 0009 already rejected streaming raw events to an external plane;
  a metering engine adds a second aggregator over data the core has already finalized.

## Fit with ADR 0009's two contracts

- **Usage snapshot (read), provider-agnostic.** The plane reads each finalized (Workspace,
  period, metric) row and pushes it: `emails_sent` as one event with `sum`, `contacts` as one
  event with `last`. A provider's lack of `max` is irrelevant; Paddle needs a custom item,
  Lemon Squeezy or Polar could take the raw figure with their own `max`.
- **Timing risk (verified fact, design inference).** Stripe does not apply usage that arrives
  after the invoice finalizes. Core period close must precede subscription invoice finalization,
  by aligning periods with the subscription's billing cycle and/or configuring Stripe's invoice
  finalization grace period (page above, not read). If core's close is slower, the plane must
  hold the invoice open.
- **Billing hold (write).** Stripe invoice payment-failure webhooks drive the plane; the plane
  waits out the dunning grace period (ADR 0009) and then calls the core hold endpoint; payment
  success clears it. This is plain webhook logic on any provider; no provider constrains it.
- **Reproducibility.** Stripe meter events carry an `identifier`; use
  `usage-snapshot-id:period:metric` so re-pushes are idempotent.

## Recommendation

**Primary: Stripe Billing (basic usage-based, Billing Meters) with Stripe Tax, as a small
Go control-plane service holding the Stripe secrets. Do not take Lago, OpenMeter or Metronome in
v1.** Treat any MoR as an option that opens only if its provider confirms in writing that
bulk email marketing software is an accepted category.

Why (verified facts first, inference marked):

- It is the only candidate whose published restricted-business list does not name marketing
  tools, and it is the pattern peers with usage metering use (Formbricks, Twenty).
- Billing Meters cover the planned shape (pay-as-you-go and flat fee plus overages) per Stripe's
  own comparison; the pre-aggregated Usage snapshot removes the need for `max`.
- Cost stack: card 2.9% + 30c, Billing 0.7%, Stripe Tax 0.5% per transaction, plus tax
  registration and filing by the operator. The cost of not having an MoR is **operator
  workload** (tax registrations, filings), not a rate.
- (Inference) The control-plane is thin: webhook handler, snapshot pusher, hold toggler, no
  rating engine.

| Option                          | Tax / MoR                 | Metering fit with ADR 0009                | Category risk (email marketing)                  | Self-host  | Verdict                                             |
| ------------------------------- | ------------------------- | ----------------------------------------- | ------------------------------------------------ | ---------- | --------------------------------------------------- |
| Stripe Billing + Tax            | Operator files            | Good: pre-aggregated push, `sum` / `last` | Not on prohibited list; underwriting unspecified | No         | **Primary**                                         |
| Stripe Managed Payments         | Stripe is MoR, 3.5% extra | Unknown: meters not documented            | Not named; "marketing" services excluded         | No         | Ask Stripe; fallback if tax load hurts              |
| Paddle Billing                  | MoR, 5% + 50c             | Custom items from app; no native metering | "Mass Marketing Products ... not limited to" SMS | No         | Blocked until written confirmation                  |
| Lemon Squeezy                   | MoR, 5% + 50c             | Native incl. `max`                        | "Services of any kind (including marketing)"     | No         | Blocked until written confirmation; roadmap unclear |
| Polar                           | MoR, 3.4-5% + 30-50c      | Native incl. `max`                        | "Marketing & outreach tools" restricted          | No         | Blocked until written confirmation                  |
| Lago (self-hosted) + Stripe PSP | Operator files            | Native incl. `max`                        | Inherits PSP                                     | Yes (AGPL) | Fallback if no Stripe or custom PSP                 |
| OpenMeter / Metronome / Orb     | n/a                       | Duplicates the snapshot                   | n/a                                              | Partly     | Not needed                                          |

## Risks

- **Stripe account risk (inference).** Email marketing attracts disputes and spam complaints;
  Stripe can restrict an account whose dispute rate rises. Keep the Stripe account separate from
  other businesses, publish AUP and anti-spam terms, and keep suppression and complaint handling
  in the product.
- **Tax load (verified fact, operator impact inference).** With Stripe as PSP, the operator owns
  VAT/sales-tax registrations and filings in each jurisdiction it crosses the threshold in.
- **Metering product churn.** Stripe is steering new usage integrations to Metronome; basic
  Billing Meters remain "fully supported" for existing users, and the plane's pre-aggregated
  design keeps a later swap local to the plane.
- **Timing.** See above: usage after invoice finalization may not be billed.
- **Meter immutability.** A configured Stripe meter cannot be changed except for its display
  name; changing the event name, formula or value key means a new meter.

## Unknown and needs the user's input

1. **Operator legal entity and bank country.** Stripe supports a fixed list of business
   locations that does not include Russia; Managed Payments has a narrower list (EU/EEA, UK,
   US, CA, AU, HK, JP, SG). If the entity is Russian, none of the Stripe, Polar or Managed
   Payments paths opens; verified 2026-10-10: Paddle lists Russia among unsupported seller
   locations, and Lemon Squeezy's bank-payout list excludes Russia (a PayPal payout for a
   Russian-based seller was not verified). Stripe takes a sole proprietor or individual with no
   separate legal entity, but only in its 51 supported countries. Do not choose before this is
   answered.
2. **Whether Russian-speaking customers must pay from Russia.** Managed Payments excludes
   Russia as a customer country; Polar excludes Russia; Stripe's own Payments eligibility for
   cardholders in Russia was not verified. If this matters, a PSP-agnostic rating layer (Lago,
   per its "any PSP via webhooks" statement) becomes the argument for it. No Russian PSP is named
   here because none was researched.
3. **Written answers from providers** on whether bulk email marketing software is an accepted
   category: Paddle, Lemon Squeezy, Polar (and Stripe's underwriting expectations).
4. **Pricing shape.** Emails only, contacts only, or tiered base plus overage on both. Stripe
   basic meters handle all three; prepaid credits and commits need Metronome.
5. **Willingness to own tax registrations.** If no, the choice shifts to an MoR that accepts the
   category, whichever it is.
6. **Stripe Managed Payments plus metered prices:** still undocumented after re-reading five
   Managed Payments pages (see B2); needs a Stripe confirmation.
7. ~~Stripe Smart Retries schedule and the invoice finalization grace period~~ read 2026-10-10
   (see B1): default 8 retries within 2 weeks; finalization grace 1 hour by default, up to 72
   hours. Size the Billing hold's dunning grace period against those numbers.
8. **Does each provider permit email-marketing SaaS?** Policy text re-read 2026-10-10; none of
   Paddle, Polar, Lemon Squeezy or Stripe names email marketing, so there is no explicit permit or
   prohibit. Only the providers' written answers close this.

## Competitor pricing metrics

Researched 10 October 2026 (fetch date for every URL below) from each product's own pricing page
or repo; no blogs. A page that returned 403, rendered prices only in JavaScript, or came back
garbled through the fetch tool is marked **not verified**. Question: which metric do open-source
email products and incumbents bill, and is "unlimited contacts, price on sends, optional BYO
provider" an existing position or an open niche (ADR 0009 snapshots both `emails_sent` and
`contacts`, so either is possible without a core change).

### Per-product facts

| Product (self-host license)                                        | Metric billed                                                                                         | Free tier                                                                  | Entry price                                                                                                                                                                                                                                     | Contacts unlimited?                                                                                                                 | BYO provider                                                                                                            |
| ------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Keila (AGPLv3) [keila.io/pricing][keila]                           | Emails per month, 6 tiers                                                                             | None ("there is no free plan")                                             | EUR 8/mo for 2,000 emails (incl. tax); EUR 64 for 50,000; EUR 256 for 250,000                                                                                                                                                                   | Yes, "Unlimited" contacts and projects on every plan                                                                                | "Send with Keila or bring your own email server" on all plans; SES/SMTP not named; no BYO discount stated               |
| Plunk (AGPL-3.0) [useplunk.com/pricing][pl]                        | Emails sent, USD 0.001 per email, no base fee                                                         | 1,000 emails/mo, Plunk branding                                            | Pay as you go, no minimum                                                                                                                                                                                                                       | Yes; "never costs more" as the list grows                                                                                           | Not mentioned on the pricing page or docs home                                                                          |
| Notifuse (BSL 1.1 per table above) [cloud][nc]                     | Active contacts (no activity in 30 days excluded); emails unlimited                                   | 14-day trial, card required; self-host free                                | Lite USD 19/mo (USD 16 yearly) for 2,500 active contacts                                                                                                                                                                                        | No: limits 2,500 / 10,000 / 25,000 / 50,000 active                                                                                  | Yes: SES, Postmark, SendGrid, Mailgun, Mailjet, SparkPost, SMTP; "unlimited email through your own provider" [main][nm] |
| Dittofeed (license not stated on page) [dittofeed.com/pricing][df] | Unique users per month (FAQ also mentions events, no rate)                                            | 14-day trial; self-host free; no free cloud plan                           | Cloud Pro USD 75/mo incl. 10,000 unique users; overage USD 0.004 per user to 100k                                                                                                                                                               | No (users are the metric)                                                                                                           | Not mentioned                                                                                                           |
| Loops (closed) [loops.so/pricing][lo]                              | Subscribed contacts; paid plans have no send cap                                                      | 1,000 contacts, 4,000 sends per rolling 30 days, "Powered by Loops" footer | USD 49/mo for 1,001-5,000 contacts; USD 99 for 5,001-10,000; USD 149 for 10,001-15,000; USD 249 for 25,001-50,000; paid plans send unlimited ([loops.so/pricing.md](https://loops.so/pricing.md), verified 2026-10-10)                          | Free tier accepts unlimited uploads but sends to newest 1,000                                                                       | n/a, closed SaaS                                                                                                        |
| Mautic (GPL, per repo, not re-fetched) [mautic.org/pricing][mt]    | Managed hosting by partner Dropsolid; tiers bundle contacts and emails                                | None                                                                       | Essential EUR 247.50/mo billed annually (EUR 275 quarterly): up to 50K contacts and 50K emails                                                                                                                                                  | No: "up to 50K", Professional extendable to 3M                                                                                      | Enterprise tier lists "custom email provider" support; lower tiers not stated                                           |
| Sendy (closed source, one-time) [sendy.co][se]                     | One-time licence, USD 69 per installation domain; SES cost separate                                   | None; SES gives 3,000 free emails/mo for 1 year (page claim)               | USD 69 once; install service USD 79                                                                                                                                                                                                             | Yes: "Self-hosted means unlimited everything"                                                                                       | Core design: Amazon SES, page quotes "$1.6 per 10,000 emails" (SES's own page not fetched)                              |
| Listmonk (AGPLv3) [listmonk.app][lm]                               | None: no hosted offering by the project                                                               | n/a                                                                        | n/a; page lists third-party one-click hosts, "no affiliation"                                                                                                                                                                                   | n/a                                                                                                                                 | n/a                                                                                                                     |
| Postal (MIT) [github.com/postalserver/postal][po]                  | None: no hosted offering advertised                                                                   | n/a                                                                        | n/a                                                                                                                                                                                                                                             | n/a                                                                                                                                 | n/a (it is itself an SMTP server)                                                                                       |
| Mailchimp [mailchimp.com/pricing/marketing][mc]                    | Contacts, with send cap as a multiple (10x / 12x / 15x contacts)                                      | 250 contacts, 500 sends/mo or 250/day                                      | Essentials USD 13 and Standard USD 20 are the 500-contact base prices; Premium USD 350, whose base "includes 10,000 contacts" (verified 2026-10-10, see note)                                                                                   | No                                                                                                                                  | n/a                                                                                                                     |
| MailerLite [mailerlite.com/pricing][ml]                            | Active subscribers; send cap 10x subscribers on Comfort, unlimited (fair use) on Power                | 250 subscribers, 2,500 emails/mo                                           | Comfort from USD 12/mo, Power from USD 25/mo at 500 subscribers; at 10,000 subscribers USD 89 (Comfort) and USD 129 (Power) monthly, USD 80.10 and USD 116.10 with yearly billing (page's embedded price table, verified 2026-10-10)            | No                                                                                                                                  | n/a                                                                                                                     |
| Kit [kit.com/pricing][kt]                                          | Subscribers                                                                                           | Free up to 10,000 subscribers                                              | Creator USD 33/mo and Pro USD 66/mo at 1,000 subscribers (yearly billing); at 10,000 subscribers Creator USD 139 and Pro USD 189 monthly list, i.e. USD 1,390 and USD 1,890 per year (page's JS bundle, verified 2026-10-10)                    | No                                                                                                                                  | n/a                                                                                                                     |
| Brevo [brevo.com/pricing][br]                                      | Emails per month (Starter from 5,000, Professional from 150,000; contacts shown as included quantity) | Free plan with 100,000 marketing contacts included (pricing API)           | Starter USD 9/mo, Standard USD 18/mo, Professional USD 350/mo base, plus email-volume tiers (Starter 40,001-50,000 emails: USD 47; Standard: USD 79) ([pricing.md](https://www.brevo.com/pricing.md), generated 2026-09-29, fetched 2026-10-10) | Free plan 100,000 contacts; paid-plan contact caps **not verified** (500 included at USD 0, no price per extra contact in the feed) | n/a                                                                                                                     |
| Drip [drip.com/pricing][dr]                                        | **not verified** (403 to every fetch, 2026-10-10; Chrome tools not connected)                         | **not verified**                                                           | **not verified**; only Drip's own blog snippets say "from $39 per month for 1 to 2,500 active people" and bill by active people, which is marketing text, not the pricing page                                                                  | **not verified**                                                                                                                    | **not verified**                                                                                                        |

General open-core metric examples (verified on pricing pages): **Plausible** bills monthly
pageviews (Starter USD 9/mo up to 10k pageviews, no free tier, 30-day trial)
([plausible.io][pa]); **Cal.com** bills per seat (free for 1 user; Teams USD 12 and Organizations
USD 28 per user per month, yearly option) ([cal.com/pricing][ca]). Neither page states the
self-host licence terms; see the licence rows above for the repo-level facts. In both cases the
metric is the unit the customer's value scales with (traffic, seats), not an artefact of storage.

Notes on weak spots:

- **Correction 2026-10-10:** the earlier note read Mailchimp's Essentials USD 13 and Standard
  USD 20 as 10,000-contact prices. They are the 500-contact base prices (Mailchimp's help page:
  "Essentials and Standard base prices cover 500 contacts"; only Premium's base "includes 10,000
  contacts"). The slider prices for Essentials and Standard at 10,000 contacts load by XHR and
  were not read, so they stay **not verified**; Premium USD 350 is verified.
- MailerLite, Kit: closed 2026-10-10 by reading the pricing data embedded in each page's own
  HTML or JS bundle (curl with a browser User-Agent), not a rendered slider, so a UI promotion
  may differ. Kit's yearly figure follows the page's own formula (10 x monthly).
- Brevo: closed from Brevo's published machine-readable pricing file; the additive reading of
  base plan plus email tier is an inference.
- Loops: closed from loops.so/pricing.md. Drip stays open (403, no browser).
- Notifuse's licence (re-checked 2026-10-10 against the repo `LICENSE` file): "All other files in
  this repository constitute the 'Licensed Work' and are distributed under the Business Source
  License 1.1"; "Notifuse v39.x and every earlier release are licensed under AGPL-3.0-or-later and
  remain so permanently"; only `web_analytics_sdk/` is AGPL-3.0-or-later. Confirmed.

[keila]: https://www.keila.io/pricing
[pl]: https://www.useplunk.com/pricing
[nc]: https://www.notifuse.com/pricing/cloud
[nm]: https://www.notifuse.com/pricing
[df]: https://www.dittofeed.com/pricing
[lo]: https://loops.so/pricing
[mt]: https://www.mautic.org/pricing
[se]: https://sendy.co
[lm]: https://listmonk.app
[po]: https://github.com/postalserver/postal
[mc]: https://mailchimp.com/pricing/marketing/
[ml]: https://www.mailerlite.com/pricing
[kt]: https://kit.com/pricing
[br]: https://www.brevo.com/pricing/
[dr]: https://www.drip.com/pricing
[pa]: https://plausible.io/#pricing
[ca]: https://cal.com/pricing

### Comparison at 10,000 contacts and 50,000 sends per month

| Product                    | Monthly price for that load                                                                                       | Basis                                                                                                                                                                                                  |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Keila L                    | EUR 64 (incl. tax), contacts irrelevant                                                                           | Verified: tier is exactly 50,000 emails                                                                                                                                                                |
| Plunk                      | USD 50                                                                                                            | Inference: 50,000 x USD 0.001; free 1,000 not netted out                                                                                                                                               |
| Notifuse Cloud Starter     | USD 49 (USD 39 yearly) + provider cost                                                                            | Verified tier (10,000 active contacts, unlimited emails via BYO); SES at the Sendy-page rate of USD 1.6 per 10,000 would add about USD 8 (inference)                                                   |
| Sendy + SES                | USD 69 once, then about USD 8/mo of SES                                                                           | Verified licence; SES rate quoted from sendy.co, inference for the sum                                                                                                                                 |
| Dittofeed Cloud Pro        | USD 75, assuming at most 10,000 unique users                                                                      | Verified; send volume does not change the price on the page                                                                                                                                            |
| Mautic Essential (managed) | EUR 247.50 (annual billing)                                                                                       | Verified; fits 50K contacts and 50K emails                                                                                                                                                             |
| Mailchimp Premium          | USD 350 (the 500-contact Essentials and Standard base prices are USD 13 and USD 20; their 10k price not verified) | Verified 2026-10-10: Premium base "includes 10,000 contacts", send cap 15x contacts = 150,000, so 50,000 sends fit                                                                                     |
| Brevo Starter / Standard   | USD 9 + 47 = USD 56 / USD 18 + 79 = USD 97                                                                        | Inference: base plan price plus the 40,001-50,000 email tier (the feed shows base before add-ons and tiers; 50,000 is the tier's upper bound)                                                          |
| MailerLite Comfort / Power | USD 89 / USD 129 monthly (USD 80.10 / USD 116.10 yearly)                                                          | Verified in page data for 10,000 subscribers; Comfort cap 10x = 100,000 sends covers 50,000; mapping Comfort = "growing", Power = "advanced" is inference from matching USD 12 and USD 25 entry prices |
| Kit Free / Creator / Pro   | USD 0 / USD 139 / USD 189 monthly (USD 1,390 / USD 1,890 per year)                                                | Verified in page data: Free is "up to 10,000 subscribers"; paid tiers for the 8,001-10,000 bracket; send caps not verified                                                                             |
| Loops                      | USD 99                                                                                                            | Verified: 5,001-10,000 contacts bracket, unlimited sends                                                                                                                                               |
| Drip                       | not verified                                                                                                      | 403 to every fetch; no primary pricing page read                                                                                                                                                       |
| Listmonk, Postal           | no hosted price exists                                                                                            | Verified: self-host only                                                                                                                                                                               |

### Answers

1. **Which metric do OSS email competitors use?** Mixed, and not contacts-first. Of the four
   with a verified hosted price list: Keila bills emails with unlimited contacts, Plunk bills
   emails with unlimited contacts, Notifuse bills active contacts with unlimited emails over BYO,
   Dittofeed bills unique users. Mautic's managed tiers bundle contacts and emails. Listmonk and
   Postal have no hosted product, and Sendy is a one-time licence over SES. Fetched incumbent text
   shows contacts or subscribers as the main metric for Mailchimp, MailerLite and Kit (Brevo's page
   leads with emails per month; Drip not verified), with sends only as a derived cap.
2. **Existing position or open niche?** "Unlimited contacts, price on sends" is an **existing,
   validated position**, not an open niche: Keila (since its Paddle Classic credit plans, section
   A) and Plunk both ship it today, and Keila's tiers are priced in one metric only. "BYO provider,
   pay only for the software" is also present (Sendy since its design, Notifuse Cloud, Keila's
   "bring your own email server"). What I could **not** find is a product that both prices on sends
   and states a cheaper tier for BYO sending (Keila names BYO but shows one price; Plunk and
   Dittofeed do not mention BYO). That narrower combination is unoccupied in the verified set
   (inference; absence on the fetched pages is not proof of absence in the market).
3. **Table above.** Verified OSS range for this load is roughly USD 49 to EUR 64, with
   Mautic managed an order of magnitude higher and Sendy cheapest in ongoing cost but self-hosted.

### Implication for 1mail (inference, not a decision)

Sends-priced with unlimited contacts is a safe, already-validated metric and matches the
`emails_sent` snapshot in ADR 0009; it differentiates from incumbent contact tiers rather than
from OSS peers, so the differentiator needs to be stated as pricing predictability, not novelty.
A BYO-SES price (software only, no send fee) is the one point nobody verified, so it is the
candidate for a distinct plan; the core needs no change, since both metrics are already
snapshotted.

## Re-verification log (2026-10-10)

Re-fetched from the primary pages today; verdict per claim.

| Claim                                                                                                                                                                                                                                                            | Verdict                                                                                                                                                                                                                                                                                 |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Paddle prohibits "Mass Marketing Products, including but not limited to SMS marketing and Message App marketing" (category 7); email marketing not named                                                                                                         | **confirmed 2026-10-10** ([page](https://paddle.com/help/start/intro-to-paddle/what-am-i-not-allowed-to-sell-on-paddle)). The page also says its lists are not exhaustive. Also lists "Marketing Services and Schemes, including but not limited to pyramid schemes, network marketing" |
| Polar prohibits "Advertising and unsolicited marketing services, including, but not limited to, lead generation," and "bulk SMS and automated outreach"; "Marketing services" and "Marketing & outreach tools" "require a closer review and may not be accepted" | **confirmed 2026-10-10** ([page](https://polar.sh/docs/merchant-of-record/acceptable-use)); no email-specific text. Addition: lead generation is named explicitly in the same prohibited item (item 8), which the summary above omitted                                                 |
| Lemon Squeezy prohibits "Services of any kind (including marketing, design, web development, consulting or other related services)" and "Advertising in newsletters, on websites, or in social media posts"                                                      | **confirmed 2026-10-10** ([page](https://docs.lemonsqueezy.com/help/getting-started/prohibited-products)); also lists "telemarketing" under regulated services; email marketing and lead generation not named                                                                           |
| Stripe Meters have no `max` aggregation                                                                                                                                                                                                                          | **confirmed 2026-10-10**: aggregation formulas are Sum, Count, Last only ([configure](https://docs.stripe.com/billing/subscriptions/usage-based/meters/configure))                                                                                                                      |
| Stripe fees: Billing 0.7%, 2.9% + 30c per card, Managed Payments 3.5% on top of Payments fees, dispute received USD 15, Tax Basic 0.5% no-code or USD 0.50 API                                                                                                   | **confirmed 2026-10-10** ([billing/pricing](https://stripe.com/billing/pricing), [pricing](https://stripe.com/pricing)); basic metering "up to 100M events per month included" also confirmed                                                                                           |
| Stripe restricted list does not name email marketing                                                                                                                                                                                                             | **confirmed 2026-10-10** ([page](https://stripe.com/legal/restricted-businesses)): marketing-related entries are telemarketing, multilevel marketing and "Sales of online traffic or engagement"                                                                                        |
| Managed Payments: business locations, Russia excluded as customer, SaaS tax code, "Professional services, such as consulting, marketing, design..." unsupported                                                                                                  | **confirmed 2026-10-10** ([eligibility](https://docs.stripe.com/payments/managed-payments/eligibility)); business locations: CA, US, 30 European countries (including CH, GB, NO, LI) plus AU, HK, JP, SG                                                                               |
| Keila pricing: EUR 8 / 16 / 32 / 64 / 128 / 256 for 2k / 5k / 15k / 50k / 100k / 250k emails, no free plan, unlimited contacts, bring your own email server                                                                                                      | **confirmed 2026-10-10** ([keila.io/pricing](https://www.keila.io/pricing))                                                                                                                                                                                                             |
| Plunk USD 0.001 per email, no base fee, 1,000 free emails per month, unlimited contacts                                                                                                                                                                          | **confirmed 2026-10-10** ([useplunk.com/pricing](https://www.useplunk.com/pricing))                                                                                                                                                                                                     |
| Notifuse Cloud: Lite USD 19 (16 yearly) 2,500; Starter 49 (39) 10,000; Pro 129 (99) 25,000; Business 319 (249) 50,000 active contacts; unlimited emails over BYO ESP; 14-day trial, card required                                                                | **confirmed 2026-10-10** ([cloud pricing](https://www.notifuse.com/pricing/cloud))                                                                                                                                                                                                      |
| Lago open-code premium gating, Paddle no native metering, Polar and Lemon Squeezy `max` aggregation                                                                                                                                                              | **not re-fetched** in this pass; carried over unchanged                                                                                                                                                                                                                                 |

## Go libraries for Stripe

Fetched 2026-10-10. "Verified" means read in the primary source at the cited pin; "inference" is
my reading and is marked as such. Pins: stripe-go
[`ba0512a50aa6db9145305bfc46b5a9e60ee14378`](https://github.com/stripe/stripe-go/tree/ba0512a50aa6db9145305bfc46b5a9e60ee14378)
(tag v87.0.0), stripe-mock
[`v0.206.0`](https://github.com/stripe/stripe-mock/tree/v0.206.0), river `v0.49.0` (the version in
this repo's `go.mod`; upstream HEAD
[`2f6b27e`](https://github.com/riverqueue/river/tree/2f6b27e69c8e747b6c8b2809fab35c69f401d948)).

### 1. stripe/stripe-go (official), verified

| Topic                 | Finding                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Module and license    | `github.com/stripe/stripe-go/v87` (`go.mod` at the pin), MIT, Go 1.24 minimum ([README](https://github.com/stripe/stripe-go/blob/ba0512a50aa6db9145305bfc46b5a9e60ee14378/README.md)). Not archived; 16 open issues, last push 2026-10-10 (GitHub API).                                                                                                                                                                                                                                                                     |
| Release cadence       | v86.2.0 2026-07-29, v86.3.0 08-11, v86.4.0 08-27, v86.4.1 09-01, v86.4.2 09-09, **v87.0.0 2026-10-01** (latest stable), plus weekly alpha/beta previews ([releases](https://github.com/stripe/stripe-go/releases)). Roughly one minor every 2 weeks and a **major roughly every 2 to 3 months**; majors contain breaking changes.                                                                                                                                                                                           |
| API-version pinning   | The SDK pins one API version per release: `APIVersion = "2026-09-30.endive"` in `api_version.go`. Upgrading the module upgrades the API version of every request. v87 changed decimal fields from `float64` to `shopspring/decimal` and dropped Go 1.22/1.23 ([CHANGELOG](https://github.com/stripe/stripe-go/blob/ba0512a50aa6db9145305bfc46b5a9e60ee14378/CHANGELOG.md)).                                                                                                                                                 |
| Client                | `stripe.NewClient(key, ...)` with per-resource services (`sc.V1Customers`, `sc.V1PaymentIntents`, ...); the older per-package `*/client.go` API is marked Deprecated in the source. Backends are injectable (`stripe.WithBackends`, `BackendConfig`).                                                                                                                                                                                                                                                                       |
| Retries and timeouts  | Automatic retries on connection errors, timeouts and some statuses such as 409; default `DefaultMaxNetworkRetries = 2`, configurable via `BackendConfig.MaxNetworkRetries`; default HTTP timeout 80 s (`defaultHTTPTimeout`).                                                                                                                                                                                                                                                                                               |
| Idempotency           | An `Idempotency-Key` header is **added automatically** to requests (`NewIdempotencyKey()` when none is supplied), so SDK retries are safe; a caller key goes in `Params.IdempotencyKey` (max 255 chars, enforced in `NewRequest`).                                                                                                                                                                                                                                                                                          |
| Billing Meters        | Meter CRUD: `billing_meter.go`. **v1** `BillingMeterEventCreateParams` has `EventName`, `Identifier` (dedupe key; "Stripe enforces uniqueness within a rolling period of at least 24 hours"), `Payload`, `Timestamp`, `billing_metereventadjustment`, `billing_metereventsummary`. **v2** `v2/billing/metereventsession` and `v2/billing/metereventstream` exist, but the official `example/v2/meter_event_stream` drives them through `rawrequest` plus a hand-managed session token (15 min expiry, per the Stripe docs). |
| Webhooks              | `webhook.ConstructEvent(payload, header, secret)` (default tolerance 300 s), `ConstructEventWithOptions` (`IgnoreTolerance`, `IgnoreAPIVersionMismatch`), `ValidatePayload`; `stripe.GenerateTestSignedPayload` for tests. By default `ConstructEvent` **rejects events whose `api_version` differs from the SDK's `stripe.APIVersion`** (doc comment in `webhook/client.go`).                                                                                                                                              |
| Thin events           | `Client.ParseEventNotification` / `EventNotificationContainer`, `V2CoreEventNotification.FetchEvent`, `UnknownEventNotification` (`event_notification.go`, `stripe_client.go`); the Stripe webhooks page calls thin the recommended payload for new integrations.                                                                                                                                                                                                                                                           |
| Tax, Portal, Checkout | Present as generated resources: `checkout_session.go`, `billingportal_session.go` / `billingportal_configuration.go`, `tax` package (the tax settings/registration/calculation resources), `testhelpers/testclock` (`New`, `Advance`, `Del`).                                                                                                                                                                                                                                                                               |
| Testing               | README documents running against stripe-mock and a GoMock `Backend` mock.                                                                                                                                                                                                                                                                                                                                                                                                                                                   |

Caveats verified in the Stripe docs ([recording usage](https://docs.stripe.com/billing/subscriptions/usage-based/recording-usage-api)):

- v1 `meter_events` is limited to **1000 calls/s per account** and one concurrent call per customer per
  meter; events are validated **asynchronously**, errors surface only as thin events
  `v1.billing.meter.error_report_triggered` / `v1.billing.meter.no_meter_found`.
- The v2 stream is 10,000 events/s, **live mode only**, with 15 minute session tokens.
- Timestamp must be within the past 35 days and at most 5 minutes in the future; late events may be
  accepted but not invoiced after finalization.
- Dimension cardinality limits: 10,000 combinations per meter per hour, 100 per customer per meter.
- The `Identifier` dedupe window is the only dedupe Stripe offers for usage; there is no batch endpoint on v1.

Inference: for this load (sends and contacts snapshots, ADR 0009) v1 `BillingMeterEvent` is enough;
the v2 stream needs hand-rolled session refresh around `rawrequest` and is not worth it before ~1000
events/s, and pre-aggregation per workspace and period reduces calls anyway.

### 2. Complementary tools

| Tool                                                                                                             | State at pin                                                                                                                                                                                                             | Use                                                                                                                                                                                                                      |
| ---------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [stripe/stripe-mock](https://github.com/stripe/stripe-mock) v0.206.0, MIT, 6 open issues, released 2026-10-01    | Verified from its README: validates params against the OpenAPI spec, returns **hardcoded fixtures**, is **stateless**, locked to the latest API version, cannot simulate specific errors.                                | Only for contract/serialization checks of our client calls (CI). It cannot test subscription state or usage aggregation. Inference: do not build behaviour tests on it.                                                  |
| [stripe/stripe-cli](https://github.com/stripe/stripe-cli) v1.53.1 (2026-10-07), Apache-2.0, 232 open issues      | `stripe listen --forward-to` (snapshot) and `--forward-thin-to ... --thin-events "*"`, `stripe trigger <event>`, `stripe events resend <id> --webhook-endpoint=...` (30 days) per the Stripe webhooks docs.              | Dev-only: add as a `mise` tool and a daemon forwarding to the plane's webhook route. Sandbox and Test Clocks are the only way to get realistic subscription lifecycle.                                                   |
| Stripe Test Clocks (`testhelpers/testclock` in stripe-go)                                                        | Verified present in the SDK.                                                                                                                                                                                             | Needed for renewal, trial end and dunning scenarios against a real sandbox (nightly, not per-PR).                                                                                                                        |
| [riverqueue/river](https://github.com/riverqueue/river) (already in repo, v0.49.0, MPL-2.0, released 2026-10-05) | `InsertOpts.UniqueOpts{ByArgs, ByPeriod, ByQueue, ByState}` and `JobInsertResult.UniqueSkippedAsDuplicate`; transactional insert (`InsertTx`).                                                                           | Inbox dispatch: verify, persist the event id, and enqueue a job in the same transaction. Inference: river `ByArgs` uniqueness is best-effort dedupe on top of, not a replacement for, a unique-constrained event-id row. |
| [ThreeDotsLabs/watermill](https://github.com/ThreeDotsLabs/watermill) v1.5.3 (in repo), MIT                      | Delivers at-least-once pub/sub; it has no Stripe-specific idempotency middleware that I verified.                                                                                                                        | Inference: not the right fit for the plane's webhook inbox; river gives retries, backoff and a table the operator can inspect, and the plane is a separate service anyway.                                               |
| Idempotent-webhook or usage-batching Go library                                                                  | **None found.** GitHub code search for `stripe-go` + `ConstructEvent` returns only application code and vendored copies of the SDK; no maintained library for inbox dedupe, meter batching or flush scheduling surfaced. | Must be written (see 4).                                                                                                                                                                                                 |

Fallback-provider SDKs (one line each, verified via GitHub API 2026-10-10):

- **getlago/lago-go-client** v1.55.0 (2026-10-07), MIT, 2 open issues, active; the Lago server's
  open-code premium gating is carried over from the earlier section, not re-verified.
- **openmeterio/openmeter** (Go SDK lives in the server repo) v1.0.0-beta.236 (2026-10-09), Apache-2.0,
  63 open issues, still beta-versioned.
- **PaddleHQ/paddle-go-sdk** v5.2.0 (2026-03-30), Apache-2.0, 28 open issues; last release is six
  months old, though the repo was pushed on 2026-10-02.

### 3. How Go projects wire Stripe (read in code)

Only a handful of Go repos using meters turned up, and almost all are small; cited only where I read the
code. Evidence is weak, treat as patterns not endorsements.

- **OpenMeter**: Stripe app webhook
  ([`webhook.go`](https://github.com/openmeterio/openmeter/blob/e00163c4b8a0726198c95a9ccb158d38ea8c05b8/openmeter/app/stripe/httpdriver/webhook.go),
  Apache-2.0) verifies with `webhook.ConstructEventWithTolerance(..., 10000h)` using a per-app secret
  and dispatches on event type in a large `switch`. I saw no event-id dedupe in that file, so state
  changes there rely on being idempotent. Its `go.mod` is on `stripe-go/v80` at that commit, so even a
  large project lags by several majors. Verified.
- **Livepeer-FrameWorks/monorepo** (Unlicense, 6 stars):
  [`meter_flusher.go`](https://github.com/Livepeer-FrameWorks/monorepo/blob/2e4b205bd506f6e24cd9a80765545c17f5867de4/api_billing/internal/stripe/meter_flusher.go)
  uses a transactional **outbox**: usage rows are written in the invoice transaction, a flusher pushes
  `BillingMeterEventParams` with `Identifier = outbox row id`, marks `sent_at`, counts attempts with a
  cap (6) and leaves failures for ops. If marking fails, the retry collapses on Stripe's `Identifier`
  dedupe. This is the same shape as ADR 0009's snapshot approach. Verified.
- **onmetal-dev/metal** (AGPL-3.0, 4 stars,
  [`serverbillinghourly.go`](https://github.com/onmetal-dev/metal/blob/aa36cb256d7f2c7760984c920336082587b935cd/lib/background/serverbillinghourly/serverbillinghourly.go)):
  a periodic job sets `Identifier` to a deterministic id per customer and hour. Verified.
- **kamilpajak/heisenberg** (`ee/billing/webhook.go`, Apache-2.0, 0 stars): verifier behind an
  interface, `switch` on `checkout.session.completed`, `customer.subscription.created|updated|deleted`,
  `invoice.payment_succeeded`. A pattern for testability only.

Common pattern: deterministic `Identifier` for meter events; subscription state synced from
`customer.subscription.*` plus `invoice.*` events. I found **no** repo with a verified Stripe
event-id inbox on river; do not cite one.

### 4. Recommendation

**Import** (all MIT/Apache-2.0, compatible with an AGPL core or closed plane):

- `github.com/stripe/stripe-go/v87` pinned exactly (v87.0.0 or the latest v87.x), via `go.mod` and
  Renovate/Dependabot with majors reviewed by hand. Nothing else for Stripe calls: no wrappers
  around it, no second SDK.
- `github.com/riverqueue/river` (already v0.49.0) for webhook processing and the usage flusher.
- Dev tooling only: `stripe-cli` v1.53.x (mise tool), `stripe-mock` v0.206.x (CI).
- samber/do for wiring a `*stripe.Client` singleton, same as the core.

**Minimal wiring (inference built on the verified facts above):**

1. Webhook route `POST /stripe/webhook` in the plane. Read the **raw body**, call
   `webhook.ConstructEvent`, return 2xx quickly (Stripe docs: do not do complex work before 2xx;
   live retries run up to 3 days with exponential backoff).
2. In one DB transaction insert `stripe_events(event_id PRIMARY KEY, type, payload, received_at)` with
   `ON CONFLICT DO NOTHING`, and if the row is new, `river.InsertTx` a `ProcessStripeEvent{event_id}`
   job (`UniqueOpts{ByArgs: true}` as a second guard). Duplicates return 200 without work, per Stripe's
   "log processed event ids" guidance.
3. The job loads the stored payload and applies it idempotently. Because ordering is **not
   guaranteed** (Stripe docs), never apply the snapshot blindly: on `customer.subscription.*` and
   `invoice.*`, re-fetch the object with `sc.V1Subscriptions.Retrieve` (or the thin event's related
   object) and upsert the plane's entitlement state from that, comparing on the object's own state
   rather than `event.created`.
4. Usage: the core's snapshot (ADR 0009) goes to a plane outbox table; a river periodic job posts
   `BillingMeterEventCreateParams` with `Identifier = "<workspace>:<meter>:<period>"` (deterministic,
   so replays collapse) and a bounded `Timestamp` inside the 35-day window; per-customer calls are
   serialized because Stripe allows one concurrent call per customer per meter.
5. Version discipline: pin the webhook endpoint's API version to the SDK's `stripe.APIVersion`
   (`2026-09-30.endive` for v87); otherwise `ConstructEvent` fails on version mismatch. Either upgrade
   the module and the endpoint together, or use `ConstructEventWithOptions{IgnoreAPIVersionMismatch: true}`
   deliberately and parse only the fields you own. Event payload shape follows the endpoint's version
   (Stripe webhooks docs).
6. Register one endpoint for snapshot events (subscriptions, invoices, checkout) and, if you want
   meter error alerts, a **separate** thin endpoint for `v1.billing.meter.error_report_triggered`
   (Stripe requires separate endpoints for thin and snapshot); parse it with
   `Client.ParseEventNotification`.

**Not covered by any library, so we write it** (small, domain-specific):

- The `stripe_events` inbox table and the dedupe-then-enqueue transaction (no Go library found).
- The usage outbox, deterministic identifier scheme, batching/serialisation per customer, and reconciliation
  of what Stripe accepted (v1 validation is async, so a reconcile job reading `v1.billing.meter.*` thin
  events and `BillingMeterEventSummary` is needed).
- Subscription-to-entitlement mapping (plan, limits, suspension), dunning policy, and the contract
  back to the core.
- v2 meter-event-session refresh around `rawrequest` (only if we ever need the stream).
- Test scenarios with realistic state: stripe-mock is stateless, so lifecycle tests need a sandbox
  with Test Clocks, run nightly.
- Tax registration/config, Customer Portal configuration and Checkout success handling are API calls
  the SDK already types; we only write the orchestration.
