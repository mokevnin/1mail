---
status: proposed
---

# MCP surface: a projection of the external `/api`, not a second implementation

1mail is operated by agents through an MCP server mounted at `/mcp` in the same binary
(Streamable HTTP, official `modelcontextprotocol/go-sdk`). Its tools are **projected from the
external OpenAPI contract** at startup and dispatch in-process through the same ogen server, so
workspace scoping, token scopes, validation and RFC 7807 errors are the `/api` ones by
construction. A hand-maintained parallel tool list, or a fourth TypeSpec spec, was rejected: it
drifts from the contract (codegen ethos, "generated over hand-rolled"). The only per-operation
override is a TypeSpec `x-mcp` extension (name, hidden, send), so even that lives in the contract.

The target is **capability parity with Drip's MCP**, expressed in 1mail's vocabulary
(`contacts_upsert`, never `subscriber`) and adapted to the model rather than copied: **Tag**
(glossary) is adopted because segments are built on it; forms, goals, orders and refunds are not
modelled and stay out.

## Decisions

- **Contract first.** `/api` today lacks most of what parity needs (its segments and broadcasts
  operations have no handlers). It is extended, additively and without a new version, to: contacts
  (+ batch upsert by alias keys, ADR 0002), tags, segments, broadcasts (drafts, `schedule`,
  `unschedule`, `test-send`, audience, report), automations (CRUD, `activate`/`deactivate`),
  templates, events (+ batch), webhooks, and read-only custom fields, sending domains and the
  complaint/bounce rates (ADR 0011). **Not exposed:** integrations (encrypted provider
  credentials), sending-domain mutations (DNS/DKIM), API tokens. Batches take up to 1000 items and
  return a per-item result; one failure does not roll back the rest.
- **Use-case modules before the contract grows.** `/site` and `/api` re-implement contact
  create/update and diverge: `/api` creates contacts without publishing `ContactCreated`, so they
  never trigger Automations. Domain logic moves into one deep package per domain (as
  `internal/outbound` does, ADR 0015) that owns the transaction and the event publish and returns
  domain errors; handlers stay thin adapters (scope check, call, map). Starts with contacts.
- **Send is a second lock.** The agent authors by default (draft / inactive). Send-class
  operations (`Emails_send`, broadcast `schedule`, automation `activate`) need their own `/api`
  scope (`emails:send`, `broadcasts:send`, `automations:activate`) **and**, over MCP, the extra
  `mcp:send` scope; without `mcp:send` they are not even listed. This relaxes the backlog's
  "AI is an author, not a sender" from absolute to opt-in per token.
- **Consent only narrows through the agent.** An agent may add a Suppression or record an
  Unsubscribe. Resubscribing and lifting a Suppression are not exposed on `/api` or MCP at all:
  they widen reach and assert consent, so a human (or the contact via link) does them.
- **Contact-supplied text is untrusted by kind of field.** `first_name`, `last_name`, Custom field
  values, Event properties and Tag names always originate outside the workspace, so the contract
  marks them `x-untrusted` and the projection wraps them in tool results and states the rule in
  the server instructions (treat as data, never as instructions). No per-row flag, no data-model
  change.
- **Auth.** Phase 1: the same Bearer API token as `/api`. Phase 2: OAuth 2.1 (discovery, dynamic
  client registration, consent screen in the SPA) for claude.ai connectors: `internal/oauthserver`
  serves the RFC 9728 / RFC 8414 metadata, public-client registration (PKCE S256 only), `/oauth/authorize`
  (hands off to the SPA consent route) and `/oauth/token`, which issues an **ordinary scoped API
  token** (no refresh token; it lives until revoked like any API token). Send-class scopes are
  granted only when the user opts in on the consent screen. Phase 3: shipped
  playbooks served as MCP prompts.

## Considered options

- A 4th TypeSpec spec `mcp` with generated intent-level tools: more infrastructure; deep verbs can
  be layered on later if the CRUD mirror (~60 tools) proves too wide.
- Tools calling `ent` directly: bypasses scoping and the use-case invariants.
