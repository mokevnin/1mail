# Billing provider options for the SaaS control-plane (ADR 0009)

Researched October 2026 from primary sources: source code cloned at each repo's default branch
(shallow clone, commit SHAs below) and the providers' own docs, pricing pages and policy pages
(URLs inline). Permalinks use the commit SHA so line numbers stay valid. Every claim is either
**verified** (URL or permalink given) or explicitly marked **inference** / **not verified**.

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
  above, and both exclude or may exclude Russia-adjacent flows (Polar verified; Lemon Squeezy not
  verified).
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
- **Webhooks and dunning.** Stripe signs webhooks and the docs cover retries via event
  destinations. I did not verify Stripe's Smart Retries schedule or the invoice finalization grace
  period page in this pass (the latter is linked from recording-usage-api:
  [configure-grace-period](https://docs.stripe.com/billing/subscriptions/usage-based/configure-grace-period)).
- **Business location (verified).** Supported countries include the EU, UK, US, Switzerland,
  Singapore, Hong Kong, UAE, Brazil and others; **Russia is not on the list**
  ([stripe.com/global](https://stripe.com/global)).

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
- **Not verified: whether Managed Payments supports metered (usage-based) prices.** The three
  pages above do not mention meters either way, and the exclusion of invoice items and one-off
  invoices means a plane that adds usage lines by hand would not work. This must be confirmed
  with Stripe before Managed Payments is considered for usage billing.
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
- **Not verified:** supported payout countries, dunning schedule, webhook signature details.

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
  ([supported countries](https://polar.sh/docs/merchant-of-record/supported-countries)). Go SDK
  not verified.

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
- **Orb:** not verified from primary sources in this pass.
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
   Payments paths opens, and Paddle's and Lemon Squeezy's payout-country rules were not
   verified. Do not choose before this is answered.
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
6. **Stripe Managed Payments plus metered prices:** needs a Stripe confirmation.
7. **Stripe Smart Retries schedule and the invoice finalization grace period:** not read in
   this pass; read before sizing the Billing hold's dunning grace period.
