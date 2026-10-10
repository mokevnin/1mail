# Outbound send prior art: Listmonk, BillionMail, Notifuse, ent multi-tenancy

Researched October 2026 from primary sources: source code cloned at each repo's default branch
(shallow clone, commit SHAs below), the ent repo's own docs (`doc/md/*`, which are the source of
entgo.io/docs), and RFC 8058. Permalinks use the commit SHA so line numbers stay valid.

| Repo                    | Branch | Commit                                     |
| ----------------------- | ------ | ------------------------------------------ |
| knadh/listmonk          | master | `82db22ce7a068b4015d401c60798265b49c4978e` |
| Billionmail/BillionMail | dev    | `fc36c76c050c3775c5e899faf7403cf0262d2744` |
| Notifuse/notifuse       | main   | `7516397cc5c1eadc2a6aa67c733261043a630a62` |
| ent/ent                 | master | `2b829a033da80bde26e7bb4b5bd768d1f577e8a4` |

Base URLs: `https://github.com/knadh/listmonk/blob/82db22c/<path>#L<n>`, and likewise for the others.

## Question

sphericon has three send surfaces (Broadcast, Automation, Transactional) that each re-implement
Send-eligibility, unsubscribe footer / `List-Unsubscribe` headers, the sending-domain gate and
`email.sent` outbox publication, with no single chokepoint. What do mature open-source senders do
for the same concerns, and what does ent officially support for enforcing a workspace predicate
on every query and mutation? This note reports facts only; it proposes no sphericon interface.

## Summary of findings

- Listmonk keeps **three separate states**: subscriber status (`enabled|disabled|blocklisted`),
  per-list subscription status (`unconfirmed|confirmed|unsubscribed`) and bounce records. A global
  "blocklist" is just `subscribers.status`; there is no separate suppression table.
- Listmonk applies eligibility **at batch-fetch time in SQL**, not per message: `next-campaign-subscribers`
  filters `s.status != 'blocklisted'` and the subscription-status rules, 1000 subscribers per batch
  (`app.batch_size`). Messages already in the in-memory queue are not re-checked.
- Listmonk's **transactional `/api/tx` bypasses the campaign pipeline and has no blocklist/unsubscribe check**:
  it looks the subscriber up and pushes straight onto `msgQ`. Both paths converge only at the
  `Messenger.Push` call in the same `worker()` goroutine.
- Listmonk adds `List-Unsubscribe` + `List-Unsubscribe-Post: List-Unsubscribe=One-Click` **in the worker**, for campaign
  messages only, gated by the global setting `privacy.unsubscribe_header` (default true). The one-click POST hits the same
  URL as the human page, `POST /subscription/:campUUID/:subUUID`, authorised only by the two UUIDs (no HMAC). RFC 8058 section 3.1
  only says the URI SHOULD carry "an opaque identifier or another hard-to-forge component", so a UUID pair is within the RFC
  (whether Listmonk's UUIDs are random v4 was not checked); the defect is the placeholder UUID below, not the lack of a signature.
- Listmonk has an open, relevant regret: opt-in confirmation mails get a one-click header pointing at a **placeholder
  all-zero campaign UUID**, so a provider's one-click POST "succeeds" and changes nothing
  (issues #3063, #3250, both open).
- Listmonk bounce/complaint handling is a separate path (SES/SNS and others via webhooks, plus POP mailbox)
  that records a bounce and applies a configurable action (`none|unsubscribe|blocklist|delete`) after N bounces of a type.
- Notifuse is closest to a chokepoint: broadcast and automation both enqueue into **one `email_queue` drained by one worker**
  (retries, circuit breaker, rate limiter). But eligibility checks are still duplicated at each producer, and
  **transactional bypasses the queue** and calls the email service directly. Its license is now BSL 1.1, not AGPL.
- BillionMail is a Go (GoFrame) control plane around Postfix/Rspamd/Dovecot: it filters inactive contacts and
  "abnormal recipients" (bounce count >= 3) when it builds a task, and finds no `List-Unsubscribe` header generation in
  app code (only DKIM sign-header lists). Treat it as weak prior art for this question.
- ent supports workspace scoping through **Privacy policies with `Filter` rules (entql) in a mixin**
  (official `examples/privacytenant`), or **Traverse interceptors + hooks** (the soft-delete pattern). Generated code
  evaluates query policy/traversers in `prepareQuery`, which also runs for Group By, Select/Aggregate and eager loading.
  Raw SQL (`sql/execquery`, `*sql.DB`) bypasses all of it.
- ent documents Postgres RLS alongside ent (`doc/md/migration/rls.mdx`, `examples/rls`) using `sql.WithVar`/`WithIntVar` session
  variables, but the guide's schema-management path is flagged as an **Atlas Pro** feature.

## A. Listmonk (AGPL-3.0, Go)

License: `LICENSE` is GNU AGPL v3 (header of file).

### A1. Blocklist vs unsubscribe vs subscription status; where checks run

Data model (`schema.sql`):

- `subscriber_status` = `'enabled','disabled','blocklisted'` ([schema.sql#L4](https://github.com/knadh/listmonk/blob/82db22c/schema.sql#L4)).
- `subscription_status` = `'unconfirmed','confirmed','unsubscribed'` ([schema.sql#L5](https://github.com/knadh/listmonk/blob/82db22c/schema.sql#L5)); stored per (subscriber, list) in `subscriber_lists`.
- `bounce_type` = `'soft','hard','complaint'` ([schema.sql#L9](https://github.com/knadh/listmonk/blob/82db22c/schema.sql#L9)).
- Default bounce actions: soft count 2 -> `none`; hard count 1 -> `blocklist`; complaint count 1 -> `blocklist`
  ([schema.sql#L287](https://github.com/knadh/listmonk/blob/82db22c/schema.sql#L287)). `privacy.allow_blocklist` defaults true ([L255](https://github.com/knadh/listmonk/blob/82db22c/schema.sql#L255)).

Semantics of the three:

- Blocklisting a subscriber sets `subscribers.status='blocklisted'` **and** sets every `subscriber_lists` row to `unsubscribed`
  (`blocklist-subscribers`, [queries/subscribers.sql#L214-L220](https://github.com/knadh/listmonk/blob/82db22c/queries/subscribers.sql#L214-L220)).
- A plain unsubscribe link only flips `subscriber_lists.status` for the lists attached to the campaign; if the user ticks
  "blocklist" (and `privacy.allow_blocklist` is on) it blocklists and unsubscribes from all lists
  (`unsubscribe-by-campaign`, [queries/subscribers.sql#L251-L267](https://github.com/knadh/listmonk/blob/82db22c/queries/subscribers.sql#L251-L267);
  handler [cmd/public.go#L271-L284](https://github.com/knadh/listmonk/blob/82db22c/cmd/public.go#L271-L284)).
- Suppression is therefore a **column on the subscriber**, not a separate address-keyed table. There is no workspace
  concept; a deleted subscriber loses the blocklist state. (Inference from the schema; no suppression table exists.)

Where checks run: **campaign-build / batch-fetch time, in SQL**, not per message.

- `next-campaigns` computes `to_send` with `s.status != 'blocklisted'` and the optin/unsubscribed rules
  ([queries/campaigns.sql#L205-L213](https://github.com/knadh/listmonk/blob/82db22c/queries/campaigns.sql#L205-L213)).
- `next-campaign-subscribers` picks the next page with `s.status != 'blocklisted'`, regular campaigns on single-optin
  lists take `sl.status != 'unsubscribed'`, double-optin lists take only `confirmed`
  ([queries/campaigns.sql#L346-L361](https://github.com/knadh/listmonk/blob/82db22c/queries/campaigns.sql#L346-L361)).
- Page size is `app.batch_size` = 1000 ([schema.sql#L237](https://github.com/knadh/listmonk/blob/82db22c/schema.sql#L237)).
  Subscribers fetched into a batch are rendered and queued without a re-check
  ([internal/manager/pipe.go NextSubscribers](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/pipe.go#L82-L145);
  worker [manager.go#L490-L587](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/manager.go#L490-L587) has no status check).
  Consequence (inferred): an unsubscribe landing after a batch was fetched still sends within that batch.

### A2. Send pipeline and transactional

- `Manager` owns `nextPipes chan *pipe`, `campMsgQ chan CampaignMessage`, `msgQ chan models.Message`
  ([manager.go#L100-L102](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/manager.go#L100-L102)); all in memory, buffered `concurrency*message_rate*2`.
- `scanCampaigns` ticks, calls `store.NextCampaigns`, creates a `pipe` per running campaign
  ([manager.go#L450-L487](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/manager.go#L450-L487)).
  `Run()` spawns `Concurrency` workers and loops `pipe.NextSubscribers()` batches
  ([manager.go#L282-L333](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/manager.go#L282-L333)).
- Rate limits: per-worker `MessageRate` msgs/sec with `time.Sleep(time.Second)` ([manager.go#L507-L513](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/manager.go#L507-L513)),
  plus an optional global sliding window (`SlidingWindowRate` per `SlidingWindowDuration`, [pipe.go#L98-L120](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/pipe.go#L98-L120)).
  Defaults: concurrency 10, message_rate 10, max_send_errors 1000 ([schema.sql#L235-L238](https://github.com/knadh/listmonk/blob/82db22c/schema.sql#L235-L238)).
- Errors: `pipe.OnError` counts failures and **pauses the campaign** at `MaxSendErrors` ([pipe.go#L150-L165](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/pipe.go#L150-L165)).
  Retries are inside the SMTP messenger only (`max_msg_retries: 2`, `msg_retry_delay: 10ms`, [schema.sql#L282](https://github.com/knadh/listmonk/blob/82db22c/schema.sql#L282));
  there is no durable per-message queue or retry table (inferred from the in-memory channels; `sent` counts are persisted at pipe cleanup).
- **Transactional (`POST /api/tx`)**: [cmd/tx.go SendTxMessage](https://github.com/knadh/listmonk/blob/82db22c/cmd/tx.go#L16-L160) resolves the
  subscriber by id/email (modes `default`, `fallback`, `external` create an ephemeral subscriber), renders a tx template and calls
  `a.manager.PushMessage(msg)` ([tx.go#L146](https://github.com/knadh/listmonk/blob/82db22c/cmd/tx.go#L146)), which feeds `msgQ`
  ([manager.go#L213-L228](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/manager.go#L213-L228)).
  `msgQ` is drained by the **same `worker()`** in the "Arbitrary message" case ([manager.go#L572-L583](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/manager.go#L572-L583)),
  so it shares workers and the messenger but **not** the per-campaign pipe, the rate limiter branch, the unsubscribe header, or any status check.
  A `grep -n blocklist` over `cmd/tx.go` and `internal/manager/*.go` returns no match; `get-subscriber`
  ([queries/subscribers.sql#L2-L9](https://github.com/knadh/listmonk/blob/82db22c/queries/subscribers.sql#L2-L9)) does not filter on status.
  So a tx send to a blocklisted subscriber is not blocked by code in these files (verified by reading; not exercised at runtime).
  I found no GitHub issue stating this is intentional or a bug (searches returned nothing); treat the intent as unverified.
  The tx API docs (`docs/docs/content/apis/transactional.md`) describe the `default` / `fallback` / `external` subscriber modes and say nothing about blocklisted subscribers,
  which confirms the behaviour is undocumented rather than stated.

### A3. Unsubscribe link, RFC 8058 headers, bounces/complaints

- Link: `UnsubURL = {root}/subscription/%s/%s` = campaign UUID + subscriber UUID ([cmd/init.go#L469-L470](https://github.com/knadh/listmonk/blob/82db22c/cmd/init.go#L469-L470));
  filled per message in [internal/manager/message.go#L13-L24](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/message.go#L13-L24) and exposed to templates as `{{ UnsubscribeURL }}`
  ([manager.go#L401-L406](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/manager.go#L401-L406)). Route: `GET/POST /subscription/:campUUID/:subUUID`
  ([cmd/handlers.go#L271-L272](https://github.com/knadh/listmonk/blob/82db22c/cmd/handlers.go#L271-L272)). No token/HMAC; the UUID pair is the credential.
- Headers: in the campaign branch of `worker()` only, `if m.cfg.UnsubHeader { h.Set("List-Unsubscribe-Post", "List-Unsubscribe=One-Click"); h.Set("List-Unsubscribe", "<"+unsubURL+">") }`
  ([manager.go#L532-L536](https://github.com/knadh/listmonk/blob/82db22c/internal/manager/manager.go#L532-L536)); config `privacy.unsubscribe_header` ([cmd/init.go#L612](https://github.com/knadh/listmonk/blob/82db22c/cmd/init.go#L612)).
  The one-click POST lands on the same handler as the human form, which unsubscribes unless `manage=true`.
  Verified by re-reading the code at the pinned commit (lines 532-536 match).
  RFC 8058 section 3.1 (fetched from rfc-editor.org): the `List-Unsubscribe` header MUST contain one HTTPS URI and `List-Unsubscribe-Post`
  MUST be exactly `List-Unsubscribe=One-Click`; the message MUST carry a valid DKIM signature covering at least both headers;
  the POST MUST NOT include cookies, HTTP authorization or other context; the sender MUST NOT answer the POST with an HTTPS redirect;
  the URI SHOULD include an opaque or hard-to-forge component that the server SHOULD verify.
  Headers are only a global on/off setting, applied post-render; the `models.Message` for tx carries only caller-supplied `Headers`.
- Opt-in confirmation mails use `makeOptinNotifyHook` with dummy UUID `00000000-...` as the campaign, so the POST "succeeds" but changes nothing:
  open issues [#3063](https://github.com/knadh/listmonk/issues/3063) and [#3250](https://github.com/knadh/listmonk/issues/3250) (the latter quotes the code and the `unsubscribe-by-campaign` SQL).
  A maintainer-side comment on #3063 links older issue #2224 and commit `e8fd12b` for history (the thread says removing the header would revert a requested behavior).
- Bounces: routes `POST /webhooks/bounce` (authenticated) and `POST /webhooks/service/:service` (public) when bounce webhooks are enabled
  ([cmd/handlers.go#L221-L236](https://github.com/knadh/listmonk/blob/82db22c/cmd/handlers.go#L221-L236)); providers in `internal/bounce/webhooks/` (ses, sendgrid, postmark, forwardemail, azure, lettermint) and a POP mailbox scanner.
  SES handler parses SNS notifications, handles topic subscription confirmation, maps `Permanent` -> hard, `Transient` -> soft, `Complaint` -> complaint, and **verifies the SNS signature** against a cached cert
  ([internal/bounce/webhooks/ses.go#L66-L213](https://github.com/knadh/listmonk/blob/82db22c/internal/bounce/webhooks/ses.go#L66-L213); note it uses `x509.SHA1WithRSA` at L213, i.e. SignatureVersion 1).
  `Core.RecordBounce` looks up the configured action per type and runs one SQL statement `record-bounce`: count prior bounces of that type, and when `>= count` either blocklists
  the subscriber, unsubscribes all their lists, or deletes them, inserting the bounce row only if the subscriber is not already blocklisted
  ([internal/core/bounces.go#L60-L87](https://github.com/knadh/listmonk/blob/82db22c/internal/core/bounces.go#L60-L87), [queries/bounces.sql#L1-L30](https://github.com/knadh/listmonk/blob/82db22c/queries/bounces.sql#L1-L30)).
  Bounces feed the blocklist only through this action config; the SQL records the campaign and subscriber but not a per-message id.

## B. BillionMail and Notifuse (brief)

### B0. Licenses (from LICENSE files)

- BillionMail: `LICENSE` is **GNU AGPL v3** (header of the file, [LICENSE](https://github.com/Billionmail/BillionMail/blob/fc36c76/LICENSE)).
- Notifuse: `LICENSE` opens "MULTIPLE LICENSES NOTICE": `web_analytics_sdk/` is AGPL-3.0-or-later; **everything else is Business Source License 1.1** (Licensed Work = Notifuse v40.0 and later);
  v39.x and earlier remain AGPL-3.0-or-later; Additional Use Grant forbids production use of six "Licensed Features" without a key (more than 3 workspaces, granular permission sets, SES tenant provisioning, OIDC SSO,
  multilingual template variants, audit log); Change Date = four years after each version's publication, Change License = AGPL-3.0-or-later
  ([LICENSE](https://github.com/Notifuse/notifuse/blob/7516397/LICENSE)). It is source-available, not OSI open source, at the current version.

### B1. BillionMail (GoFrame, Postfix-backed)

1. Model: contact row has `active` (1/0) and `status` (1 = confirmed); unsubscribing sets `bm_contacts.active=0` per group and inserts into `unsubscribe_records`
   ([controller/batch_mail/batch_mail_v1_unsubscribe.go#L40-L75](https://github.com/Billionmail/BillionMail/blob/fc36c76/core/internal/controller/batch_mail/batch_mail_v1_unsubscribe.go#L40-L75)).
   A separate `abnormal_recipient` table holds bounced/bad addresses. Checks run **when a task is created**: `GetActiveContacts` filters `active=1, status=1`
   ([service/batch_mail/batch_mail.go#L330-L337](https://github.com/Billionmail/BillionMail/blob/fc36c76/core/internal/service/batch_mail/batch_mail.go#L330-L337)) and recipients with `abnormal_recipient.count >= 3` are skipped
   ([batch_mail.go#L446-L486](https://github.com/Billionmail/BillionMail/blob/fc36c76/core/internal/service/batch_mail/batch_mail.go#L446-L486)).
2. Pipeline: task + recipient rows executed by `service/batch_mail/task_executor.go`; delivery is Postfix, not a provider API. API sends (`api_mail_send.go`) are a separate code path that
   renders the same unsubscribe link logic (duplicated at [api_mail_send.go#L474-L513](https://github.com/Billionmail/BillionMail/blob/fc36c76/core/internal/service/batch_mail/api_mail_send.go#L474-L513) and [task_executor.go#L1056-L1151](https://github.com/Billionmail/BillionMail/blob/fc36c76/core/internal/service/batch_mail/task_executor.go#L1056-L1151)).
   I did not trace the API path's eligibility checks (unverified).
3. Unsubscribe is a JWT-signed link (`GenerateUnsubscribeJWT`) to a hosted page. A repo search finds `list-unsubscribe` only inside Rspamd DKIM `sign_headers` strings
   ([service/domains/domains.go#L603](https://github.com/Billionmail/BillionMail/blob/fc36c76/core/internal/service/domains/domains.go#L603)), so I found no generation of a `List-Unsubscribe` header in application code.
   A GitHub code search for `List-Unsubscribe` in the repo returned only `conf/rspamd/local.d/dkim_signing.conf`, `core/internal/service/mail_service/fix.go` (the `sign_headers` string, line 210) and `domains.go`;
   code search covers the default branch only and may be incomplete. Whether the header is injected some other way remains unverified.
   Bounces are derived by **parsing the Postfix mail log** (`maillog_stat`, `status=bounced`) and a periodic job upserting `abnormal_recipient` ([service/abnormal_recipient/abnormal_recipient.go#L214-L270](https://github.com/Billionmail/BillionMail/blob/fc36c76/core/internal/service/abnormal_recipient/abnormal_recipient.go#L214-L270)).

### B2. Notifuse (Go, Postgres, multi-workspace)

1. Model: unsubscribe/bounce/complaint are **statuses on the contact-list membership**: `unsubscribed`, `bounced`, `complained` ([internal/domain/contact_list.go#L23-L26](https://github.com/Notifuse/notifuse/blob/7516397/internal/domain/contact_list.go#L23-L26)).
   Broadcast audience queries exclude those statuses in SQL ([internal/repository/contact_postgres.go#L1459-L1461](https://github.com/Notifuse/notifuse/blob/7516397/internal/repository/contact_postgres.go#L1459-L1461)).
   Automation email nodes **re-check** the list status at execute time and exit the contact with the status as reason (skipped for non-marketing template categories), see
   [internal/service/automation_node_executor.go#L250-L275](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/automation_node_executor.go#L250-L275). So the same rule exists in at least two producers (duplication similar to sphericon's).
2. Pipeline: broadcast and automation emails are enqueued as `EmailQueueEntry` with `SourceType` of only `broadcast` or `automation`
   ([internal/domain/email_queue.go#L23-L29](https://github.com/Notifuse/notifuse/blob/7516397/internal/domain/email_queue.go#L23-L29)) and drained by one `EmailQueueWorker` (per-workspace, poll 1s, batch 50, 3 attempts, circuit breaker, per-integration rate limiter;
   [internal/service/queue/worker.go#L16-L60](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/queue/worker.go#L16-L60)). `processEntry` checks the circuit breaker, marks processing, sends, then upserts message history and fires sent/failed callbacks.
   **Transactional does not use the queue**: `TransactionalNotificationService` calls `emailService.SendEmailForTemplate` / `SendEmail` directly
   ([internal/service/transactional_service.go#L775](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/transactional_service.go#L775), [#L1009](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/transactional_service.go#L1009)), and a grep finds no unsubscribe-header/ListUnsubscribeURL logic in it. The shared `internal/service/email_service.go` it calls was also grepped
   for `unsubscrib|bounced|complain|ContactList|suppress|blocklist`: the only hits are comments about unsubscribe links in link rewriting (lines 246-260), so no list-status check was found on the transactional path.
3. One-click: producers put `oneclick_unsubscribe_url` from template data into `EmailOptions.ListUnsubscribeURL` (three separate producers: [queue_message_sender.go#L423-L425](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/broadcast/queue_message_sender.go#L423-L425),
   [automation_node_executor.go#L418-L420](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/automation_node_executor.go#L418-L420), [broadcast_service.go#L1321](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/broadcast_service.go#L1321)); each provider adapter then emits the
   headers (e.g. SendGrid, [sendgrid_service.go#L347-L351](https://github.com/Notifuse/notifuse/blob/7516397/internal/service/sendgrid_service.go#L347-L351)). The one-click endpoint `/unsubscribe-oneclick` is authorised by an **`email_hmac`** over the workspace secret,
   requires the body token `List-Unsubscribe=One-Click` (RFC 8058 section 3.1), and the code comment explains it deliberately does **not** do User-Agent bot filtering
   ([internal/http/public_handler.go#L249-L310](https://github.com/Notifuse/notifuse/blob/7516397/internal/http/public_handler.go#L249-L310)). Bounces/complaints arrive via provider webhooks to `inbound_webhook_event_service.go`
   and set message/contact-list status `bounced`/`complained` (lines ~425, ~868; I did not trace the full path).

## C. ent multi-tenancy (official docs and repo)

Sources: `doc/md/privacy.mdx` (entgo.io/docs/privacy), `doc/md/interceptors.mdx` (entgo.io/docs/interceptors), `doc/md/hooks.md` (entgo.io/docs/hooks), `doc/md/schema-mixin.md` (entgo.io/docs/schema-mixin),
`doc/md/features.md`, `doc/md/migration/rls.mdx` (entgo.io/docs/migration/row-level-security). Paths below are in `ent/ent@2b829a0`.

### C1. The official multi-tenancy example

The Privacy doc has a section "Multi Tenancy" ([privacy.mdx#L292](https://github.com/ent/ent/blob/2b829a0/doc/md/privacy.mdx#L292)); `features.md#entql-filtering` points to it. Runnable code: `examples/privacytenant` (`ent/schema`, `rule`, `viewer`).
Required codegen features: `--feature privacy,entql` (the doc: "The privacy filtering option needs to be enabled using the entql feature-flag"). Shape from the docs:

```go
// TenantMixin: shared mixin embedding tenant_id + edge, and the policy for every schema that mixes it in.
func (TenantMixin) Policy() ent.Policy { return rule.FilterTenantRule() }

// examples/privacytenant/rule/rule.go
func FilterTenantRule() privacy.QueryMutationRule {
	type TenantsFilter interface{ WhereTenantID(entql.IntP) }
	return privacy.FilterFunc(func(ctx context.Context, f privacy.Filter) error {
		tid, ok := viewer.FromContext(ctx).Tenant()
		if !ok { return privacy.Denyf("missing tenant information in viewer") }
		tf, ok := f.(TenantsFilter)
		if !ok { return privacy.Denyf("unexpected filter type %T", f) }
		tf.WhereTenantID(entql.IntEQ(tid)) // "Make sure that a tenant reads only entities that have an edge to it."
		return privacy.Skip
	})
}
```

([privacy.mdx#L398-L470](https://github.com/ent/ent/blob/2b829a0/doc/md/privacy.mdx#L398-L470)). The docs state a `Filter` rule can "limit the scope of the queries a viewer can make, in addition to returning privacy decisions",
and that adding the rule to the mixin applies it to "all schemas that use this mixin" ([privacy.mdx#L461-L462](https://github.com/ent/ent/blob/2b829a0/doc/md/privacy.mdx#L461-L462)).

Exact API names (verified in the doc text and generated templates):

- Interfaces: `ent.Policy` with `EvalQuery(ctx, Query) error` and `EvalMutation(ctx, Mutation) error`; decisions `privacy.Allow`, `privacy.Deny`, `privacy.Skip` ([privacy.mdx#L19-L63](https://github.com/ent/ent/blob/2b829a0/doc/md/privacy.mdx#L19-L63)).
- Rule types in the generated `privacy` package: `privacy.QueryRuleFunc`, `privacy.MutationRuleFunc`, `privacy.FilterFunc`, `privacy.QueryMutationRule`, `privacy.DecisionContext(ctx, privacy.Allow)` to bypass rules ([privacy.mdx#L274-L291](https://github.com/ent/ent/blob/2b829a0/doc/md/privacy.mdx#L274-L291)).
- Mixin interface has `Hooks() []Hook`, `Interceptors() []Interceptor`, `Policy() Policy`; docs note "mixin hooks are executed before schema hooks" and "mixin policy are executed before schema policy" ([schema-mixin.md#L9-L25](https://github.com/ent/ent/blob/2b829a0/doc/md/schema-mixin.md#L9-L25)).
- Registration: must `import _ "<project>/ent/runtime"` or the policy/hooks/interceptors are nil (docs call this out for `Policy`, `Hooks` and `Interceptors`; generated code returns "uninitialized ... (forgotten import .../runtime?)" at [builder/query.tmpl#L391-L394](https://github.com/ent/ent/blob/2b829a0/entc/gen/template/builder/query.tmpl#L391-L394)).
- The doc demonstrates that, with the filter rule also applied on mutations, `client.User.Delete().ExecX(labView)` deletes only that tenant's rows, and `DeleteOne`/`UpdateOne` for another tenant fails with `NotFoundError`
  ([privacy.mdx#L506-L520](https://github.com/ent/ent/blob/2b829a0/doc/md/privacy.mdx#L506-L520), [#L650-L680](https://github.com/ent/ent/blob/2b829a0/doc/md/privacy.mdx#L650-L680)).
  The doc's example **denies admin mutation of tenant data** and lets admins read across tenants (the viewer carries an admin flag).

### C2. Alternative: interceptors + hooks (soft-delete pattern)

- Query side: `ent.Interceptor` / `ent.InterceptFunc(func(next ent.Querier) ent.Querier {...})`, and `ent.Traverser`; with `--feature intercept` the generated `intercept` package gives `intercept.Func`, `intercept.TraverseFunc`, `intercept.NewQuery`
  and a generic `q.WhereP(...)` ([interceptors.mdx#L13-L167](https://github.com/ent/ent/blob/2b829a0/doc/md/interceptors.mdx#L13-L167)). Docs: "Traversers are called one stage earlier, at each step of a graph traversal"; "a Traverse function is a better fit for adding default filters to graph traversals"
  ([interceptors.mdx#L220-L237](https://github.com/ent/ent/blob/2b829a0/doc/md/interceptors.mdx#L220-L237)).
- Mutation side: `ent.Hook` (`func(ent.Mutator) ent.Mutator`), five mutation ops `Create, UpdateOne, Update, DeleteOne, Delete` ([hooks.md#L8-L50](https://github.com/ent/ent/blob/2b829a0/doc/md/hooks.md#L8-L50)); the soft-delete mixin in
  [interceptors.mdx#L241-L360](https://github.com/ent/ent/blob/2b829a0/doc/md/interceptors.mdx#L241-L360) combines a `TraverseFunc` interceptor and a `hook.On(...)` mutation hook, with an opt-out `SkipSoftDelete(ctx)` context key (the same escape-hatch shape is needed for any tenant bypass).
- Generated runtime: a schema `Policy` is installed as `Hooks[0]` calling `Policy.EvalMutation` ([runtime.tmpl#L99-L110](https://github.com/ent/ent/blob/2b829a0/entc/gen/template/runtime.tmpl#L99-L110)) and queries call
  `Policy.EvalQuery` inside `prepareQuery` ([builder/query.tmpl#L390-L424](https://github.com/ent/ent/blob/2b829a0/entc/gen/template/builder/query.tmpl#L390-L424)).

### C3. Coverage and limitations (what the generated code does; the official docs list no explicit limitations page)

- **Group By / Aggregate / Select**: `GroupBy.Scan` and `Select.Scan` call `prepareQuery` (traversers + query policy) and run through `scanWithInterceptors`
  ([builder/query.tmpl#L459-L492](https://github.com/ent/ent/blob/2b829a0/entc/gen/template/builder/query.tmpl#L459-L492)); `Aggregate` is `Select().Aggregate(...)` ([#L385-L388](https://github.com/ent/ent/blob/2b829a0/entc/gen/template/builder/query.tmpl#L385-L388)).
  So the same tenant filter applies (inferred from the templates; not stated in the docs).
- **Eager loading (`With*`)**: for M2M it runs `withInterceptors(..., query.inters)` on the child query ([dialect/sql/query.tmpl#L263](https://github.com/ent/ent/blob/2b829a0/entc/gen/template/dialect/sql/query.tmpl#L263)) and calls `query.prepareQuery` for loaded edges
  ([#L217](https://github.com/ent/ent/blob/2b829a0/entc/gen/template/dialect/sql/query.tmpl#L217)); the child type's own policy/traversers therefore apply. The own-FK (M2O/O2O) and O2M branches of `load<Edge>`
  call `query.All(ctx)` on the child query instead of `prepareQuery` + `withInterceptors` directly (same template, lines after L290), so they go through the child builder's normal `All` path;
  I did not trace `All` itself in this pass, so treat their policy coverage as inferred.
- **Edge traversal** (`user.QueryGroups().QueryPosts()`): each step's Traverse functions run; "Post traverse and intercept functions applied" at the final stage ([interceptors.mdx#L229-L237](https://github.com/ent/ent/blob/2b829a0/doc/md/interceptors.mdx#L229-L237)).
  Interceptors (as opposed to Traversers) only run at the final stage, so a tenant filter written as an `Interceptor` rather than a `Traverser` or policy filter does not filter intermediate hops.
- **Bulk update/delete**: `Update()`/`Delete()` (predicate-based) run the mutation policy/hooks once with a `Filter` and the filter rule narrows the SQL `WHERE`; shown in the doc example above. `CreateBulk` is used in the docs examples with tenant rules ([privacy.mdx#L487-L495](https://github.com/ent/ent/blob/2b829a0/doc/md/privacy.mdx#L487-L495)); `CreateBulk` runs hooks **per row**: the generated `create_bulk` builds one mutator per builder and wraps each with that builder's own `hooks`
  (`for i := len(builder.hooks) - 1; i >= 0; i-- { mut = builder.hooks[i](mut) }`, [dialect/sql/create.tmpl#L450-L453](https://github.com/ent/ent/blob/2b829a0/entc/gen/template/dialect/sql/create.tmpl#L450-L453)),
  so a schema policy (installed as a hook) evaluates for each row.
- **Raw SQL**: `sql/execquery` adds `client.ExecContext/QueryContext` ([features.md#L347-L355](https://github.com/ent/ent/blob/2b829a0/doc/md/features.md#L347-L355), template [execquery.tmpl](https://github.com/ent/ent/blob/2b829a0/entc/gen/template/dialect/sql/feature/execquery.tmpl)); these call the driver directly,
  so no policy/interceptor/hook runs. The same is true for the underlying `*sql.DB` and for `entsql`/`sql/modifier` raw fragments you write yourself (the latter is inferred: modifiers are appended to a query that already passed `prepareQuery`).
- **Hooks are application-level**: the hooks doc says so and refers to database triggers for DB-level logic ([hooks.md#L24-L27](https://github.com/ent/ent/blob/2b829a0/doc/md/hooks.md#L24-L27)).
- **Context dependency**: every rule reads the tenant from `context.Context` (the example uses a `viewer` in ctx); a missing viewer yields a deny, which is the documented fail-closed behavior in the example (`privacy.Denyf("missing tenant information in viewer")`).
- **Create is not filtered by the example's filter rule**: `FilterTenantRule` narrows existing-row predicates; the example adds a separate `DenyMismatchedTenants` mutation rule and a mixin-provided `tenant_id` field for create paths
  ([privacy.mdx#L560-L650](https://github.com/ent/ent/blob/2b829a0/doc/md/privacy.mdx#L560-L650)). Setting `tenant_id` on create from ctx is left to the application: verified in `examples/privacytenant/ent/schema/mixin.go` that `TenantMixin` defines only `Fields` (`tenant_id`, `Immutable`),
  `Edges` (required, immutable) and `Policy`, with no `Hooks()`, so the mixin does not auto-set the tenant on create.

### C4. Postgres row-level security with ent

- Documented: "Using Row-Level Security in Ent Schema" ([doc/md/migration/rls.mdx](https://github.com/ent/ent/blob/2b829a0/doc/md/migration/rls.mdx), entgo.io/docs/migration/row-level-security) and runnable `examples/rls`.
  The policy is plain SQL (`ALTER TABLE "users" ENABLE ROW LEVEL SECURITY; CREATE POLICY tenant_isolation ON "users" USING ("tenant_id" = current_setting('app.current_tenant')::integer);`), combined with the ent schema through an Atlas `composite_schema` in `atlas.hcl`.
- Runtime API: `sql.WithVar(ctx, name, value)` / `sql.WithIntVar(ctx, "app.current_tenant", id)` set a session variable "to be executed before every query" ([dialect/sql/driver.go#L94-L120](https://github.com/ent/ent/blob/2b829a0/dialect/sql/driver.go#L94-L120)).
  Doc: "In real applications, users can utilize hooks and interceptors to set the `app.current_tenant` variable based on the user's context."
- Caveat stated in the doc: the Atlas support for RLS policies used in the guide "is available exclusively to Pro users" (requires `atlas login`). Raw SQL policies applied by other means (e.g. hand-written migrations) are not covered by the guide.
  The example test uses two DB roles (app role vs superuser), implying RLS does not bind table owners/superusers by default (that is general Postgres behavior; the example comment mentions the two roles).
- I found no ent doc that calls RLS "recommended" over privacy policies; the two are presented as separate guides. (No statement either way; unverified beyond the file list.)

## Gaps / unverified

- Listmonk: no issue found confirming whether tx sends skipping the blocklist is intended; behavior read from code, not run, and the tx API docs are silent on it. Subscribers blocklisted after batch-fetch still being sent is inferred from the code, not tested.
  Whether Listmonk's subscriber/campaign UUIDs are random v4 (relevant to the RFC's "hard-to-forge" SHOULD) was not checked.
- Listmonk bounce-webhook coverage of non-SES providers and the POP scanner was only listed, not read.
- BillionMail: API-send eligibility checks, `task_executor` retry behavior, and any `List-Unsubscribe` injection outside the searched files not traced (code search is default-branch only). Treat its Go code as thin evidence.
- Notifuse: webhook-to-status path for bounces/complaints only spot-checked; email-queue worker `processEntry` body past the circuit-breaker check not fully read. Transactional sends were checked in both `transactional_service.go` and the shared `email_service.go` and no list-status check was found.
- ent: no official "limitations" page exists in docs I read; limitations in C3 are derived from generated-code templates and are inferences marked as such. Still unverified: policy application on the own-FK and O2M eager-load branches (they call `query.All`, not traced). Resolved in a second pass: `CreateBulk` runs hooks per row; the example mixin does not auto-set the tenant on create.
- ent RLS: could not determine from the docs whether Atlas Pro is required to _apply_ policies (the guide's composite-schema path says Pro); plain SQL migrations would work independently of Atlas Pro (general Postgres, not an ent claim).
- No runtime testing of any of the four codebases; all findings are from static reading at the commits listed.
