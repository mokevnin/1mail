# Keila Feature Breakdown (for competitive gap analysis)

Researched July 2026 via web search + primary Keila pages (keila.io, keila.io/docs,
keila.io/roadmap, github.com/pentacent/keila). Keila's public docs are thin on
implementation detail, so several model claims are inferred from the docs' *usage* framing
and flagged inline. Where a claim below says "not in docs" it means the capability may
exist but is undocumented — not that it is known absent.

Keila positions itself as **"a 100% Open Source newsletter tool, made in Germany and hosted
in the EU"** — a self-hostable **Mailchimp/Brevo alternative** with a privacy/EU-data-control
angle. Pentacent (the vendor) runs a managed cloud at app.keila.io alongside the AGPL binary.

**At a glance:** mature *newsletter* product (broadcasts + forms + segments + transactional),
but **no automations/drip yet** (roadmap: *Planned*) and **no on-site behavioral tracking**.
That is precisely the layer 1mail is built around — so on our core thesis (event-driven
automation + `/collect`) Keila is not a competitor; on newsletter fundamentals it is ahead
of where we are and worth mining for UX and query-model ideas.

---

## 1. Positioning & stack

- **Language/stack:** Elixir/Phoenix, server-rendered HTML + SCSS + a little JS + Liquid
  templates. A classic Phoenix monolith — **no SPA + generated-API-client split** like our
  Go + React/TypeSpec pipeline. Frontend and backend are one deployable.
- **License:** AGPL-3.0 (logo + `/extra` dir excluded) — the **same open-core shape as ours**
  (AGPL core, carve-outs for the vendor's proprietary bits).
- **Maturity:** ~2.1k GitHub stars, 32+ contributors, 55 releases, latest v0.30.2 (June 2026).
  Steady, real, single-vendor-led project.
- **Deploy:** official `pentacent/keila` Docker image + sample docker-compose; one-click
  Railway template. Self-host or managed cloud.

## 2. Campaigns (broadcasts)

- Keila's send unit is a **campaign** = our **Broadcast**. Scheduling + delivery actions
  via UI and API.
- **Editors — several parallel options:** visual **Block Editor**, **Markdown WYSIWYG**,
  raw **MJML**, and plain text. A **Visual MJML Editor** is *In Progress* on the roadmap.
  - Contrast with 1mail's deliberate **MJML-only** stance ("simple and powerful": one
    powerful primitive, not parallel toggles — see the `simple-and-powerful` memory). Keila
    took the opposite bet (many editors). Not something to copy; a validation that MJML is a
    first-class citizen for OSS newsletter tools.
- **Public archive links** (shipped Jan 2026) — publish a sent campaign as a public web page
  to showcase/share newsletter content. **1mail has no equivalent.** → candidate feature (§9).

## 3. Contacts & custom data

- Contact carries arbitrary **JSON custom data, incl. nested fields**; segment language reaches
  into it by dot-notation (`data.age`, `data.tags.foo`).
  - This is the **schemaless-trait model 1mail explicitly rejected** (CONTEXT anti-vocabulary:
    every non-core attribute is a *typed, named, governed* **Custom field**, auto-created on
    first sight). Keila's JSON bag is more flexible but gives the segment builder no governed
    field list to offer — you must know your own JSON shape. Our typed-field decision looks
    better for a segment UI; Keila's is easier to ingest arbitrary payloads. Worth keeping our
    line but noting the tradeoff.
- **Import:** manual entry + CSV bulk import, with a "Replace duplicates" toggle (email is the
  dedup key). No documented multi-key identity (email/phone/subject_id) or visitor stitching —
  Keila is **email-keyed**, with **no identity-resolution / anonymous-visitor model** (our
  Contact + Visitor + Identify spine has no counterpart here).
- Contact status / subscription state: **not documented**; given the feature set it is almost
  certainly a stored subscribe/unsubscribe flag per contact — i.e. the classic model 1mail
  deliberately replaced with *derived* Send-eligibility. (Flagged — not verified.)

## 4. Segments — the most interesting part to compare

Keila's segments use a **JSON query syntax "inspired by MongoDB Query Documents"**, evaluated
live (dynamic membership). Documented operators:

- **Logical:** `$and`, `$or`, `$not`
- **Comparison:** `$lt`, `$lte`, `$gt`, `$gte`
- **Pattern:** `$like` (`%` wildcard) — e.g. `{ "email": { "$like": "%@keila.io" } }`
- **State:** `$empty` (null / empty string / empty array/object / unset)
- **Nested access:** dot-notation into JSON custom data (`data.tags`)
- **Behavioral (`messages` object):** filter by campaign interaction —
  `{ "messages": { "campaign_id": "…", "opened_at": { "$empty": false } } }` with sub-fields
  `opened_at`, `clicked_at`, `bounced_at`, `complaint_received_at`, `unsubscribed_at`.

**Comparison to 1mail's Segment** (combinator + nested rules over contact fields, custom fields,
*and Events*, evaluated live, never materialized):

- **We are architecturally ahead on behavior.** Keila's behavior is a bolted-on `messages`
  sub-object with a fixed set of email-engagement timestamps — it can only ask about *email*
  interactions, one campaign at a time. 1mail folds `email.sent/opened/clicked` into the **same
  Event stream as customer-tracked actions** (CONTEXT: "engagement is segmentable through the
  very same machinery … no parallel event stream"). We can segment on arbitrary events; Keila
  can only segment on its own email metrics.
- **But their operator set is a good concrete checklist** for our rule query: `$like` (wildcard),
  `$empty` (null/unset/empty — a genuinely useful state test), and the `$lt…$gte` family are
  exactly the leaf predicates a segment builder needs. Worth confirming our rule-query DSL
  covers all of these before we ship the builder. → §9.
- Their "segment by message interaction — received/opened/clicked a specific campaign" (shipped
  Nov 2025) is table-stakes we get for free via Events.

## 5. Forms (sign-up)

- **Form Builder** — real, shipped product: design sign-up forms, add custom checkboxes / text
  fields / dropdowns, **captcha**, and **double opt-in verification**.
- Maps onto 1mail's **ADR 0013 (double opt-in / confirmed subscription)** and the deferred
  "on-site popup/form" surface (CONTEXT "Channels & future surfaces"). Keila has *built* the
  capture surface + double-opt-in loop that 1mail has only *modelled the consent side* of
  (Confirmation as event + gate row). Their form builder is a reference for when we build ours.
- **Embedded form markup generation** is on their roadmap (*Planned*) — i.e. even Keila doesn't
  yet emit copy-paste embed snippets. Minor.

## 6. Sending & deliverability

- **Providers:** AWS SES, Sendgrid, Mailgun, Postmark, SMTP, + local inbox (dev). Same
  provider surface as our **Integration** concept.
- **"Send with Keila"** (Nov 2025) — onboarding wizard: automated sender setup, **DNS checks**
  (SPF/DKIM/DMARC), and **sending from shared domains** (e.g. Proton, Gmail addresses).
  - The **DNS-check + guided setup UX** is directly relevant to our **ADR 0010 (native DKIM,
    live verification)** — Keila surfaces the same SPF/DKIM/DMARC checks as a friendly wizard.
    Good UX reference for our sending-domain verification screen. → §9.
  - "Shared domains" (send from a pooled Keila domain) is a **cloud/onboarding** convenience;
    less relevant to our native-signing-per-workspace-domain model, but the *low-friction first
    send* it enables is worth noting for SaaS onboarding.
- **Fully-European send stack** is *In Progress* (their EU-data differentiator). Not applicable
  to us directly.
- No evidence of per-domain reputation metrics (complaint/bounce rate) like our ADR 0011 — our
  deliverability-metrics work appears **ahead of Keila**.

## 7. Transactional

- **Transactional email via API** shipped **June 2026** (order confirmations, password resets,
  magic links) with reusable **MJML/HTML/plain-text templates**.
- Direct parity with 1mail **ADR 0005 (transactional send surface)** + CONTEXT "Transactional
  send". Both platforms now pitch "one service for marketing *and* transactional." Notable that
  a mature OSS newsletter tool only added this a month ago — validates it as the right
  differentiating surface, and we're at parity by design.
- Not confirmed whether Keila's transactional templates bind **by reference** (live) the way our
  model deliberately splits (marketing copies, transactional references — CONTEXT/Template).
  (Flagged.)

## 8. Automations — Keila's gap, our thesis

- **Email Automations / drip = roadmap `#132`, status *Planned*** (8 votes, 20 comments). Not
  built. No visual workflow builder, no event triggers, no delays/waits, no enrollment model.
- 1mail already models this: **Automation / Trigger / Enrollment / Step** (CONTEXT), backed by
  **ADR 0001 (send-eligibility)** and **ADR 0002 (unified identity)**, with per-Automation
  unsubscribe scopes. This is the whole point of the product.
- **No on-site behavioral tracking** in Keila at all — no JS snippet, no `/collect`-style
  ingestion, no anonymous→identified visitor stitching. Keila only knows email opens/clicks.
- **Net:** Keila is a *newsletter* tool reaching toward automation; 1mail is an *automation +
  CDP* tool that also does newsletters. Different centers of gravity. On automation and
  behavioral segmentation **we lead**; there is nothing to copy from Keila here because it
  doesn't exist yet.

## 9. What's worth taking (ranked)

1. **Public campaign archive links** (Keila §2). Low-cost, genuinely nice: a public,
   shareable web version of a sent Broadcast. Fits our SPA-renders-everything stance (a public
   route rendering stored Message content). No consent implications (content only). *Strong
   candidate — small, additive, no core-model change.*
2. **Segment leaf-operator checklist** (Keila §4). Before we ship the Segment builder, make
   sure the rule-query DSL covers `$like`/wildcard, `$empty`/is-unset (null vs empty vs missing),
   and the full `< <= > >=` comparison family — not just equality. `$empty` semantics
   (distinguishing null / empty / unset) are easy to under-spec. *Adopt as a spec checklist,
   not code.*
3. **Sending-domain setup wizard with live DNS checks** (Keila §6). UX reference for our
   ADR 0010 verification screen: show SPF/DKIM/DMARC status inline with copy-paste records and
   a re-check button, framed as a guided flow rather than a raw records table. *UX pattern to
   borrow when we build the sending-domain screen.*
4. **Low-friction first-send onboarding** (Keila "Send with Keila", §6). For the eventual SaaS,
   the "verify DNS, then send" wizard (and optionally a shared warm-up domain) lowers time-to-
   first-send. *Note for SaaS onboarding; revisit when cloud onboarding is scoped.*

**Explicitly NOT taking:**
- Multiple parallel editors (we keep MJML-only — `simple-and-powerful`).
- Schemaless JSON custom-data bag (we keep typed, governed Custom fields — CONTEXT anti-vocab
  "Trait"). Keila's dot-notation flexibility is real but costs the governed field list our
  segment UI depends on.
- MongoDB-style raw JSON query as the *user-facing* segment format (we have a structured rule
  builder; borrow the operator *set*, not the hand-written-JSON UX).
- Email-only identity (we keep multi-key Contact + Visitor + Identify).

## 10. Uncertainty flags

1. Contact subscription/status model — assumed a stored per-contact flag; **not documented**,
   not verified.
2. Whether transactional templates bind by-reference (live) vs copied — not documented.
3. Segment `messages` behavioral filter — confirmed for email engagement only; unknown if any
   non-email event filtering exists (docs suggest not).
4. "Shared domains" sending — mechanics (pooled reputation? DKIM alignment?) not detailed.
5. No automation/drip in product is confirmed by the roadmap (*Planned*), but a partial/hidden
   capability can't be fully ruled out from docs alone (low risk — roadmap is explicit).

---

## Sources

- https://www.keila.io/ · /docs · /docs/segments · /docs/contacts · /docs/campaigns · /roadmap · /updates
- https://github.com/pentacent/keila
- https://elixirforum.com/t/keila-open-source-mailchimp-alternative-built-in-elixir/37476
