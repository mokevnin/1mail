# Reference projects: how their tests are architected

Researched October 2026 from primary sources: each repo shallow-cloned at its default branch (commit SHAs
below), reading test code, test helpers, Makefiles and CI workflows. No blog posts. Every claim is either
**verified** (file path and line numbers at the commit SHA, so lines stay valid) or marked **inference** /
**not verified**. Nothing was executed: no suite was run, so statements about speed or flakiness are absent.

| Repo                       | Branch  | Commit                                     |
| -------------------------- | ------- | ------------------------------------------ |
| go-gitea/gitea             | main    | `01fe36985d394eb9f6f66ce6c66b2ec0a278f002` |
| forgejo/forgejo (Codeberg) | forgejo | `5baddea51e362942b3f41faaa7c90c6c3e233217` |
| rudderlabs/rudder-server   | master  | `74940c24ae5e2f35ca939f2c18e17fb00c2c82fa` |
| mjl-/mox                   | main    | `58d0a7788a2e51629e1527dff73167ee6238b100` |
| foxcpp/maddy               | master  | `8d049a208908bfa729b2a0b44cc3af39ba84cf36` |
| hatchet-dev/hatchet        | main    | `82fce9703e358c5158c2998c92e91880d45d2fd1` |
| grafana/grafana            | main    | `891f7973f81c6e4e0a24bed3120cca3b3cc93ebc` |
| mattermost/mattermost      | master  | `47eebd5530c0485391efa77915965b4b7317f16c` |
| dittofeed/dittofeed        | main    | `fe89657bbaad3bf25fa3571288287022f12a8347` |
| postalserver/postal        | main    | `d96eddbadeda600595d6247486f9ceb8e99cbc15` |
| mautic/mautic              | 7.x     | `bc46b64883255608e5b14a41987c207de9535616` |
| plausible/analytics        | master  | `bc2c7b395caddf7bb4eec2548be7958824d07d4a` |
| knadh/listmonk             | master  | `70ae5162f3bf4d435facc7f33d989361602fd7db` |

Base URLs: `https://github.com/<org>/<repo>/blob/<sha>/<path>#L<n>`; Forgejo is on Codeberg:
`https://codeberg.org/forgejo/forgejo/src/commit/<sha>/<path>#L<n>`. Below, paths are given as
`path:line` and the SHA is the one in this table.

## Question

sphericon's test stack: committed YAML fixtures with generated named constants (`fixtures/*.yml` to
`internal/fixtures/catalog_gen.go`), a baseline applied once per process then one go-txdb transaction per
test (`internal/testhelper/testhelper.go`), no direct SQL in tests, a scratch database for fresh-instance
migration tests (`internal/testhelper/scratchdb.go`), and an in-process e2e suite with domain steps on
`e2e.Workspace` and a Mailpit `Inbox` (ADR 0024). Which well-known open-source projects have test
architectures worth reading for ideas, and what exactly is portable? This note reports facts and
per-project verdicts. It does not change any ADR.

## Summary of findings

- **Most relevant by number of sphericon mechanisms touched:** Forgejo, Mattermost, Plausible, Hatchet, mox.
  Ranking is at the end.
- **Grafana deprecated the exact pattern sphericon does not use and moved to the one it nearly has:** a shared
  database truncated between tests is now `Deprecated` and lint-banned (staticcheck SA1019); new tests call
  `NewTestStore`, which gives each test its own temporary database, optionally cloned from a pre-migrated
  template. sphericon's txdb rollback is a stronger and cheaper isolation than either, so this is confirmation,
  not a change request.
- **Gitea reloads fixtures without transactions** and avoids the cost by tracking which tables a test
  dirtied (a SQL-text hook marks them) and reloading only those. txdb makes that unnecessary for Go tests.
  It would matter only for the e2e suite, where the app really commits.
- **Forgejo runs Playwright from a Go test** that boots the app in-process and exposes
  `PATCH /_e2e/fixtures/reload`. It also has a `forgery` package of per-test entity factories, which
  conflicts with sphericon's fixtures-only policy (see verdict).
- **Mattermost writes store contract tests once** (`storetest`) and runs them against each store
  implementation. That maps onto sphericon's smtp and ses providers behind `messaging`.
- **Plausible and Hatchet both run the suite in more than one product mode in CI** (CE vs EE, optimistic
  scheduling on/off, race on/off). That is the direct analogue of sphericon's `ee/` with `WithoutLicense()`.
- **Hatchet's `TestMain` runs `goleak.Find`.** Relevant because river and watermill start goroutines.
- **mox and maddy test mail servers with an in-process or subprocess SMTP client and a mocked DNS**
  (`dns.MockResolver`, `go-mockdns`), no real network. Relevant to sending-domain verification and DKIM.
- **Three projects exercise schema migrations in tests** (Gitea `test-migration`, Plausible `--include
  migrations`, Hatchet `TESTING_MATRIX_MIGRATE=penultimate`). sphericon already runs its real migration path on
  an empty database (`cmd/server/migrate_test.go` calls `applyMigrations`, which runs goose over the embedded
  `migrations` FS, then river's migrator, twice for idempotence). What no project-style test does in sphericon is
  run migrations **against a database that already has rows** (the Hatchet `penultimate` idea). This is an
  observation from reading that one test; other CI jobs were not checked.
- **listmonk: nothing worth taking** (already reviewed, re-confirmed below).

## A. Go projects

### A1. Gitea (MIT, Go, xorm)

**What it solves.** Self-hosted git forge. **Overlap:** large multi-tenant-ish web app with a YAML fixture
corpus, a DB matrix, an in-process integration suite and Playwright e2e. Fork parent of Forgejo.

**Test architecture.**

- Layout: `models/*/` unit tests next to code, `tests/integration/` (268 files), `tests/e2e/` Playwright
  (`*.test.ts`), `models/fixtures/` (78 YAML files, one per table, e.g. `models/fixtures/access_token.yml`).
- Fixtures: hand-written YAML, rows keyed by numeric ids, tests reference ids as literals
  (`unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})`,
  `tests/integration/incoming_email_test.go:31`). No generated named constants. sphericon's catalog_gen is
  stricter.
- DB handling: `unittest.PrepareTestDatabase()` or `tests.PrepareTestEnv` (`tests/test_utils.go:118`) calls
  `LoadFixtures`. The loader does `DELETE FROM <table>` then re-inserts, and only for tables flagged dirty
  (`models/unittest/fixtures_loader.go:146-178`, `MarkTableChanged` at `:180`). Dirtiness comes from an xorm
  hook that inspects the SQL text of every non-SELECT statement and marks the table
  (`models/unittest/fixtures.go:84-137`). Unit tests default to SQLite; integration runs per DB in CI.
- Consistency checks: `unittest.CheckConsistencyFor` (`models/unittest/consistency.go:29`) walks every row of
  a bean type and asserts denormalised counters match reality.
- Integration harness: `tests/integration/integration_test.go` `TestMain` initialises fixtures once, mounts
  the real router (`routers.NormalRoutes()`), and tests use `httptest` with a cookie-jar `TestSession`. The
  suite is compiled to a binary (`Makefile:444-450`), because testlogger forwards app logs to `t.Log`.
- Mail: unit tests swap the package-level sender (`defer test.MockVariableValue(&SendAsync, ...)`,
  `services/mailer/mail_test.go:161`). The incoming-email integration test talks real SMTP to a `smtpimap`
  service container in CI: the service is defined in the mysql job (`.github/workflows/pull-db-tests.yml:193`),
  the unit-test job only adds the hostname (`:155`), and `tests/mysql.ini.tmpl:97` points `HOST = smtpimap`.
  Which test uses that host was not traced.
- CI: `pull-db-tests.yml` runs sqlite, postgres 14, mysql 8.4, mssql jobs, each with `make test-migration`
  and `make test-integration` (`:99-103`, `:209-211`, `:242-244`). E2E: `tools/test-e2e.sh` builds the
  binary, creates an admin via CLI, runs Playwright (`workers: '50%'`, chromium + firefox,
  `playwright.config.ts`); Playwright can run in a container when the host is not Debian/Ubuntu.
- E2E helpers (`tests/e2e/utils.ts`) create state through the app's REST API with `apiRetry` and unique random
  names, not through fixtures.

**Verdict.**

- Borrow, e2e only: dirty-table tracking is worth knowing about if sphericon's e2e ever needs to reset a shared
  database between scenarios. sphericon's e2e uses fresh Workspaces instead (`env.NewWorkspace(t).Ready()`), which
  avoids the reset entirely. **Inference:** keep the fresh-Workspace approach.
- Borrow: `CheckConsistencyFor`-style invariant checks, if sphericon has denormalised counters (not verified
  whether it does).
- Borrow: running the integration suite from a compiled test binary to keep passing-test log noise out of
  output, if sphericon's ent/river logging becomes noisy.
- Skip: hook-parses-SQL-text dirty tracking (fragile), numeric literal ids in tests (sphericon's generated named
  constants are better), per-engine DB matrix (sphericon is Postgres only).

### A2. Forgejo (GPL-3.0-or-later, Go; hosted on Codeberg)

**What it solves.** Community fork of Gitea. **Overlap:** same as Gitea, plus newer test tooling in 2026.

**Test architecture.**

- Layout: as Gitea plus `tests/forgery/` (per-entity factories: `access_token.go`, `issue.go`, `project.go`,
  `repo.go`, `team.go`, `user.go`), `tests/internaltest/`, `tests/e2e/` with both Go files and
  `*.test.e2e.ts` Playwright specs, and `tests/e2e/fixtures/` as an extra fixture dir layered on
  `models/fixtures/`.
- Go-driven e2e: `tests/e2e/e2e_test.go` `TestMain` initialises fixtures with an extra directory
  (`Dirs: []string{"tests/e2e/fixtures/"}`) and registers `PATCH /_e2e/fixtures/reload` on the real router
  (it calls `unittest.LoadFixtures()`). `TestE2e` boots the server with `internaltest.NewInternalTestServer`,
  globs `tests/e2e/*.test.e2e.ts` and shells out to `npx playwright test`
  (`tests/e2e/e2e_test.go:36-99`). So `go test` is the single entry point for e2e too.
- `forgery`: `forgery.CreateAccessToken(t, user, &opts)` creates an entity through the service layer and
  defaults its name to `uniqueSafeName(t.Name())`, so parallel tests never collide
  (`tests/forgery/access_token.go`). Declarative repo setup (`tests/e2e/declare_repos_test.go`,
  `FileChanges` with `Versions []string`) builds git history from a table.
- CI: `.forgejo/workflows/testing.yml`, with frontend build cached for Playwright, per-DB integration jobs,
  remote-cacher matrix (redis, valkey, garnet), coverage merged into a PR comment.

**Verdict.**

- Borrow: one Go entry point for e2e that boots the in-process app then runs the browser suite. sphericon's
  `e2e/` is Go-only today (Mailpit Inbox, no browser); if a Playwright layer is ever added, this wiring is
  the template. **Inference:** only relevant if browser e2e is planned.
- Borrow: a fixture-reload endpoint registered only in the test router, to reset between Playwright specs.
- Borrow, with a conflict: `forgery`'s `t.Name()`-derived unique names solve a real parallelism problem.
  **It conflicts with sphericon's rule that tests build on committed fixtures and do not fabricate entities
  inline** (memory: tests use fixtures; add a fixture row if uncovered). Do not adopt `forgery`-style
  factories unqualified. A defensible subset: factories only for entities whose uniqueness is the point
  (api tokens, sending-domain names), living in `testhelper` and generated or documented as the exception.
  That is a policy decision for the user, not taken here.
- Skip: GPL-licensed code cannot be copied into sphericon's AGPL core without a licence review (not checked);
  read for ideas only.

### A3. rudder-server (Elastic License 2.0 for the core, per repo LICENSE; Go)

**What it solves.** Customer-data-platform ingestion: gateway accepts events, a Postgres-backed jobs queue
(`jobsdb`) moves them through processor and router to destinations. **Overlap:** event ingestion from
customer sites (sphericon `/collect`), Postgres as a queue, workspace-scoped config.

**Test architecture.**

- Libraries: `ory/dockertest/v3`, `testify`, `ginkgo/v2` + `go.uber.org/mock/gomock` in older packages
  (`go.mod:80-106`, `gateway/gateway_test.go:19-25`), and `rudder-go-kit/testhelper/docker/resource/*`
  wrappers (121 files import dockertest).
- DB handling: unit tests start a throwaway Postgres container per test or per package
  (`jobsdb/jobsdb_test.go:2654`: `postgres.Setup(pool, t, postgres.WithTag("17-alpine"))`); no fixtures.
- Config as code: `testhelper/backendconfigtest/` has fluent builders for the workspace configuration the
  server consumes (`SourceBuilder`, `DestinationBuilder`, `ConfigBuilder`, a `ServerBuilder` that returns an
  `httptest.Server`). Test scenarios compose configs instead of loading YAML.
- Black-box integration: `integration_test/docker_test/docker_test.go:81` boots the whole app against real
  containers and a webhook sink, sends events through the gateway, and asserts with `require.Eventually`
  plus direct SQL on the warehouse schema (`:140-190`). It runs the scenario twice (shared vs separate
  connection pools).
- Rules written down: `AGENTS.md:121` says: use `require.Eventually`, never `time.Sleep`, and return a bool only
  inside the callback; `Makefile:29-34` runs `gotestsum` with `-p=1 -failfast -shuffle=on`, race
  optional; `AGENTS.md:125` says a matrix checker in `make fmt` enforces that new warehouse integration
  packages are added to the CI matrix.
- CI: `.github/workflows/tests.yaml` runs the docker flow in an oss/enterprise matrix; warehouse integration
  tests are a separate matrix gated by `SLOW=1`.

**Verdict.**

- Borrow: the `require.Eventually`-not-sleep rule, as a lint or review rule for river/watermill assertions
  (sphericon already has `Outbox*` helpers; whether any test sleeps was not checked).
- Borrow: fluent builders for configuration that has no fixture row (for example provider settings), only if
  YAML fixtures prove awkward. sphericon's catalog is the stated policy, so this is optional.
- Borrow: a CI check that every new package of a class is registered in the matrix (matrixchecker idea).
- Skip: per-test Docker Postgres (sphericon's shared baseline + txdb is far faster; **inference**), ginkgo/gomock
  (sphericon uses testify and a real app), direct SQL assertions (forbidden in sphericon).

### A4. mox (MIT / MPL-2.0, Go)

**What it solves.** Complete modern mail server (SMTP in/out, IMAP, DKIM, SPF, DMARC, MTA-STS, DANE,
queue, DSN bounces, junk filter). **Overlap:** outbound queue with retries and DSN generation, DKIM signing
and verification, suppression list (`queue/suppression.go`).

**Test architecture.**

- Layout: package-local `_test.go` everywhere, `testdata/` per area (`smtp`, `dkim`-adjacent, `queue`,
  `dsn`, `store`), `integration_test.go` behind `//go:build integration` and `testdata/integration/` with
  docker-compose.
- In-process SMTP: `smtpserver/server_test.go` `newTestServer` (`:118`) builds a real server on a temp
  account store; `runRaw` connects client and server with `net.Pipe()` (`:257-275`), no sockets, no ports.
  Tests assert on SMTP reply codes and enhanced status codes with a typed comparison (`smtpErr`, `:280`).
- DNS: `dns.MockResolver` (`dns/mock.go:14`) is a map of A, PTR, TXT, MX records injected into every
  component, so SPF/DKIM/DMARC/MX behaviour is tested with zero network
  (`smtpserver/server_test.go:549-554`).
- DKIM: `dkim/dkim_test.go` has table tests for parse, sign (RSA, Ed25519), verify, body hash
  (`TestParseSignature :91`, `TestSign :205`, `TestVerify :332`).
- Queue and bounces: `queue/queue_test.go` `TestQueue :108` plus hook and suppression tests.
- Fuzzing: `Makefile:74-87` lists native Go fuzz targets for DKIM signature and record parsing, DMARC,
  SMTP, SPF, TLS-RPT, MTA-STS.
- CI: `.github/workflows/build-test.yml` runs `make build`, `make test`, `make fmt`
  (`:25-39`). **The docker-compose integration suite (`make test-integration`, `Makefile:94`) is not run in
  the GitHub workflow I read.** Flags: `-shuffle=on`, `-fullpath`, optional `-race`.

**Verdict.**

- Borrow: native Go fuzz targets for any sphericon parser of untrusted input that is also security-relevant
  (signed unsubscribe/confirm token parsing in `internal/consent`, SES/SNS hook payloads, the `/collect`
  body). Cheap to add. Whether sphericon already has any fuzz target was **not verified**.
- Borrow: table-driven DKIM tests with a mock DNS map, relevant to the known go-mail `h=` folding gotcha
  (memory: DKIM h= folding). A regression table of header lists and expected canonicalisation fits the same
  style.
- Borrow: asserting enhanced status codes (not just 2xx/5xx) when testing bounce handling.
- Skip: `net.Pipe` SMTP server harness (sphericon is an SMTP client; Mailpit already covers the server side).

### A5. maddy (GPL-3.0, Go)

**What it solves.** Composable all-in-one mail server (SMTP, IMAP, DKIM signing/verification, queue).
**Overlap:** outbound queue, DKIM modifier and check modules, DNS-dependent policy.

**Test architecture.**

- Two tiers: unit and module tests in `internal/**` (43 `_test.go` files, with `internal/testutils/`:
  in-memory SMTP server `smtp_server.go`, `table.go`, `target.go`, `check.go`, `modifier.go`), and a
  black-box `tests/` package behind `//go:build integration`.
- Black-box harness `tests/t.go`: `tests.NewT(t)` writes a config to a temp dir, allocates ports, optionally
  starts a `go-mockdns` server (`t.DNS(map[string]mockdns.Zone{...})`), runs the compiled `maddy` binary as a
  subprocess, and exposes `t.Conn("smtp")` with `Writeln` / `ExpectPattern("250 *")`
  (`tests/mta_test.go:34-70`). A second in-process fake SMTP target (`testutils.SMTPServer`) receives
  outbound mail and the test asserts on what arrived.
- Coverage: the integration binary is built with coverage counters and merged (`tests/run.sh`,
  `tests/build_cover.sh`).
- CI: `.github/workflows/test.yml` runs `go test ./...` then `cd tests/ && ./run.sh`
  (verified, run lines near `:55-65`).

**Verdict.**

- Borrow: `go-mockdns` for sending-domain verification tests (SPF/DKIM/DMARC TXT lookups) if sphericon's
  resolver (`internal/messaging/resolver_test.go` exists) is not already injectable; whether it is was not
  checked.
- Skip: subprocess-binary harness (sphericon's in-process app is simpler and faster), GPL code.

### A6. Hatchet (MIT, Go; Postgres-backed task queue, multi-tenant)

**What it solves.** Durable task and workflow engine on Postgres (plus optional RabbitMQ). **Overlap:**
Postgres as the queue, multi-tenant (tenant = workspace), a scheduler with concurrency limits, EE-ish gating.

**Test architecture.**

- Build tags split tiers: default unit tests carry `//go:build !e2e && !load && !rampup && !integration`
  (e.g. `pkg/repository/dag_orchestrator_test.go:1`); `integration` (12 files) needs the compose Postgres;
  `e2e`, `load`, `rampup` are heavier. CI runs `go test` then `go test -tags integration`
  (`.github/workflows/test.yml:99`, `:137`; `Taskfile.yaml:361-372`).
- Integration DB: `internal/testutils/env.go:20` `Prepare` points at a **shared dev Postgres** from
  `docker compose up -d` with a hard-coded tenant UUID, creating the tenant if missing and minting a token.
  `RunTestWithDatabase` wraps a test body; tests isolate by random emails and ids
  (`pkg/repository/user_integration_test.go:25-45`), not by rollback.
- Full-engine harness: `pkg/testing/harness/engine.go` uses testcontainers for Postgres (and RabbitMQ,
  PgBouncer), runs real migrations, seeds, starts the engine in-process, and is parameterised by env
  (`TESTING_MATRIX_MIGRATE` latest or `penultimate`, `TESTING_MATRIX_PG_VERSION`,
  `TESTING_MATRIX_RABBITMQ_ENABLED`, optimistic scheduling): `:34-60`. `penultimate` runs
  `migrate.RunMigrations(ctx, migrate.WithUpToPenultimate())` before the suite (`:180-185`), so a test can
  exercise the last migration's effect on existing state.
- Leak detection: `TestMain` runs `goleak.Find(...)` with an explicit ignore list after a successful run
  (`:73-95`).
- CI also runs the load suite in a `race x optimistic-scheduling` matrix (`test.yml:421-428`), and
  `AGENTS.md:1-5` forbids CI from phoning home.

**Verdict.**

- Borrow: `goleak.Find` in `TestMain` with a curated ignore list. sphericon runs river and watermill workers;
  leaked goroutines after the in-process app stops are a real class of bug. **Inference:** worth a trial in
  `e2e/main_test.go` first.
- Borrow, optional: a "migrate to N-1, load fixtures, migrate to N" test, to check the newest migration
  against pre-existing rows. sphericon's `migrate_test.go` covers only the empty-database path. Hatchet gets the
  N-1 state by running the whole suite after `WithUpToPenultimate()`; sphericon could do the narrower version
  with `ScratchDatabaseURL` and goose's provider.
- Borrow: build tags as tiers is already effectively sphericon's `e2e` tag.
- Skip: the shared dev DB with a hard-coded tenant id (the opposite of txdb isolation), per-suite container
  start (sphericon uses a shared Postgres daemon).

### A7. Grafana (AGPL-3.0, Go)

**What it solves.** Observability dashboards and alerting. **Overlap:** very large Go monolith with a
mature test-DB story, multi-org tenancy, an enterprise split.

**Test architecture.**

- Naming contract: integration tests must be named `TestIntegration*`; `testutil.SkipIntegrationTestInShortMode`
  fails the test if the name lacks the prefix and skips under `-short`
  (`pkg/util/testutil/testutil.go:36-44`). `make test-go-unit` runs `-short`; `make test-go-integration`
  runs `-run "^TestIntegration"` over only the packages that contain such tests
  (`Makefile:547-565`). Both are sharded by a script (`scripts/ci/backend-tests/shard.sh`).
- DB matrix by env: `GRAFANA_TEST_DB=postgres|mysql|sqlite` (`pkg/services/sqlstore/sqlutil/sqlutil.go:33-41`,
  `Makefile:582-595`).
- DB isolation, old to new: `InitTestDB` (shared DB, truncate between tests) is now `Deprecated` and calls
  to it fail lint (`pkg/infra/db/db.go:87-95`). `NewTestStore` creates a unique temporary database per test,
  dropped on cleanup, safe for `t.Parallel()`, and with `GRAFANA_TEST_DB_TEMPLATE=true` clones from a
  pre-migrated template keyed by a fingerprint of config and migrations
  (`pkg/services/sqlstore/sqlstore_testinfra.go:231-272`).
- Server-level: `pkg/tests/testinfra` `StartGrafanaEnv` boots the whole server in-process on a free port
  with a generated config dir and returns an HTTP address plus the store, used by `pkg/tests/api/*`.
- CI: `backend-unit-tests.yml` runs sharded unit jobs for OSS and Enterprise.

**Verdict.**

- Borrow: the `TestIntegration` prefix guard with a `-short` skip, if sphericon wants a fast `-short` lane
  alongside txdb tests. sphericon has no such split today (**not verified** beyond AGENTS.md).
- Borrow: the lint-enforced deprecation of the old test-setup path, as a pattern: when sphericon changes a
  testhelper, deprecate the old one so staticcheck blocks new uses. Fits "never disable lint rules".
- Confirmed, not borrowed: per-test temporary DB from a migrated template is the heavier version of what
  txdb gives sphericon for free. Template-cloning is the only relevant idea, and only for tests that must commit
  (e2e, tests with `WithinScopedTx` and concurrent connections).
- Skip: sharding scripts (sphericon's suite is small enough to run `go test -p 1`), the DB matrix.

### A8. Mattermost (mixed licensing, Go)

**What it solves.** Team chat. Licensing is mixed (`LICENSE.txt` is a licensing overview; `server/enterprise/`
has its own licence); not analysed further. **Overlap:** open-core with `enterprise/`, license-gated
features, a single binary, an API test helper, Inbucket-style mail capture.

**Test architecture.**

- Store contract tests: `server/channels/store/storetest/` (67 files) hold backend-agnostic test functions of
  type `func(*testing.T, request.CTX, store.Store, storetest.SqlStore)`; each `sqlstore/*_store_test.go`
  just binds one, for example `StoreTestWithSqlStore(t, storetest.TestUserStore)`
  (`sqlstore/user_store_test.go:13`). The wrapper loops over the configured store types, runs each as a
  subtest and skips under `-short` (`sqlstore/store_test.go:128-152`). The same contract is reused for the
  search-layer stores (`StoreTestWithSearchTestEngine`).
- API tests: `api4` `Setup(tb)` returns a `TestHelper` with a live server, logged-in client and a clean
  store: `dbStore.DropAllTables()`, then preload migrations from a SQL warmup file instead of re-running
  them (`server/channels/api4/apitestlib.go:267-286`, `testlib/helper.go:267-284`). With
  `ENABLE_FULLY_PARALLEL_TESTS=true` it takes stores from a pool (`GetNewStores`,
  `testlib/helper.go:161-186`). `SetupEnterprise` is the licensed variant (`apitestlib.go:288`).
- Mail: tests read real delivered mail from Inbucket over its HTTP API through a client in non-test code,
  `server/platform/shared/mail/inbucket.go` (`GetMailBox`, `GetMessageFromMailbox`, `DeleteMailBox`,
  `RetryInbucket`), used from `api4/user_test.go`, `api4/team_test.go`, `app/email/email_test.go`
  (verified by grep). Inbucket is a service in `server/docker-compose.yaml`.
- Browser e2e lives in `e2e-tests/` (Cypress and Playwright), separate from Go tests.
- CI: `IS_CI=true` and `MM_SQLSETTINGS_DRIVERNAME` select the DB per job (`sqlstore/store_test.go:163-168`).

**Verdict.**

- Borrow (strongest idea in this note): contract tests written once against an interface and bound to each
  implementation. sphericon has smtp and ses providers behind `messaging` and shares `messaging.BuildMIME`;
  a shared `providertest.Run(t, provider)` suite would assert identical behaviour (headers, errors, retry
  classification) on both. Whether sphericon already does this was not checked (`internal/messaging` has
  `signer_test.go`, `mime_test.go`, `resolver_test.go`, `coverage_test.go`).
- Borrow: `SetupEnterprise` vs `Setup`, which mirrors sphericon's `testhelper.Setup` and `WithoutLicense()`.
  Already equivalent.
- Borrow: the Inbucket-client-in-non-test-code shape with a `RetryInbucket` helper, which is what sphericon's
  `e2e/inbox.go` and `mailpit.go` already are.
- Skip: `DropAllTables` per test plus warmup SQL (txdb rollback is better and sphericon builds schema with ent),
  the fully parallel store pool.

## B. Non-Go product references (lighter touch)

### B1. Dittofeed (MIT `LICENSE` at repo root, TypeScript; `docker-compose.ee.yaml` suggests an EE tier, not examined)

**Solves.** Open-source customer-engagement / marketing automation (journeys, broadcasts, segments,
messaging) on Postgres, ClickHouse and Temporal. **Overlap:** the closest product match to sphericon.

**Tests.** Jest with ts-jest, projects per package (`jest.config.js`); backend-lib has 63 `*.test.ts` files
beside the source. Integration-style: `globalSetup` migrates Postgres via Drizzle and bootstraps ClickHouse
once (`packages/backend-lib/test/globalSetup.ts`). **No per-test reset:** each test creates a brand-new
workspace with a random name (`createWorkspace({ id: randomUUID(), name: \`test-\${randomUUID()}\` })`,
`src/subscriptionManagementEndToEnd.test.ts:50-58`) and builds its own templates, user properties and a
built-in `Test` email provider type. CI shards the suite and gives each shard its own DB names to avoid
races (`.github/workflows/shared-workflow.yaml`, `test_shards`). Factories live in `test/factories/`.

**Verdict.** Borrow: a first-class test provider type is the same idea as sphericon's `fakeses.go` and Mailpit;
nothing new. The "fresh workspace per test, no teardown" isolation is exactly what sphericon's e2e does with
`env.NewWorkspace(t)`, which confirms the approach for tests that commit. Skip: random-name inline
construction for Go unit tests (conflicts with fixtures policy), Temporal-specific harness.

### B2. Postal (MIT, Ruby on Rails)

**Solves.** Self-hosted outbound and inbound mail delivery platform (SMTP server, queue, DKIM, tracking,
webhooks). **Overlap:** outbound pipeline, DKIM, tracking, route and endpoint model.

**Tests.** RSpec (55 spec files) under `spec/`: `lib/` (`dkim_header_spec`, `message_dequeuer/*`,
`smtp_server/`, `smtp_client/`, `tracking_middleware_spec`), `models/`, `requests/`, `apis/`. FactoryBot
factories per model; `config.use_transactional_fixtures = true`; `ActionMailer` delivery method `:test`;
WebMock to block network; Timecop for time. **Notable:** `before(:suite)` runs `FactoryBot.lint`, which
builds every factory and fails the suite if any is invalid (`spec/rails_helper.rb:32-53`). CI runs rspec
inside docker compose (`.github/workflows/ci.yml:55-58`).

**Verdict.** Borrow: a "lint the fixtures" test, i.e. one test that loads or touches every generated catalog
constant and validates each row against the ent schema and API validators, so a stale fixture row fails
fast instead of failing a distant test. sphericon's generator already checks naming; row validity beyond DB
constraints was not checked. Skip: everything else.

### B3. Mautic (GPL, "either version 3" per `LICENSE.txt:7`; PHP/Symfony)

**Solves.** The most established open-source marketing automation platform (contacts, segments, campaigns,
email). **Overlap:** domain match.

**Tests.** PHPUnit unit and functional tests inside each bundle (`app/bundles/*/Tests/`, about 1232 test
files) plus Codeception acceptance specs (`tests/acceptance/*Cest.php`, 10 files). Functional base class
`MauticMysqlTestCase`: transaction rollback per test, with a documented fallback when a test runs schema
DDL; and, when rollback is not possible, a reset from a SQL dump regenerated with `mysqldump`
(`app/bundles/CoreBundle/Test/MauticMysqlTestCase.php:24`, `:190-266`). Mailer DSN is `null://null` and the
messenger transport `in-memory://default` in tests (`AbstractMauticTestCase.php:57-60`). CI matrix is PHP
version by MySQL/MariaDB version with the DB on tmpfs (`.github/workflows/tests.yml:23-57`).

**Verdict.** Skip mostly. Borrow only the documented rule "rollback by default, explicit opt-out with a
reason when the test needs DDL or commits", which is how sphericon's txdb vs scratch-DB split already works.

### B4. Plausible Analytics (AGPL-3.0, Elixir)

**Solves.** Privacy-friendly web analytics: a tracker script, an ingestion endpoint, ClickHouse events,
Postgres metadata, teams and sites. **Overlap:** tracker snippet plus `/collect`, open-core (CE vs EE),
workspace-like teams.

**Tests.** ExUnit under `test/` (and a guard: `test_helper.exs:1-4` raises if any `_test.exs` is found under
`lib/`). Tracker has its own Playwright specs in `tracker/test/*.spec.ts` and `e2e/`. DB: Ecto SQL sandbox
(`Sandbox.mode(Plausible.Repo, :manual)`), each test in a rolled-back transaction, with
`Plausible.Test.Support.Sandbox.allow_salts_process` letting a background process share the test's
connection. Factories via ExMachina (`test/support/factory.ex`). External seams are `Mox` mocks
(`HTTPClient.Mock`, `DnsLookup.Mock`). Modes: `ce_test`, `e2e_test` and EE mix envs select tag excludes
(`:ee_only`, `:e2e`, `:slow`, `:minio`, `:migrations`) in `test_helper.exs`. CI runs the suite across mix
envs with Postgres 18 and ClickHouse, `--include slow --include migrations`, 6 partitions
(`.github/workflows/elixir.yml:21-39`, `:101-108`).

**Verdict.**

- Borrow: running the full suite in both core-only and licensed modes in CI, with tag-based excludes for
  edition-only tests. sphericon has `WithoutLicense()`; whether CI runs both modes for the whole suite was not
  checked.
- Borrow: a guard test that fails if tests exist in an unexpected directory (cheap structural check).
- Borrow: the explicit "allow this background process into the test's transaction" step maps to sphericon's
  documented `env.DeliverToEE(t)` step (the router does not run under txdb); same idea, already done.
- Borrow: tracker tests in a real browser (Playwright) for `/t.js`; sphericon's `@sphericon/analytics` package has
  its own tests (not inspected here).
- Skip: Elixir-specific machinery.

## C. Already reviewed

### knadh/listmonk (AGPL-3.0, Go): nothing worth taking

Re-confirmed at `70ae516`: `tests/` contains Playwright specs only, 9 files in `tests/specs/`; there are no
Go `_test.go` files anywhere in the repo (count 0). `tests/helpers.js:62` shells out to `psql` for state
changes (`UPDATE settings SET value = ...`, `:74`, `:85`); mail is read from MailHog at `localhost:8025`
(`helpers.js:6`, `playwright.config.js:5`); teardown is `pkill -9 listmonk` (`global-teardown.js:4`,
`helpers.js:94`); `resetDB` runs between specs (`helpers.js:93`) and `workers: 1`
(`playwright.config.js:9`). Everything sphericon's stack already does better (in-process app, ent-only state,
fixtures, parallel-safe Workspaces).

## What sphericon already does that these projects confirm

- One baseline plus per-test rollback (txdb) is stronger than every DB strategy found except Grafana's
  template-cloned temp DBs, and cheaper than all of them. Gitea reloads tables, Mattermost drops all tables,
  Hatchet and Dittofeed rely on random ids in a shared DB, Mautic rolls back like sphericon.
- Fresh tenant per scenario for tests that commit (Dittofeed, Hatchet) equals `env.NewWorkspace(t)`.
- Capturing mail in a real local SMTP sink with an HTTP read API (Mattermost Inbucket, Gitea smtpimap,
  listmonk MailHog) equals the Mailpit `Inbox`.
- Edition split in tests (Mattermost `SetupEnterprise`, Plausible `ce_test`) equals `WithoutLicense()`.

## Unknown / not verified

- No suite was executed; nothing here measures speed, flakiness or coverage.
- Licences of copied-from code were not reviewed beyond the headers named above; Forgejo and maddy are GPL
  and rudder-server is not OSI open source, so read for ideas only. The rudder-server licence name is from
  the repo `LICENSE` header and was not re-read in full.
- Whether sphericon already has a shared provider contract suite, fuzz targets, an `-short` lane, a sleep ban,
  or CI jobs that run Atlas migrations was not checked beyond the greps stated above.
- The Mattermost "uses Inbucket in CI" statement rests on `server/docker-compose.yaml` and the Go client, not on a CI workflow. Gitea's and Mattermost's CI job layouts beyond the lines cited were not read; Mattermost's Cypress and
  Playwright suites were not opened.
- Dittofeed, Postal, Mautic and Plausible were read at lighter depth (layout, base classes, CI matrix).

## Peek at first (ranked by how many sphericon mechanisms each touches)

1. **Forgejo**: Go-driven e2e with fixture-reload endpoint, `forgery` (and why it conflicts with the
   fixtures policy), per-test unique naming, in-process boot.
2. **Mattermost**: `storetest` contract tests (maps to smtp/ses providers), `Setup` vs `SetupEnterprise`,
   Inbucket mail client.
3. **Plausible**: CE vs EE modes in CI, tag excludes, sandbox allowance for background processes, the
   `lib/` test-location guard, tracker Playwright tests.
4. **Hatchet**: `goleak` in `TestMain`, penultimate-migration matrix, build-tag tiers, multi-tenant harness.
5. **mox**: DKIM and bounce tests with `dns.MockResolver`, enhanced-status assertions, fuzz targets.
6. **Gitea**: dirty-table fixture reload, `CheckConsistencyFor`, compiled integration binary.
7. **Grafana**: `TestIntegration` prefix guard, lint-enforced deprecation of old test setup, template-cloned DB.
8. **rudder-server**: `require.Eventually` rule, config builders, matrix checker.
9. **maddy**: `go-mockdns` for sending-domain tests; subprocess harness is a skip.
10. Dittofeed, Postal (`FactoryBot.lint` idea), Mautic: read only the cited files. listmonk: skip.
