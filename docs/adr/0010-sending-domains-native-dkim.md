# Sending domains: 1mail-native DKIM, verified-domain required to send

To reach deliverability parity, mail must be authenticated (DKIM/SPF/DMARC) and sent from a
domain the workspace controls — today `from_email` is a free string, nothing is signed, and
nothing verifies the From domain. We add a first-class, workspace-scoped **Sending domain**
entity and gate sending on it.

**1mail signs outbound mail itself (native DKIM)** rather than delegating to the provider:
1mail generates a per-domain keypair, signs every message in the send path, and the user
publishes one `selector._domainkey` TXT. This makes the sending identity **independent of the
Integration** — the same verified domain works identically across SMTP, SES, and any future
provider, with no re-verification when transport changes. Keys are stored encrypted (reuse the
existing Tink keyset); signing leans on a maintained Go DKIM library, not hand-rolled.

**A verified Sending domain is required to send.** `from_email` becomes a verified-domain +
local part; the send path rejects (RFC 7807) any From whose domain isn't a verified Sending
domain, on all three surfaces (Broadcast, Automation, Transactional). This is a breaking change
from the free-string `from_email` and needs a migration + an onboarding step.

## Considered options

- **Provider-delegated verification (SES manages DKIM)** (rejected): less code, but per-provider,
  breaks for raw SMTP, and couples the domain to a specific Integration — fragmenting the model
  the moment a second provider lands. Native signing keeps one transport-independent story.
- **Soft enforcement (send unsigned with a warning)** (rejected): unsigned mail from arbitrary
  domains *is* the deliverability hole; it defeats the feature.

## Consequences

- **The gate is DKIM only.** Verification = the DKIM TXT is published and matches our key. SPF and
  DMARC are generated, shown, and status-checked but do **not** block: SPF depends on the
  transport's sending IPs (which native signing deliberately abstracts), and aligned DKIM already
  yields a DMARC pass. Gate on the one thing we control end-to-end.
- **`verified` is a live property, not a one-time flag.** A background re-check re-validates the
  DKIM DNS; if the record disappears the domain reverts to unverified and its sends block, with the
  owner notified the instant it flips (mirrors the Workspace-suspension notify pattern). This
  prevents signing+sending for a domain whose DKIM receivers can no longer validate — silent
  deliverability rot — at the cost that a DNS edit can halt a live workspace (mitigated by
  immediate notification and trivial re-verification).
- Distinct from **Sending source** (the unsubscribe scope) despite the similar name — one is an
  authenticated identity, the other a consent boundary.
