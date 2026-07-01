# 1mail

Marketing automation platform. The domain is workspace-scoped (multi-tenant): every
contact, event, segment, and tracking entity belongs to exactly one Workspace.

## Language

### Core

**Workspace**:
The tenant boundary. Every domain entity belongs to exactly one Workspace, and all
queries are scoped by it. Owns its own secret keys (collect key for browser tracking,
ingest key for inbound provider webhooks).
_Avoid_: Account, tenant, organization, project

**Contact**:
The single record for a person in a workspace — whether fully identified or still anonymous.
Anchored by a stable internal id, with email, phone, and subject_id (the customer's own user
id) as alias keys, each unique per workspace and any of which may be absent. Carries
Custom fields (typed, named attributes beyond the core fields) and owns the Visitors
(devices) seen as this person.
The thing Events attach to and the thing a campaign is sent to. Whether a message may be sent
is *derived* (see Send-eligibility), never a stored status — so anonymous, un-consented
Contacts are harmless. There is no separate tracking "profile": this entity is both the
behavioral identity and the marketing audience member.
_Avoid_: Profile, tracking profile, lead, subscriber, user, recipient

**Custom field**:
A workspace-defined, typed, named attribute on the Contact, beyond the core fields
(email / phone / subject_id / name). The one attribute concept — there is no separate,
schemaless "trait". An unknown key arriving from Identify or an Event payload is
*auto-created* as a Custom field with an inferred type (declared-by-use), but it is a
first-class, typed, renameable definition from the first sight, not an anonymous bag of
keys — so the Segment builder always has a real, governed field list to offer. Core fields
and Custom fields together are a Contact's attributes.
_Avoid_: Trait, property, attribute (unqualified), metadata

**Visitor**:
An anonymous device/browser identity — a `visitor_id` cookie, unique per workspace. Resolves
to a Contact once Identify establishes who it is; before that it may belong to no Contact. One
Contact owns many Visitors (the same person across devices).
_Avoid_: Tracking visitor, session, device, anonymous user

**Identify** (identity resolution):
The act of binding a Visitor to a Contact and asserting that Contact's alias keys
(subject_id / email / phone). It stitches the Visitor's earlier anonymous Events onto the
Contact, so pre-identify behavior becomes visible to segmentation.
_Avoid_: Alias, merge, reconcile

**Event**:
An immutable, append-only record that something happened (an `action`, optional properties,
at a time). Attached to a Contact by **stable identity resolved at ingest** — not by email —
and carries denormalized identity snapshot fields for debugging. Never FK-constrained to the
mutable Contact. The raw material behind behavioral segmentation. 1mail's own delivery and
engagement facts (`email.sent`, `email.opened`, `email.clicked`) are Events too — reserved,
system-generated actions — so engagement is segmentable through the very same machinery as
customer-tracked actions, with no parallel event stream.
_Avoid_: Activity, log entry, signal

**Segment**:
A named, reusable definition of "which contacts match these conditions" — the one
targeting primitive. Stores a rule query (combinator + nested rules over contact fields,
custom fields, and events), evaluated live: membership is never materialized and shifts as
data changes. There is no other segment kind; a segment is always a rule.
_Avoid_: List, audience, group, filter, snapshot

### Send-eligibility

Whether a message on channel C from sending source S may reach destination D is decided in
layers — never by a single flag on the Contact: (1) (C, D) in Suppression → never; (2) (C, D)
unsubscribed from *everything* → never; (3) (C, D) unsubscribed from S → never; (4) otherwise,
send. Transactional messages skip layers 2–3 (they still respect Suppression's hard bounces).
Consent is per-channel: an email opt-out never silences SMS.

**Destination**:
The channel-specific address a message is sent to — an email address on the email channel, a
phone number on SMS, a messenger id on a bot channel. Suppression and Unsubscribe are keyed by
(channel, destination); a Contact may have several (its email, its phone).
_Avoid_: Address (unqualified), recipient, endpoint

**Suppression**:
The workspace's authoritative, (channel, destination)-keyed do-not-send registry for hard,
global facts: hard bounce, spam complaint, and manual bans. Destination-centric (covers
destinations with no contact) and a compliance ratchet (bounce/complaint do not auto-clear).
The send path's global hard floor, checked on every send.
_Avoid_: Blacklist, denylist, do-not-mail

**Sending source** (the unsubscribe scope):
The unit a Contact unsubscribes from — and it is automatic, never hand-authored: the source
*is* the sender. Each Automation is its own scope (unsubscribing from one drip leaves the
others untouched); all Broadcasts share one scope ("broadcasts"); and a reserved *everything*
scope is the deliberate "leave entirely" opt-out (kept separate so a per-source unsubscribe
never silently loses the whole contact). A source only ever *subtracts* opt-outs from a send
— it is never a target set ("send to everyone subscribed to source X" is the rejected List
model leaking back; targeting is always a Segment). In the email, the unsubscribe link is
labelled with the source's display name ("unsubscribe from this mailing list") — that
"mailing list" wording is user-facing copy only, not a reintroduction of the List concept.
_Avoid_: Topic, list, mailing list, subscription group

**Unsubscribe** (the record):
A per-(workspace, channel, destination, Sending source) opt-out from marketing — keyed by
destination, not by Contact, so it survives contact deletion and re-import (the GDPR-erasure
case: the opt-out is the minimum data retained to honor the refusal). Toggleable: a contact can
resubscribe.
Absence means subscribed — there is no positive subscription row and no "subscribed" flag on
the Contact. The default in-email link unsubscribes from the *sending source* only;
"unsubscribe from everything" is a distinct, deliberate action. A bounce or complaint is not
an Unsubscribe; those are Suppression entries.
_Avoid_: Opt-out flag, status

The **scope** of an unsubscribe (what the contact is opted out of) is distinct from its
**attribution** (which sent message provoked it). A broadcast unsubscribe has scope
"broadcasts" (shared) yet is attributed to the one broadcast whose email triggered it — two
separate facts, never one field.

### Sending

**Broadcast**:
A one-off email send to an audience, authored and sent once (draft → scheduled → sending →
sent / failed). Its audience is a Segment (or all Contacts) resolved and **frozen into
Recipients at send time**, then filtered per-recipient by Send-eligibility. Sends under the
shared "broadcasts" unsubscribe scope. There is no "active contacts" audience — eligibility is
derived, not a contact status.
_Avoid_: Campaign, newsletter blast, mailshot, email blast

**Broadcast recipient**:
The per-(Broadcast, Contact) delivery record — the frozen audience snapshot taken at send
time, and the home of per-recipient delivery state (sent / failed) plus a denormalized
**rollup** of engagement (opened / clicked) derived from the underlying `email.opened` /
`email.clicked` Events, for per-broadcast stats. The Events are the source of truth; the
rollup is a convenience view. Contrast Segment, which is a live query and never materialized.
_Avoid_: Send log, delivery row

**Sent** (vs delivered):
"Sent" means a message was *accepted by the email provider* — not that it reached the inbox.
True delivery failures surface afterward as bounces / complaints, which flow into Suppression.
A broadcast's delivery rate is accepted-by-provider ÷ targeted, not inbox delivery.
_Avoid_: Delivered (for the accepted-by-provider sense)

**Transactional send**:
A single-recipient email triggered by the customer's own application through the `/api`
surface (e.g. password reset, receipt, OTP) — the third send surface alongside Broadcast and
Automation, and what makes 1mail one service for marketing *and* transactional. It carries no
Sending source and is **never** authored as a campaign: the app supplies a Destination, a
referenced Template id, and per-call variables, and the content is rendered at send time. It
skips Unsubscribe (Send-eligibility layers 2–3 — you cannot opt out of your own password
reset) but **still respects Suppression** (a hard-bounced or complained address is never
sent to, transactional or not). Unlike marketing, it binds its Template by *reference*, not
by copy (see Template).
_Avoid_: Notification, system email, trigger (that is the Automation term), API send

**Automation**:
A workspace-scoped, event-triggered sequence: when its Trigger fires for a Contact, the
Contact is enrolled and walks the Automation's ordered steps (send email, wait, …). Each
Automation is its own unsubscribe Sending source. Activating starts enrolling; deactivating
stops *new* enrollments but lets in-flight Enrollments finish (deactivate ≠ halt).
_Avoid_: Workflow, flow, journey, drip, campaign

**Trigger**:
The single Event action that enrolls a Contact into an Automation (e.g. "contact.created", a
custom event). Not a separate entity — one Trigger per Automation. Matched by the Event's
resolved identity, not by email.
_Avoid_: Hook, rule, entry condition

**Enrollment**:
One Contact's membership-and-progress in one Automation — enrolled at most once ever
(re-enrollment is deliberately out of scope). Holds the current step and a terminal state:
`completed`, `failed`, or **`exited`** (left early). An unsubscribe from the Automation has two
effects from one action — it records the durable opt-out (scope `automation:<id>`) *and* moves
the active Enrollment to `exited`; suppression and hard bounces likewise exit the Enrollment.
A run never silently keeps walking steps while skipping every email.
_Avoid_: Run, automation run, journey instance, subscription

**Step**:
One node in an Automation's **ordered, linear** sequence; an Enrollment points at exactly one
current Step. Two kinds today: **send** (an email — it holds its *own copy* of Message
content, per the marketing copy-at-author-time rule) and **wait** (a delay before the next
Step). The Enrollment's single "current step" pointer is deliberate: there is no branching,
no parallel paths, no per-step conditions yet. Conditional / branching steps are a real
future want but are **deferred until the sequence builder is real** — not modelled now.
_Avoid_: Action, node, block, stage

**Template** (email template):
A workspace-scoped, named, reusable piece of email content — a **starting point copied at
author time**. A Broadcast or Automation step takes a *copy* of the template's content with no
reference back, so editing or deleting a Template never changes any already-authored or sent
message. A content library, not a live layout that propagates.

This copy-at-author-time rule is for **marketing** sends (Broadcast, Automation send step). A
**Transactional send** binds the *opposite* way: it references a Template by id and renders
its current content with per-call variables at send time, so fixing a typo in a receipt
template instantly corrects every future receipt — without the customer redeploying their
app. Two deliberately different binding models, one per surface: marketing copies (sent
content is immutable history), transactional references (the template is live).
_Avoid_: Layout, theme, master, partial

**Message content** (a value, not an entity):
The reusable shape every email carries — a subject plus an MJML body (compiled to email-safe
HTML on send). A Template is the saved, named instance of it; a Broadcast and each email
Automation send step each hold their own copy; a Transactional send renders a *referenced*
Template's content with per-call variables. The same value across surfaces — copied for
marketing, referenced for transactional.
_Avoid_: Email body, content block

### Operational

**Integration**:
A workspace-scoped, **outbound** connection to an external sending provider — channel-agnostic
by design (email today via smtp/ses; sms reserved), with encrypted credentials and at most one
default per (workspace, channel). Broadcasts and Automations send *through* an Integration (the
workspace default when unspecified). Sending is one half of a loop: bounces and complaints come
back via the Ingest hook and land in Suppression.
_Avoid_: Provider (alone), ESP, sender, connector

**API token**:
A workspace-scoped, **scoped** bearer credential for the external `/api` surface — a public
`prefix` plus a hashed secret, carrying scopes, optional expiry, and revocation. Distinct from
a Key: a token is scoped, revocable, and expiring.
_Avoid_: API key, secret, password

**Key** (collect key / ingest key):
A bare, single-purpose secret string that identifies a workspace for one job — `collect_key`
(browser tracking ingestion) and `ingest_key` (routing inbound provider webhooks). No scopes,
not a bearer token. The deliberate counterpart to an API token.
_Avoid_: Token (for these), API key

**Operator** (platform operator):
A member of the **platform's** staff who acts *across* all Workspaces (suspend an abusive
sender, impersonate for support). Deliberately **not** a User and **not** reached through a
Membership: a distinct identity with its own store and its own auth surface, holding **no**
Membership, so the "every query is scoped by a Workspace" invariant has no exception —
workspace-scoped code has no path that can return an Operator. A person who is both staff and
a customer holds two separate identities (an Operator *and* a User), by design, for
least-privilege and clean audit. A SaaS/platform concept, absent from a plain self-hosted
install.
_Avoid_: Admin (that is a workspace Role), superuser, staff user, root

**Workspace suspension**:
A reversible Workspace state that **freezes all outbound sending** — every one of the three
send surfaces (Broadcast, Automation, Transactional) refuses while it is set — as a
platform reputation-protection measure against an abusive sender. It is *not* a lockout:
login, dashboard reads, `/api` reads, and `/collect` tracking keep working, so the owner can
still see the suspension and appeal. Enforced in the **core** send path (the AGPL binary is
the only reliable choke point). It records **attribution**: the actor that set it (an
automated abuse detector — actor `system` — or a platform Operator) and a reason. The
automated path only fires above a minimum send volume (a rate is noise at low volume),
notifies the workspace owner, and is one-click reversible by an Operator. Suspension is the
*reputation* freeze; a Billing hold is a separate money-driven freeze on the same core send
chokepoint — the two are independent reasons, never merged.
_Avoid_: Ban, lockout, disable, quota (suspension is not a billing state — see Billing hold)

**Webhook endpoint** (outbound):
A workspace-scoped HTTP destination that **1mail calls** when domain events occur — subscribes
to event types (empty = all) and signs each delivery (HMAC). Outbound only; the opposite of an
Ingest hook.
_Avoid_: Webhook (unqualified), callback

**Ingest hook** (inbound provider webhook):
The inbound endpoint that **receives** provider callbacks (e.g. SES bounce / complaint via SNS),
routed by the workspace's `ingest_key`. The opposite direction from a Webhook endpoint; it
feeds Suppression.
_Avoid_: Webhook (unqualified), inbound webhook

**User**:
A human who authenticates (email + password). A User reaches Workspaces through Memberships,
each carrying a Role — a Workspace is never owned by a single User directly.
_Avoid_: Account, member, customer, owner

**Membership**:
The join that grants a User access to a Workspace with a Role — Workspaces are reached through
Memberships, not owned by one User. Laid in now (rather than single-owner) because a multi-user
SaaS is planned and retrofitting access control onto single-owner is expensive.
_Avoid_: Team, seat, collaborator, ownership

**Role**:
A Membership's permission level in a Workspace (e.g. owner / admin / member). The exact set is a
follow-up; the structural decision is that access is role-scoped per Membership.
_Avoid_: Permission, scope (scope is the API-token term)

### Metering & billing

1mail's core *measures* billable activity; it does not *price* it. Metering (what happened,
how much) is a domain concern; rating, plans, invoices, payment, and dunning are **not** — they
live in an external billing plane and never enter the core glossary.

**Usage snapshot** (metering):
A per-(Workspace, billing period, metric) materialized aggregate of billable activity, finalized
(made immutable) at period close — the billing-grade number a biller consumes. Distinct from the
raw Events it is computed from, which stay the source of truth for audit and dispute. Different
metrics aggregate differently: sends are a *sum* over `email.sent`; contact count is a
**high-water-mark** (the peak reached during the period), not an instantaneous value. It
materializes live Events for money the same way a Broadcast recipient's rollup materializes
engagement — the Events are truth, the snapshot is the closed, reproducible figure. Carries no
price.
_Avoid_: Meter, counter, invoice line, quota (quota is enforcement, not measurement)

**Billing hold**:
A reversible Workspace state that **freezes all outbound sending** — all three send surfaces
(Broadcast, Automation, Transactional), exactly like Workspace suspension — but for a *money*
reason (non-payment / plan-limit breach) rather than reputation. It is a **distinct cause on the
same core chokepoint**, never a repurposing of suspension: the send path asks one question ("may
this Workspace send now?") answered by several independent freeze reasons. Two properties set it
apart from suspension: it engages **only after a dunning grace period** (during dunning nothing
is frozen — the grace, not the message type, is what protects a tenant's password-reset mail),
and it is cleared by **payment (self-service)**, whereas suspension is cleared by appeal. The
state and its dunning lifecycle are an EE concept; only the freeze check lives in the core send
path.
_Avoid_: Suspension (that is the reputation freeze), quota, lockout, dunning (dunning is the
grace period, not the hold)

## Channels & future surfaces

The identity spine — **Contact + Visitor + Event** — is channel-agnostic and is the stable point
every current and future surface plugs into. Sending is modelled *per channel*: an Integration
has a channel, Send-eligibility is keyed by (channel, Destination), and Message content is
channel-specific. Email is the only built channel; SMS is reserved.

Deferred, additive surfaces — **not modelled until their shape is real**, and none should reshape
the core now:

- **On-site popup / form** — a capture-and-display surface; it feeds Events + Identify through
  the tracker. Additive entity, not a core change.
- **Messenger bot** — just another channel (a Destination is a chat id); covered by the
  channel-aware send model.
- **On-site chatbot** — a separate, two-way conversational context; shares only the identity
  spine. Explicitly out of the send/eligibility/sequence model.

## Anti-vocabulary (rejected concepts)

- **List / mailing list** — the old-school primitive that fused container + targeting +
  consent into one. Deliberately not used: pool = Contacts, targeting = Segment, consent =
  per-(Contact, Sending source) Unsubscribe. If a feature wants "the people on list X", model
  it as a Segment.
- **Tracking profile** — there is no separate person record for the behavioral/CDP side; the
  Contact *is* the identity. (The old tracking profile is absorbed into Contact.)
- **Trait** — the schemaless CDP attribute. Not a separate concept: every non-core attribute
  is a typed, named Custom field (auto-created on first sight). We keep one governed attribute
  notion, not raw-traits-plus-promoted-fields.
- **Prospect** — a dropped, never-read flag. "Not yet a real contact" is not a stored term;
  it is the anonymous/un-identified state of a Contact, and mailability is derived anyway.
- **Contact status / subscribed flag** — send-eligibility is not a property of a Contact;
  see Send-eligibility above.
- **Blacklist** — internally the concept is Suppression. "Blacklist" in email means
  external DNSBLs (sender IP/domain reputation), a different concern not modeled here.
