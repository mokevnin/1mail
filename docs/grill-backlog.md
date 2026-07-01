# Grill backlog

A prioritized queue of **design decisions worth grilling** — each clears the bar of
*hard-to-reverse + surprising + real trade-off*, and each output is a glossary term
(`CONTEXT.md`) and/or an ADR (`docs/adr/`). This is the working queue for
`/grill-with-docs` sessions; it is not itself authoritative (CONTEXT.md + ADRs are).

Legend: 🔴 launch-blocker / "can't send otherwise" · 🟡 parity with drip.com · ⚪ later.

## Done (grilled)

- ✅ **Platform abuse control** — Workspace suspension (core mechanism) + Operator identity.
  → ADR 0007, ADR 0008; CONTEXT: *Operator*, *Workspace suspension*.
- ✅ **Sending domains** — native DKIM, verified-domain-required-to-send.
  → ADR 0009; CONTEXT: *Sending domain*.

## Queue

### 🔴 Bulk-sender compliance (Gmail/Yahoo 2024+)
Table-stakes to deliver at all. Open forks:
- One-click unsubscribe (`List-Unsubscribe` + `List-Unsubscribe-Post`, RFC 8058) — how the
  signed token maps to the existing per-(destination, Sending source) Unsubscribe model.
- Complaint-rate as a first-class, monitored metric held < 0.3% (feeds auto-suspension).
- DMARC posture guidance (we already gate on DKIM; what do we tell users about `p=`).
Ties to: Sending domains (ADR 0009), Suppression, Workspace suspension.

### 🔴 Complaint / bounce-rate metrics
Prerequisite for the auto-suspension detector already designed in ADR 0007 — the detector
has nothing to read until these exist. Open forks: per-(workspace, domain) rate windows,
where the rollup lives (Events are source of truth), what thresholds/floors mean.

### 🔴 Billing / plans / metering / quota enforcement (SaaS)
Distinct from suspension: suspension = abuse, quota = commerce. Absent entirely. Open forks:
- Metered units (contacts, emails/month) and where enforcement sits (send path? like
  suspension) vs. suspension (reuse the choke point or a separate gate?).
- Plan/entitlement model; is it core, EE, or SaaS-only.
- Over-quota behavior: block vs. degrade vs. dun.

### 🔴 Double opt-in / confirmed subscription
Regulatory must for EU. The model deliberately rejected a "subscribed flag" — so a confirmed
opt-in flow (confirmation email → activation) has to be modeled *without* reintroducing a
subscription status. Open forks: what "confirmed" is a fact about, how it coexists with the
derived Send-eligibility model, per-source vs. global confirmation.

### 🟡 Automation action steps (beyond send/wait)
Step is only `send` | `wait` today. A whole class is unnamed: update custom field,
add/remove segment membership (tag), call a Webhook endpoint, notify team, enroll into
another Automation. Open forks: which actions are first-class Step kinds vs. deferred; how
"tagging" fits when the model rejects Lists; idempotency/side-effects on re-run.
Grill *before or with* automation branching (they reshape the same Step/Enrollment model).

### 🟡 Automation branching / goals / per-step conditions
CONTEXT.md deferred this "until the sequence builder is real" — the builder now is. Attacks
the stated invariant *"Enrollment points at exactly one current Step… no branching"*. Open
forks: branch/condition model, goals & exit conditions, whether the single-current-step
pointer survives, re-enrollment (currently out of scope).

### 🟡 Time/date-based & recurring triggers
Trigger is a single Event action today. Missing: date-based (birthday/anniversary),
scheduled/recurring (RSS-style newsletter), wait-until-time-of-day / send-time. Open forks:
is a time trigger a synthetic Event or a new Trigger kind; how it reconciles with enroll-once.

### 🟡 GDPR data tooling
Contact data export (portability) + right-to-erasure workflow + consent audit. The model
says the Unsubscribe survives contact deletion; the erasure/export *mechanism* is undefined.

### 🟡 Contact import/export + dedup/merge
Bulk CSV import with field mapping and duplicate resolution; manual merge of duplicate
Contacts. Open fork: how merge interacts with the multi-key identity spine (subject_id /
email / phone) and Event re-attachment.

### 🟡 Lead / contact scoring
Marketing-automation staple, absent. Open forks: is a score a Custom field, a derived
value, or its own concept; what mutates it (Events? automation actions?).

### 🟡 Audit log (workspace-scoped)
Who changed what — team feature + compliance. Open forks: what's audited, retention,
whether Operator actions (cross-workspace) log here or separately.

### 🟡 External `/api` breadth + rate limiting
The external surface is thin (broadcasts stub). Integrations need CRUD for
contacts/events/segments via `/api`, plus per-token rate limiting. Open forks: API shape
parity vs. site API, rate-limit unit (per token? per workspace?).

### 🟡 Deliverability observability
Google Postmaster Tools / feedback loops (FBL) ingestion; pre-send spam-score/render check
(mail-tester-like). Open forks: how FBL complaints reconcile with the SES-over-SNS ingest
path already feeding Suppression.

### ⚪ Later / lower-urgency
- Send-time optimization / timezone-aware sending.
- Dedicated IP pools + the warmup ramp (roadmap has send-rate control; pool model is open).
- Usage telemetry (opt-out) — roadmap flags it as an unwritten ADR candidate (payload +
  consent contract).
- Anonymous-Contact promotion policy — ADR 0002 left open (when/if a Visitor becomes a
  Contact row).
- RBAC roles beyond owner/admin/member (EE), SSO/SAML (EE).
- Impersonation (deferred in ADR 0008 until support load appears).
- Visual MJML editor (roadmap Phase 3 remainder).
- Native CRM / Zapier / e-commerce connectors (roadmap Phase 7).

## Candidate dependencies

Ready, popular, actively-maintained solutions per item, so the dependency surface is known
up front. **Confirm liveness at `go get` / `pnpm add` time** (roadmap principle). ⚠️ flags a
tension to resolve *before* adopting: the **Postgres-only, single-binary self-host** constraint
(needs a Postgres-native path, not Redis/an extra service). Billing/metering is SaaS-only by
nature and therefore has no such tension: self-host simply has no billing.

| Item | Candidate | Kind | Status / note |
|---|---|---|---|
| Sending domains — DKIM sign/verify, DMARC | **emersion/go-msgauth** (`dkim`, `dmarc`, `authres`) | Go lib | MIT, v1, active (Apr 2025). The pick for ADR 0009 signing. |
| " — SPF check | **blitiri.com.ar/go/spf** (albertito/spf) | Go lib | Standard Go SPF; slower-moving (last big update ~2022) but stable. SPF is advisory-only per ADR 0009. |
| " — DNS record lookups (verify TXT/CNAME) | **miekg/dns** | Go lib | De-facto Go DNS lib, very active. |
| Bulk-sender: one-click List-Unsubscribe (RFC 8058) | *no lib* — set `List-Unsubscribe(-Post)` headers via **go-mail** (already a dep) + sign token with **JWT** (already a dep) | build | Headers + existing token signer; nothing new to add. |
| Complaint/bounce-rate metrics | *no lib* — aggregate from `Event` rows in **Postgres** | build | Source of truth already exists (`email.bounced`/complaint Events); no ClickHouse needed. |
| Billing / subscriptions / payments | **stripe/stripe-go** | Go SDK | Official, active. SaaS-only. |
| Usage metering + entitlements/quota | **openmeterio/openmeter** (Go SDK, Stripe sync) or **getlago/lago** | service | SaaS-only, and cleanly so: billing/quotas don't exist in self-host, so these separate services never touch the single-binary path — no tension. |
| Double opt-in | *no lib* — confirmation email + signed token (JWT), existing send path | build | Flow only; must not reintroduce a "subscribed" flag (grill it). |
| Automation action steps / branching | **@xyflow/react** (already used) + **river** (already used) for execution | in-repo | No workflow-engine dep needed; river drives steps. |
| Time/date & recurring triggers | **river** periodic jobs / **robfig/cron** | Go lib | river (already in repo) has periodic jobs — prefer it over a new cron dep. |
| GDPR export / erasure | *no lib* — custom, stdlib serialization | build | Mechanism is custom; policy is the grill. |
| Contact CSV import | **gocarina/gocsv** (or stdlib `encoding/csv`); FE parse **papaparse** | Go + JS | Import mapping + dedup logic is custom. |
| Lead scoring | *no lib* — reuse the `internal/segments` rule engine | in-repo | Scoring rules ≈ segment predicates; build on what exists. |
| Audit log | *no lib* — append-only Postgres table (optionally off the domain-event bus) | build | No dominant Go lib; DB-level `pgaudit` is a different (ops) concern. |
| External `/api` rate limiting | **sethvargo/go-limiter** | Go lib | ⚠️ Memory store = single-node; shared store needs Redis, which 1mail avoids. Either a Postgres-backed limiter store or per-node limits + document it. Grill the unit (per token/workspace). |
| Deliverability: Postmaster/FBL | **google.golang.org/api** (Postmaster Tools) + **jhillyerd/enmime** (parse ARF `message/feedback-report`) | Go lib | FBL complaints reconcile into the existing SES-over-SNS → Suppression path. |
| Pre-send spam score / render | **rspamd** (self-run service) or mail-tester (SaaS) | service | Optional; heavier. |
| Send-time / timezone-aware | *no lib* — stdlib `time` + IANA tz | build | — |
| IP warmup / send-rate | **river** native rate/concurrency limiting | in-repo | Per roadmap; no hand-rolled throttler. |
| Usage telemetry (opt-out) | **posthog/posthog-go** or a minimal custom pinger | Go lib | Payload/consent contract is the grill (unwritten ADR candidate). |
| 2FA (TOTP) / passkeys | **pquerna/otp** / **go-webauthn/webauthn** | Go lib | Already named in the roadmap. |
| Visual MJML editor | **zalify/easy-email-editor** (React + MJML) — chosen | JS | Matches the stack; MJML-native (pairs with the backend **gomjml** compile). Its bundled widget styles are third-party (not hand-written CSS) — same precedent as the already-used **@xyflow/react**, so no conflict with the Mantine-only rule. |
