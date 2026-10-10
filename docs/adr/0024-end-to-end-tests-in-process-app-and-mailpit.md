---
status: accepted
---

# End-to-end tests: in-process application, Mailpit, driven through the public API

Handler tests drive handlers through an in-memory recorder, inside a rolled-back transaction, with mail captured in
memory. Nothing exercises the real chain: HTTP API, event bus, job queue, MIME build, SMTP delivery, the message a
recipient gets and the link they follow. We add a black-box end-to-end suite that boots the whole application,
sends through a real SMTP server and observes the result only in that server's inbox.

## Decisions

- **One seam: the public HTTP API (`/api`, `/collect`) in, a Mailpit inbox out.** Tests never read the database, the
  event outbox or an in-memory sender.
- **The application runs in the test process**, with the real event router, job queue and HTTP server, started on a
  caller-provided listener so the public URL (the base of unsubscribe, confirm and tracking links) is the test
  server's own address. In-process beats driving the compiled binary: it is faster, debuggable, and reuses the
  existing run, stop and shutdown entry points. Binary and reverse-proxy routing are left to a separate smoke test.
- **A dedicated database, rebuilt on every run, and a fresh Workspace per test.** The per-test transaction rollback
  used elsewhere (go-txdb) cannot work here: the job queue and bus workers use their own connections and would never
  see uncommitted rows. Everything is Workspace-scoped (ADR 0017), so tests need no cleanup and may run in parallel.
- **Mailpit is started by the suite on free ports**, not shared with the dev stack, so the suite never collides with a
  running `mise run dev` and needs no Docker. Mail is matched by a recipient unique to the test. Mailpit's own API
  is read through a thin typed helper: it publishes only Swagger 2.0, which the project's OpenAPI generator does not
  accept. The helper sits behind each Workspace's Inbox (one wait, one absence check, cleanup of what it saw), so
  scenarios never call it.
- **Only the arrange step is outside the API:** creating the User, Workspace and first API token with the product's
  own domain functions. Everything else is an API call. The bootstrap token is not used; it addresses only the
  oldest Workspace.
- **Sending domain verification uses a harness-only DNS lookup.** The send path requires a live DKIM check
  (ADR 0010); the suite injects the existing development lookup, which echoes a domain's own stored key, through an
  e2e-only app option (`app.WithE2EDKIMLookup`, refused outside the e2e profile) rather than the development flag. Because that lookup resolves by domain name alone
  while uniqueness is per (Workspace, domain), every test uses a unique Sending domain.
- **Asynchronous steps are polled with a bounded timeout.** The job queue clock is not controlled, Automation wait
  steps are not used and Deferral (ADR 0023) is out of scope.
- **The external API gains what the scenarios need** (Integration and Sending domain), because the API is meant to
  expose every capability and the MCP surface is a projection of it (ADR 0016). Only what the scenarios require is
  added; the rest of the parity gap is tracked separately.
- **Scenarios are written as domain steps** (import contacts, send a Broadcast, wait for an email), with the
  transport inside the harness. An MCP-driven run can then replay the same scenarios on a second transport. No
  common driver interface is introduced until that second implementation exists.
- **The suite is its own package behind a build tag, with its own task and CI job.** The default test task does not
  run it.

## Exception to "tests build on committed fixtures"

Tests normally start from committed YAML fixtures. End-to-end tests deliberately create their own User, Workspace
and data through the API, because fixtures rely on the transaction rollback that this suite cannot use. The
exception covers the end-to-end package only; every other test keeps the fixtures rule.

## Considered options

- **Compiled binary behind the reverse proxy:** truest to production, but slow, hard to debug and unable to choose
  its own public URL. Kept for a separate smoke test.
- **Shared Mailpit daemon from the dev stack:** simpler, but collides with a running dev stack and shares one inbox.
- **testcontainers:** adds a Docker dependency the toolchain otherwise avoids.
- **Provisioning through the site API with a cookie login:** needs a login flow the suite does not otherwise test and
  hides the missing external resources instead of fixing them.
