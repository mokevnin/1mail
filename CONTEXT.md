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
A person in a workspace's audience, identified by email (unique per workspace). Carries
name, time zone, and arbitrary custom fields. The unit a campaign is sent to. A Contact
has no "unsubscribed" status — whether a message can be sent is answered by the
Suppression list (global) and Topic consent (per-topic), never by a flag on the contact.
_Avoid_: Subscriber, lead, user, recipient

**Event**:
A record that something happened, attributed to an address (`email`/`phone`) and an
`action` name, with optional properties. The raw material behind behavioral segmentation.
Linked to a Contact by matching email within the workspace — there is no foreign key.
_Avoid_: Activity, log entry, signal

**Segment**:
A named, reusable definition of "which contacts match these conditions" — the one
targeting primitive. Stores a rule query (combinator + nested rules over contact fields,
custom fields, and events), evaluated live: membership is never materialized and shifts as
data changes. There is no other segment kind; a segment is always a rule.
_Avoid_: List, audience, group, filter, snapshot

### Send-eligibility

Whether a given message may reach an address is decided in layers — never by a flag on the
Contact: (1) address in Suppression → never; (2) Contact opted out of the message's Topic →
never, unless the message is transactional; (3) otherwise, send.

**Suppression**:
The workspace's authoritative, address-keyed do-not-send registry — the single source of
truth for "can we ever email this address at all". An address lands here on hard bounce,
spam complaint, global unsubscribe, or manual entry. Address-centric (covers addresses with
no contact) and a compliance ratchet (bounce/complaint do not auto-clear). The send path's
hard floor, checked on every send.
_Avoid_: Blacklist, denylist, do-not-mail

**Topic** _(proposed — pending confirmation)_:
A named consent category within a workspace ("Newsletter", "Promotions", "Onboarding").
The unit of marketing consent: a Contact's opt-out is recorded per (Contact, Topic), and
every Broadcast and Automation sends under exactly one Topic. A Topic carries consent only
— it is never a target set ("send to everyone on Topic X" is the rejected List model
leaking back; targeting is always a Segment). Distinct from both the contact pool and
Segments.
_Avoid_: List, mailing list, subscription group, audience

## Anti-vocabulary (rejected concepts)

- **List / mailing list** — the old-school primitive that fused container + targeting +
  consent into one. Deliberately not used: pool = Contacts, targeting = Segment, consent =
  Topic. If a feature wants "the people on list X", model it as a Segment or a Topic.
- **Contact status / subscribed flag** — send-eligibility is not a property of a Contact;
  see Send-eligibility above.
- **Blacklist** — internally the concept is Suppression. "Blacklist" in email means
  external DNSBLs (sender IP/domain reputation), a different concern not modeled here.
