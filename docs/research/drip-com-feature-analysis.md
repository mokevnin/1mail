# Drip.com Product Feature Breakdown (for competitive gap analysis)

Researched June 2026 via multi-source web search + primary Drip pages (drip.com, help.drip.com, developer.drip.com) and current third-party pricing/review trackers. drip.com and help.drip.com return HTTP 403 to automated fetchers; most help-center claims are sourced from search-engine snippets that quote those same official pages plus the directly-fetchable developer API reference. Uncertainties are flagged inline.

Drip positions itself as an **Ecommerce CRM (ECRM)** — "Ecommerce Email Marketing & Automation," tagline "Serious ecommerce email a small team can run." Strongest in the Shopify/WooCommerce ecosystem.

---

## 1. Email marketing

- **One-off / broadcast campaigns** are called **"Single Email Campaign"**; recurring multi-step sequences are an **"Email Series Campaign."** Recipients are a saved or one-time Segment.
- **Three email builders:**
  - **Visual Email Builder** — template-based drag-and-drop WYSIWYG (managed under Campaigns > Email Templates).
  - **Text Builder** — HTML editing with a Plain Text fallback view; requires the Liquid `{{ email.html }}` tag to save.
  - **HTML Builder** — paste/edit custom HTML, import external templates.
- **Templates** — ecommerce-focused template library (no authoritative count confirmed — flagged).
- **Design / content blocks** — preview in desktop/mobile/tablet; named blocks include a **Product** block and dynamic e-commerce blocks (see §8).

## 2. Marketing automation

- **Workflows** = the visual automation builder, composed of Triggers, Actions, Decisions, Goals, Delays, and Exits.
- **Triggers** (forward-facing only; provider + event model): signup, tag applied, event performed, link clicked, **"Entered a Segment"** (workflow-only), **Date-Based Trigger** (supports recurring date custom fields for birthdays/anniversaries). A **Trigger Filter** narrows entry using the same criteria as segments.
- **Actions** (verbatim list captured): Send an email to a person; Apply/Remove a tag; Set a custom field; **Add a person to a Workflow** (send-to-workflow); Subscribe/Remove from an Email Series; **Send an HTTP post** (the webhook action); record conversions/custom events; plus integration actions (Salesforce, Pipedrive, Facebook Custom Audiences, Demio, etc.).
- **Delays / waits** are a separate node (not in the Actions list).
- **Decisions** — Yes/No conditional split paths that can re-converge.
- **Goals / goal exits** — milestones that "pull people down" the workflow; one-directional; can be date-based, used as entry points, placed side-by-side, or mid-workflow (an "Exitless Path").
- **Exits** — every workflow has a non-removable bottom Exit; re-entry possible on re-trigger.
- **Rules** — older, simpler if-then automation (two-step, optional delay, instantaneous) that coexists with Workflows for lightweight tag/field reactions.
- **Templates / playbooks** — pre-built playbooks (welcome series, abandoned cart, post-purchase, win-back, browse abandonment).

## 3. Segmentation

- **Dynamic segments** — "a filtered list of people that have something in common"; auto-include/exclude in real time as people match/stop matching (e.g., a "one-time buyers" segment drops a person on their second purchase).
- **Filter criteria** — purchase history, website activity, email engagement, location, Tags, Custom Fields, custom events. Multi-criteria with AND/OR (exact UI operator labels unconfirmed — flagged).
- **Tags** — binary attributes ("has it or doesn't"), applied manually or automatically by the automation engine on actions (page visit, click, purchase, custom event). Drip's guidance: Tags for fixed attributes, Segments for behavior-driven groups that "refresh themselves."
- **Saved Segments** live on the People page, sortable, and are used as the audience for sends and as automation entry conditions.

## 4. Contacts / subscribers

- **People-based identity model** ("People" tab). A person is keyed by `email` / Drip `id`, and supports a `user_id` from your own DB. Drip appends a subscriber ID to every email link so click-throughs resolve identity on-site.
- **Custom fields** — effectively unlimited per person; usable in segmentation and Liquid personalization.
- **Person Profile + activity timeline** — "All Activity" timeline showing event history (activation, tags, clicks, form submits, custom-field updates, custom events, e-commerce events); a "Recent Activity" report surfaces recently active people.
- **Lead scoring — LEGACY.** Labeled **"Lead Scoring (Legacy Feature)"** with a published "Rule Workaround" article, indicating it is deprecated/de-emphasized for new accounts (no explicit dated sunset found — flagged). Legacy mechanics: new people start as "prospects" at score 30; reaching the lead threshold (default 65, range 5–150) marks them a "lead"; engagement adds up to 5 pts/open or click; inactivity decays the score.

## 5. Behavioral tracking

- **JavaScript snippet** — a unique per-account tracking script (Settings > Account > Site Setup), pasted before `</body>`, deployable via Google Tag Manager. Enables visitor identification and behavior tracking; prerequisite for the Shopper Activity API. JS methods: `Drip.identify({ email })`, `Drip.track('event')` (legacy `_dcq` queue also exists).
- **Events** — Predefined Events (tag applied, link clicked, "Submitted a form," custom field updated) vs Custom Events (via JS `track` or `POST /v2/:account_id/events`; Batch API up to 1,000 events/request). Events drive Workflows and Rules.
- **E-commerce events (Shopper Activity API)** — three streams: **Product** ("Viewed a Product"), **Cart** ("Created/Updated a cart"), **Order** ("Placed an Order"), each with batch endpoints; emitted automatically by native store integrations.
- **Cart abandonment** — built-in pattern (cart event without a matching purchase); shareable cart-abandonment workflow; follow-up emails render cart items via Liquid.
- **Browse abandonment** — dedicated workflow triggered when an identified subscriber views a product page ("Viewed a Product") without adding to cart or purchasing; pre-built templates exist.

## 6. Forms & popups (onsite campaigns)

Two distinct current products (legacy "Hosted Forms" exist only for pre-Nov-2022 accounts):
- **Onsite Pop-ups ("Drip Onsite")** — built on acquired **Sleeknote** tech. Formats: pop-ups, slide-ins, sticky bars, sidebars, embedded. Drag-and-drop builder with separate desktop/mobile editing; one-line install. Advanced experiences: **multistep forms, quizzes, gamification (Spin-to-Win), surveys, countdown timers, free-shipping bars, upsells, product recommendations** — used to collect zero-party data. Triggers: time delay, scroll %, exit-intent (desktop only), on-click, teaser. Targeting by URL, geolocation, session activity, cart status, segment, new-vs-returning. **No session/impression limits; unlimited popups on every plan including trial.**
- **Embedded Forms** — static, always-visible forms added to site code; styled via your CSS; reCAPTCHA v3; GDPR consent fields.
- **A/B testing of popups specifically is unconfirmed** (Drip lists "Smart A/B testing" as a platform feature but documents it around emails/campaigns — flagged).

## 7. E-commerce integrations

- **Headline one-click store connectors:** **Shopify, BigCommerce, WooCommerce**, plus custom stores via the **Shopper Activity API**. **Magento** is supported (with caveats: Adobe Commerce 2.4.4+ disables the stand-alone bearer token Drip's integration needs; Magento Product block pulls only from past orders).
- **Catalog size:** Drip cites "90+ integrations" on the pricing FAQ (and "200+"/"50+ ecommerce" elsewhere — figures vary by page).
- **Shopify** — marketed as the easiest integration; syncs customer profiles, full order history, product catalog, and cart/checkout/purchase/product-view events; Product block auto-populates within ~24h. Drip cites $231.7M Shopify revenue attributed (2025).
- **WooCommerce** — native (official WooCommerce Marketplace extension); tracks products purchased, conversions, LTV, and abandonment events.
- **BigCommerce** — install the "Drip Email Marketing" app to sync store/customer data into dynamic segments.
- **Revenue attribution / data layer** — reports Order Activity and attributes revenue per customer, Email Series, Workflow, and Single Email Campaign. Homepage claims $1.5B attributed revenue across 6,000+ brands.

## 8. Personalization & dynamic content

- **Liquid templating** — Drip's dynamic templating language: Objects (`{{ subscriber.tags }}`), Tags (logic/conditionals), Filters (`|` formatting). The standalone Liquid manual URL appears retired; current docs live in the help center (flagged).
- **Merge tags / personalization** — Liquid pulls custom-field values and renders tag-based conditional copy so one email serves multiple segments.
- **Dynamic content blocks** (Visual Builder) — named e-commerce blocks: **Cart Abandonment** (auto-populates cart items), **Top Selling Products** (store-data-driven), **Discounts**.
- **Product Content Block** — manually insert a store product (by title/variant/SKU) for Shopify/Magento/BigCommerce/WooCommerce/custom. IMPORTANT NUANCE: once inserted "it will not dynamically update" — it is a curated, static product block, NOT live 1:1 recommendations. Fully-automated, no-curation "product recommendations" (claimed by some third parties) are **unconfirmed at the product level** (flagged); the Onsite suite does offer "product recommendations" widgets in popups/sidebars.
- **Custom Dynamic Content** — deep personalization calling an external endpoint, exposed via a `my.<name>` Liquid shortcode (e.g., weather, cross-sell). Requires Drip's engineering team to set up case-by-case (gated).
- **Coupons** — a **Discounts content block** exists to surface/insert discounts; **native unique single-use code generation is unconfirmed** — appears to rely on a third-party integration ("Coupon Carrier") (flagged).

## 9. Deliverability

- **Sending infrastructure** — Drip sends from its own (shared) infrastructure; un-authenticated mail shows a "via Drip.com" byline.
- **Custom Sending Domains (CSD)** — Drip's term for domain authentication; publish provided CNAME (or TXT, via support) records to enable **SPF and DKIM** and remove the "via Drip.com" tag. Documented registrar setups (GoDaddy, Cloudflare, Namecheap, etc.).
- **DMARC** — Drip instructs adding a DMARC record (post-Feb 2024 Gmail/Yahoo bulk-sender rules) for full compliance.
- **Dedicated IPs — NOT confirmed.** No Drip doc found offering a dedicated-IP product; guidance centers on domain reputation, not IP selection. Treat as likely not self-serve (flagged — verify with Drip sales).
- **Reputation / sunsetting** — recommended as practices you build yourself ("send to fewer people," prune unengaged after 90–120 days, use list-cleaning tools), not one-click automated engagement management.

## 10. Analytics & reporting

- **Campaign metrics** — open rate, CTR, bounces, unsubscribes (Campaign Dashboard / "Email Metrics").
- **Revenue reporting ("Insights")** — dashboards for e-commerce revenue attribution; opens/clicks/unsub alongside revenue and **revenue per subscriber** for every email and workflow; real-time revenue chart. Requires an e-commerce integration (gated by integration, not plan tier).
- **Attribution** — default model: revenue counted if the customer clicked an email within 5 days of purchase, OR was "seen on-site" within a day of receiving an email and purchased within 5 days. Attribution window configurable (default 5 days); an "Email Only Attribution" toggle restricts to direct click-throughs.
- **Workflow analytics** — revenue/engagement surfaced per workflow; split tests expose Revenue, Order Conversion, Site Visits, Email engagement.

## 11. A/B testing

- **Single email "Split Test"** — up to **four** subject-line or content variations; set a test pool from a Segment, a duration, then Drip sends the winner to the rest. Per-variation opens, clicks, and revenue shown.
- **Workflow Split Tests** — test whole sequences (e.g., email+discount+Facebook audience vs email-only) on four measurements (Revenue, Order Conversion, Site Visits, Email engagement).
- **Winner selection is MANUAL** (Results Modal; Drip recommends ≥1,000 people enter before declaring). No documented fully-automatic statistical winner.

## 12. SMS marketing

- **Current status: legacy / gated, NOT generally available to new users.** Legacy SMS is US-only, requires the "Support All People" feature, and you must email support to enable; closed to new signups.
- **A new SMS product is in development** — Feb/Mar 2026 release notes introduce an **"SMS Consent Element"** in Onsite forms (TCPA/CTIA carrier-approved opt-in) that "sets you up for Drip's upcoming SMS product."
- No explicit public discontinuation date for the original SMS product was found (flagged). SMS pricing is described as tiered, starting ~$39/mo.
- **SMS via integrations** is the practical path today (marketplace lists Bulk SMS, Call Loop, etc.).

## 13. APIs & integrations

- **REST API** (`api.getdrip.com/v2/`, some v3) — HTTP Basic Auth (token) or OAuth 2.0 Bearer. Exposes Subscribers (+ batch up to 1,000), Campaigns (Email Series + Single Email/Broadcasts), Events, Conversions/Goals, Tags, Custom Fields, Shopper Activity (carts/orders/products + batch), Workflows (+ triggers), Forms, Webhooks, Accounts, Users. Rate limits ~3,600 individual / ~50 batch requests per hour. Official Node.js wrapper.
- **JavaScript / tracking API** — identify, custom events, conversions, product views, client-side campaign subscribe/unsubscribe, form events, query-string tagging.
- **Webhooks** — outgoing only (no inbound receiver); rich subscriber-lifecycle, data, engagement, and behavior/scoring event types (e.g., `subscriber.applied_tag`, `subscriber.opened_email`, `subscriber.performed_custom_event`, `subscriber.updated_lead_score`).
- **Zapier** — official app (triggers + actions) bridging ~8,000 apps; plus third-party no-code layers (Make, Integrately).
- **Native catalog** — Shopify, WooCommerce, Magento, BigCommerce, **Facebook/Meta Custom Audiences** (native, usable as a Workflow/Rule/Bulk action to add/remove people and seed lookalikes), LoyaltyLion, Smile.io, Recharge, Judge.me, Drift, Sleeknote, etc.

---

## Account / pricing model (context for gap analysis)

- **Single-plan, contact-based pricing** — **CONFIRMED CURRENT (June 2026)** across multiple independent pricing trackers (G2, CheckThat.ai, ThatMarketingBuddy): one tier structure scaling by contacts + send volume, **no feature gating** ("everything from day one"). Entry **$39/mo for up to 2,500 people**, unlimited sends, up to 50 workflows, dynamic segments, onsite campaigns, unlimited sub-accounts, open API.
  - NOTE: one research thread inferred a "Free/Standard/Pro" tier split from release notes; this was NOT corroborated by current pricing sources and is treated as an error. The single-plan model stands. (drip.com/pricing 403s automated fetch, so exact live page not directly read; multiple 2026 trackers agree.)
- **Support gating** — live chat on $99/mo+ plans; email support on all paid plans.
- **Free trial** — 14 days, no card; limited (≈100 sends / 2 campaigns per day; no unlimited sends during trial).
- **Multi-account** — unlimited sub-accounts under one pooled (per-person) bill; only the owner creates them. Light agency/multi-brand model.
- **Roles** — Account Owner, Account Admin, Account Contributor; per-user login + MFA; admins/contributors scopeable to specific sub-accounts. No explicit per-seat cap/charge found (per-contact billing).

## Key uncertainty flags
1. Lead scoring = legacy/deprecated (no dated sunset found).
2. Dedicated IPs — not documented; likely not self-serve.
3. Native unique coupon-code generation — unconfirmed (likely via Coupon Carrier integration).
4. Fully-automated 1:1 product recommendations in email — unconfirmed; Product block is manual/static.
5. A/B testing of onsite popups specifically — unconfirmed.
6. SMS — legacy US-only/gated + new product in progress; no dated discontinuation found.
7. Tier model — single-plan confirmed by trackers; live pricing page not directly fetched.
