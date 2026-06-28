# 1mail Roadmap — toward drip.com-level marketing automation

## Context

1mail is an open-core marketing automation platform (Go backend + React/Vite frontend,
workspace-scoped multi-tenant). The goal is to incrementally bring the product to the level of
**drip.com**, but as a **general-purpose marketing automation platform** (not the e-commerce-specific
ECRM that drip is — e-commerce is designed-for architecturally and built later).

**Current state (the foundation exists):**
- Contacts (CRUD, custom fields, status), segments (but `definition` is just a string — no engine yet),
  events (immutable log) + tracking (visitor/profile identity resolution), Collect API (identify + events).
- Sending: SMTP/SES integrations (encrypted), `internal/messaging` abstraction + default-provider resolver.
- Infra: watermill (pubsub) and river (jobs) are wired but have no real workers/handlers yet (stubs only).
- Three API surfaces (site/external/collect), scoped API tokens.
- Frontend: contacts, segments, activity feed, settings (tracking/integrations/tokens), profile, auth.
  In the navbar, **Campaigns** and **Automations** are "Coming soon" placeholders.

**Gaps to reach parity (the big rocks):** broadcasts (only a TypeSpec spec exists in `external`, with no
ent schema or handlers), email builder/templates, delivery tracking (opens/clicks/bounces/unsub),
a working segment engine, automations/workflows, onsite forms/popups, personalization (Liquid),
analytics/dashboards, A/B testing, deliverability (DKIM/SPF domains).

For a detailed competitive feature breakdown of drip.com, see
[research/drip-com-feature-analysis.md](research/drip-com-feature-analysis.md).

## Guiding principle: lean on maintained libraries

Minimize hand-rolled code. Where popular, **actively maintained** solutions exist, use them. Confirm a
library's maintenance status at `go get` / `pnpm add` time and pick the living variant.

| Task | Library (instead of hand-rolling) |
|---|---|
| Email templating / merge tags | **`github.com/osteele/liquid`** (Liquid, like drip); fallback `flosch/pongo2` |
| HTML → plaintext (text part) | **`github.com/k3a/html2text`** or `jaytaylor/html2text` |
| Link rewriting + pixel injection | **`golang.org/x/net/html`** (official parser), not regexes |
| Signing tracking tokens | **JWT** (already a dep via go-pkgz/auth) / `golang-jwt`, not hand-rolled HMAC |
| Send queue / retries | **river** (already in the repo) — don't write an engine |
| Email HTML editor | **`@mantine/tiptap`** (official Mantine package) |
| Responsive email (later) | **`github.com/Boostport/mjml-go`** (MJML) |

## Phased roadmap

Each phase builds on the previous ones. The order is chosen so that the first value (real email sending)
appears quickly, and the largest block (automations) lands on a ready foundation of sending + segments +
tracking.

| Phase | Block | Status | Depends on | Summary |
|---|---|---|---|---|
| **1** | **Broadcasts MVP** | ✅ Done | — | One-off email campaigns end-to-end + delivery tracking (opens/clicks/unsub) + per-campaign report. Audience = all active contacts (+ rule segment). |
| **2** | **Segment engine** | ✅ Done | 1 | react-querybuilder rule definition compiled to an ent predicate (attributes + custom fields), preview count, usable as broadcast audience. Events-based conditions still to come. |
| 3 | Email templates + builder | ⬜ Next | 1 | Reusable templates, merge tags / Liquid, test sends, a proper editor. |
| 4 | Automations / Workflows | ⬜ | 1, 2, 3 | Schema (workflow/node/run) + engine (trigger → action → delay → branch → goal) + visual builder (React Flow). The heart of drip. |
| 5 | Forms & onsite | ⬜ | 1, 2 | Signup forms/popups, embed, feeding into contacts/events (on top of Collect API + tracker). |
| 6 | Analytics + deliverability | ⬜ | 1, 4 | Dashboards (aggregates over campaigns/automations), sending domains + DKIM/SPF, suppression/bounce handling, A/B. |
| 7 (later) | E-commerce | ⬜ | 2, 4 | Shopify/Woo connectors, product catalog, purchase/cart events, revenue attribution. Enabled architecturally via the events model from Phase 1. |

> **Progress:** Phases 1–2 are implemented, tested, and on `main`. Phase 2 added a
> standalone, domain-agnostic rule engine (`internal/segments`) compiling the
> react-querybuilder format to SQL; segment-based targeting and a deliverable-count
> preview are wired into broadcasts. Remaining for Phase 2 later: event-based
> conditions ("performed action X") and snapshot (static-list) segments.

---

## Phase 1 — Broadcasts MVP (detailed)

Goal of the MVP: a user can **create a broadcast, pick an audience, write an email, send (or schedule) it,
and see a report** (sent / opened / clicked / unsubscribed). This is the first "sellable" value.

### MVP scope (in / out)
- **In:** a single email to an audience; HTML editor; merge tags (Liquid); immediate and scheduled
  sending via the queue; open/click/unsubscribe tracking; per-campaign report.
- **Audience (MVP):** "all active contacts in the workspace" + optionally a **snapshot segment**
  (static list). **Dynamic rule segments — Phase 2** (to avoid pulling the segment engine into the MVP).
- **Out (later phases):** visual drag-and-drop builder, reusable templates, full Liquid feature set,
  A/B, bounce/complaint handling, dedicated domains.

### 1. Data model (ent — `ent/schema/`)
Follow the pattern in `ent/schema/segment.go` (workspace edge, `id` Int64 immutable, timestamps).

- **`Broadcast`** — `workspace_id`, `name`, `subject`, `from_name`, `from_email` (or taken from the
  default integration), `body_html`, `body_text` (auto-generated), `segment_id` (nullable → null = all
  active contacts), `integration_id` (nullable → default), `status`
  (enum: `draft`/`scheduled`/`sending`/`sent`/`failed`), `scheduled_at` (nullable), `sent_at` (nullable),
  aggregate counters (`recipients_total`, `sent_count`, `opened_count`, `clicked_count`,
  `unsubscribed_count`, `failed_count`), timestamps.
- **`BroadcastRecipient`** (per-recipient delivery log — the basis for tracking) — `broadcast_id`,
  `contact_id`, `workspace_id`, `status` (enum: `pending`/`sent`/`failed`), `opened_at`/`clicked_at`
  (nullable), `error` (nullable), `sent_at` (nullable), timestamps. Unique index on
  (`broadcast_id`, `contact_id`) for idempotency.

Register the new edges on `Workspace` (`ent/schema/workspace.go`).
**Order matters** (Atlas diffs migrations from generated `ent/`, not from `ent/schema/`):
1) edit `ent/schema/*.go` → 2) `make generate-backend` (regenerates `ent/`) →
3) `make db-generate name=add_broadcasts` → 4) `make db-migrate`.

### 2. API contract (TypeSpec → ogen → TS)
The frontend talks to the **site API**, so the primary contract goes there.
- New `typespec/site/resources/broadcasts.tsp` (pattern: `typespec/site/resources/segments.tsp`):
  a resource under `/w/{slug}/broadcasts`, CRUD + actions **`POST .../{id}/send`** and
  **`POST .../{id}/schedule`**, a `stats` field on the resource. Wire it into `typespec/site/main.tsp`.
- The existing `typespec/external/resources/broadcasts.tsp` — extend to the richer model later; for the
  MVP touch only the site spec.
- `make generate-typespec` → `make generate-openapi` (TS client + react-query hooks in `src/generated/site`).

### 3. Backend — handlers + converters
- `internal/api/site/broadcasts.go` — handler methods (pattern: `internal/api/site/segments.go`):
  CRUD + `Send`/`Schedule`. Scope every query by workspace.
- goverter mapping for `BroadcastResource` in `internal/api/site/sitemap/` (add methods to the
  hand-written interface next to `sitemap.go`; `converter_gen.go` is regenerated).

### 4. Send engine (river — `internal/jobs/`)
river is chosen for the actual sending (retries/concurrency); worker registration pattern is in
`internal/jobs/worker.go` (currently only `ExampleWorker`).

> **Start river first.** Verified: in `cmd/server/main.go` only watermill runs in a goroutine
> (`application.RunPubSub`); `river.Client.Start()` is NOT called, and `jobs.NewClient` only constructs
> the client. Without this, jobs enqueue and never run. The first step of this block is to wire river
> worker startup (mirroring `RunPubSub` in `app.go`: provide the client via `do.Provide`, run
> `client.Start(ctx)` in a goroutine from `main.go`, stop it in `Shutdown`).

- **`SendBroadcastJob{broadcast_id}`** — moves the broadcast to `sending`, resolves the audience (all
  active contacts or snapshot-segment members), creates `BroadcastRecipient` rows (status=pending), and
  enqueues a `SendMessageJob` per recipient. On completion → `sent`.
- **`SendMessageJob{recipient_id}`** — renders html/text (merge tags via Liquid), rewrites links +
  injects the open pixel + unsubscribe link, resolves the sender via `internal/messaging` (the
  workspace's default enabled integration), sends, writes the result to `BroadcastRecipient`, increments
  the broadcast counters.
- Scheduled sending: `scheduled_at` → river `ScheduledAt` (native delayed enqueue).

### 5. Delivery tracking (public endpoints — `internal/server/`)
Pattern: the public tracker `internal/server/tracker.go` (serves `/t.js`, ingests events).
- A signed token encodes `recipient_id`. Note: `internal/secrets/cipher.go` is **symmetric encryption**
  (for integration configs), not HMAC. Either add a small JWT/HMAC helper (key from `cfg.EncryptionKey`)
  or encrypt `recipient_id` with the existing cipher — don't assume `internal/secrets` provides a signing
  primitive. Endpoints:
  - `GET /e/o/{token}` — 1×1 gif pixel → `opened_at`, increment `opened_count`.
  - `GET /e/c/{token}?u=<url>` — record `clicked_at`, increment, 302-redirect to the original URL.
  - `GET /e/u/{token}` — unsubscribe action → `Contact.status = unsubscribed`, increment
    `unsubscribed_count` (required for compliance — mandatory in the MVP).
- When rendering the email in `SendMessageJob`: rewrite all `<a href>` to `/e/c/...`, add the pixel and
  an unsubscribe footer.
- **Hook for Phase 4 (automations):** record opens/clicks not only as columns on `BroadcastRecipient`
  but also as **first-class rows in `Event`** (`email.opened` / `email.clicked`, with `subject_id` = the
  contact and `properties.broadcast_id`). The recipient columns remain a denormalized convenience. This
  gives automations (the "performed event" trigger) an engagement-event source from day one, so Phase 4
  needs no backfill.

### 6. Frontend (React — `src/`)
- Enable the **Campaigns** item in `src/components/AppNavbar.tsx` (currently disabled "Coming soon").
- Routes (pattern: `src/routes/segments/` and `src/routes/contacts/`): list, create/edit (composer),
  report. Register them in `src/router.tsx`.
- **Composer:** name, audience selector (all active / snapshot segment), subject, from, and an
  **HTML editor** using **`@mantine/tiptap`** (official Mantine package — no custom CSS, Mantine only).
- **Send/Schedule:** buttons calling the generated `siteBroadcastsSend/Schedule` hooks from
  `src/generated/site`.
- **Campaign report:** stat cards (recipients/sent/opened/clicked/unsubscribed) from `stats`.
- i18n strings in `locales/`, then `make generate-i18n-types`.

### 7. Tests (pattern: `internal/api/site/contacts_test.go`)
- Handler tests for broadcasts CRUD + auth/workspace isolation (testhelper.Setup + typed ogen client).
- Engine test: `SendBroadcastJob` creates recipients and sets statuses (with a fake sender).
- Tracking test: hitting the open/click/unsub endpoints updates the recipient/contact and counters.
- Frontend: composer-form tests (`make test-watch`).

### Execution order (commits, directly on `main`, Conventional Commits)
1. `feat`: ent schemas Broadcast + BroadcastRecipient + migration.
2. `feat`: TypeSpec site/broadcasts + regeneration (`make generate`).
3. `feat`: site handlers broadcasts (CRUD) + goverter mapping + tests.
4. `feat`: river worker startup in main.go + SendBroadcastJob/SendMessageJob + render/merge tags + sender resolve.
5. `feat`: public tracking endpoints (open/click/unsub) + link rewrite/pixel/footer.
6. `feat`: frontend — Campaigns navbar + list + composer (@mantine/tiptap) + send/schedule.
7. `feat`: frontend — campaign report (stats) + i18n.

### End-to-end verification (Phase 1)
1. `make setup` / `make dev` — bring up the stack (https://1mail.localhost), mailpit on :8025.
2. Create an SMTP integration (point it at mailpit) in Settings.
3. Add a few active contacts.
4. Create a broadcast → audience "all active" → write an email with `{{ first_name }}` → **Send**.
5. Verify the emails arrive in **mailpit** (http://localhost:8025), merge tags are substituted, and there
   is an unsubscribe link and a pixel.
6. Open the email / click a link / hit unsubscribe → confirm the **campaign report** shows
   opened/clicked/unsubscribed increasing, and the contact becomes `unsubscribed`.
7. Scheduled send: set `scheduled_at` in the future → river sends it on time.
8. `make test` (backend) and `make check` (tsgo + biome + golangci-lint) are green.

### Open questions / later
- **Snapshot segments:** need a membership table (`SegmentMember`) — add as a small extra in Phase 1, or
  defer and keep "all active contacts" only in the MVP. Recommendation: MVP = "all active" only; snapshot
  ships with Phase 2.
- **Bounce/complaint handling** (SES SNS / SMTP DSN) — Phase 6 (deliverability).
- **External API broadcasts** — after the site contract stabilizes.
