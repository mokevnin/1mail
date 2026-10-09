# Quality-Control Tooling Options (what to add next, and is ast-grep one of them?)

Researched 2026-10-08 against primary sources only (each tool's docs, GitHub README or
releases page). Every factual claim carries its source URL. Anything not confirmed on a
primary page fetched during this research is marked **unverified**; "release dates" quoted
from GitHub release listings are as the listing showed them. Repo facts come from reading
this repo (`.golangci.yml`, `.oxlintrc.json`, `Makefile`, `.github/workflows/ci.yml`,
`hk.pkl`, `.mise.toml`, `knip.json`, `.gitleaks.toml`, `.github/dependabot.yml`).

**Short answer:** yes, ast-grep is worth adding, but as a _narrow_ tool for a handful of
repo-specific structural rules (about five, three of which already have real violations in
the tree). It is not the best tool for most of what the owner listed: oxlint and
golangci-lint already cover more of it than expected, and the biggest quality gap in this
repo is not a linter at all but a **generated-code drift check** in CI. Section 4 has the
prioritized shortlist.

---

## 1. Baseline: what is already installed (not re-recommended)

- Frontend: `tsc` (TS 7), oxlint (type-aware, jsx-a11y, react, import, vitest plugins),
  oxfmt, knip, `check-css` (fails on any tracked `.css`), `i18next-cli extract --ci` and
  `types --ci`.
- Backend: golangci-lint with `.golangci.yml` = `version: '2'`, `default: standard` and
  **nothing else enabled**; `govulncheck`.
- Security/CI hygiene: gitleaks, jactionlint, zizmor, pinact (SHA-pinned actions), Dependabot
  (npm, gomod, actions, docker, docker-compose, 7-day cooldown), hk git hooks, release-please,
  goreleaser.
- Tests: `go test -p 1 ./...` against Postgres with txdb + YAML fixtures; Vitest Browser Mode
  (Playwright Chromium) in a separate CI job.
- Not present anywhere: generated-code drift check, migration linting, OpenAPI breaking-change
  detection, CodeQL / dependency review / Scorecard, Dockerfile lint, spell/link check,
  commit-message lint, coverage reporting, bundle-size budgets.

The repository is **public** (`gh repo view`: `visibility: PUBLIC`), which matters because
several GitHub-native tools are free only for public repos (see section 3.4).

---

## 2. ast-grep

### 2.1 What it is and how it is configured

- Structural (AST) search/lint/rewrite tool built on tree-sitter. Rules are YAML files with
  `id`, `language`, and a `rule` object. Atomic rules: `pattern` (code-by-example with
  `$META` variables), `kind` (tree-sitter node kind), `regex`. Relational rules: `inside`,
  `has`, `follows`, `precedes` (with `stopBy`, `field`). Composite: `all`, `any`, `not`,
  `matches` (utility rules).
  Source: https://ast-grep.github.io/guide/rule-config.html
- Optional rule fields: `severity` (`hint`/`info`/`warning`/`error`/`off`), `message`,
  `note`, `fix`, `files` (globs), `ignores` (globs, checked before `files`), `constraints`,
  `transform`, `url`, `labels`.
  Source: https://ast-grep.github.io/reference/yaml.html
- Project config `sgconfig.yml`: `ruleDirs` (required), `testConfigs` (`testDir`,
  `snapshotDir`), `utilDirs`, `languageGlobs`, `customLanguages`, `languageInjections`
  (experimental).
  Source: https://ast-grep.github.io/reference/sgconfig.html
- Testing: `ast-grep test` runs per-rule test files with `valid:` and `invalid:` code
  snippets (plus optional snapshots, `-U` to update). A rule is classified as validated,
  reported, noisy (fires on valid code) or missing (silent on invalid code).
  Source: https://ast-grep.github.io/guide/test-rule.html
- `ast-grep scan` runs the rules of the project; (`scan` itself was not on a fetched page, so
  its flags are **unverified** here, but the GitHub Action below runs it with the project
  config).
- Languages: built-in Go (`go`, `golang`; `.go`), TypeScript (`ts`; `.ts/.cts/.mts`) and Tsx
  (`tsx`; `.tsx`).
  Source: https://ast-grep.github.io/reference/languages.html
- GitHub Action `ast-grep/action`: inputs `version` (default latest), `config` (default
  `sgconfig.yml`), `paths`. The README example uses `@latest`, which this repo's pinact/zizmor
  policy would reject; pin to a SHA instead.
  Source: https://github.com/ast-grep/action
- Maintenance: releases page lists 0.45.3 as latest (31 Aug), then 0.45.2, 0.45.1, 0.45.0,
  0.44.x ... roughly a release every 2-4 weeks; the page does not show the year (the cadence is
  consistent with 2026, but that is **unverified**). Still 0.x, so rule syntax can move.
  Source: https://github.com/ast-grep/ast-grep/releases
- Install route (**unverified**, not read from the install page): npm `@ast-grep/cli`, or a
  mise tool. Because `check-security` already runs mise-installed tools on the host, ast-grep
  fits that same pattern (add to `.mise.toml`), which satisfies "runs natively in CI".

### 2.2 Honest overlap analysis

Tree-sitter is syntactic: ast-grep has no type information (**unverified** against a primary
page; it follows from the tree-sitter design). So it cannot say "this `Create()` is on an ent
client", only "this call has this shape".

| Wished-for rule                                 | Already covered / best tool                                                                                                                                                                                                                                                                                                                                            | Verdict                                                                                              |
| ----------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| No inline `style={{}}` / `className`            | **Already enforced** by oxlint `react/forbid-dom-props`, `forbid-component-props`, `forbid-elements` in `.oxlintrc.json`.                                                                                                                                                                                                                                              | Do not duplicate.                                                                                    |
| No raw `fetch` in the frontend                  | oxlint ships `no-restricted-globals` natively (rule list: https://oxc.rs/docs/guide/usage/linter/rules.html). Prefer that. Note: 3 real hits today (`src/components/UserMenu.tsx`, `src/routes/confirm.tsx`, `src/routes/unsubscribe.tsx`), all calling non-`/site` endpoints (`/auth/logout`, `/e/confirm/…`, `/e/u/…`); decide whether those belong in the contract. | oxlint, not ast-grep.                                                                                |
| No `samber/mo`                                  | golangci-lint `depguard` (deny list by package regex with a message) or `gomodguard`. `depguard` config shape: https://raw.githubusercontent.com/golangci/golangci-lint/main/.golangci.reference.yml. Currently no hit (`git grep` found none).                                                                                                                        | golangci-lint `depguard`, not ast-grep.                                                              |
| No hardcoded path strings in navigation         | oxlint has no `no-restricted-syntax` in its rule list (it sits between `no-restricted-properties` and `no-restricted-types` alphabetically and is absent; https://oxc.rs/docs/guide/usage/linter/rules.html, last ~8k chars of the page not read). oxlint JS plugins could do it but are **alpha** and cannot use type info (below).                                   | **ast-grep is the better tool** (selector on `to:` props / `navigate({ to })` with a string literal) |
| No hardcoded colors in TSX                      | Nothing native. `.oxlintrc.json` has no color rule.                                                                                                                                                                                                                                                                                                                    | **ast-grep** (`regex` on string literals / JSX attribute values like `c="#fff"`, `rgb(`)             |
| No `client.X.Create()` in tests (use fixtures)  | golangci `forbidigo` matches identifiers by regex and can use type info with `analyze-types`; `gocritic`'s ruleguard can express it (settings in the reference file above), but both need per-path exclusion plumbing. 16 `_test.go` files contain `.Create()` today (`git grep -lE "\.Create\(\)" -- '*_test.go'`); some are legitimate "incidental one-off" cases.   | **ast-grep** for a _warning-level_ ratchet; the fixture policy is a convention, not an invariant.    |
| No hand-edited generated files                  | Not a syntax problem. Handled by regeneration drift check (section 4, item 1), plus `Code generated … DO NOT EDIT` headers that `golangci-lint` already honors for Go.                                                                                                                                                                                                 | Not ast-grep.                                                                                        |
| Enforce `workspace` scoping on external queries | Needs type info and data flow (ent query builders). Syntactic matching gives false positives.                                                                                                                                                                                                                                                                          | Neither; keep as test coverage (cross-workspace isolation tests).                                    |

oxlint JS plugins, for completeness: **alpha**, ESLint v9+ API targeted, rules via
`jsPlugins` in `.oxlintrc.json`; explicitly unsupported: custom parsers/file formats and
**rules that depend on TypeScript type information**. Conformance-tested plugins include
`sonarjs`, `regexp`, `testing-library`, `playwright`.
Source: https://oxc.rs/docs/guide/usage/linter/js-plugins.html
Writing a custom JS rule would also add a Node-side authoring surface to maintain; ast-grep
keeps rules as ~10-line YAML with built-in tests.

The repo rule "never disable a lint rule or use ignore comments" fits ast-grep's `ignores:`
(path globs in config, e.g. `src/generated/**`), which is config-level scoping, not
inline suppression. ast-grep also has an inline suppression comment (**unverified**); simply
do not use it.

### 2.3 Sketch (starting point, not validated; validate with `ast-grep test` before adopting)

```yaml
# sgconfig.yml
ruleDirs: [ast-grep/rules]
testConfigs:
  - testDir: ast-grep/tests
```

```yaml
# ast-grep/rules/no-path-literal-navigation.yml
id: no-path-literal-navigation
language: tsx
severity: error
message: Navigate via route references, not path string literals.
files: ['src/**']
ignores: ['src/generated/**', 'src/**/*.test.tsx']
rule:
  kind: pair
  has: { field: key, regex: '^to$' }
  all:
    - has: { field: value, kind: string }
```

```yaml
# ast-grep/rules/no-direct-ent-create-in-tests.yml
id: no-direct-ent-create-in-tests
language: go
severity: warning # ratchet: existing 16 files; promote to error after cleanup
message: Build scenarios on committed fixtures; add a fixture row instead of client.X.Create().
files: ['internal/**/*_test.go']
rule:
  pattern: $C.$E.Create()
```

Existing tests in `src/**/*.test.tsx` assert `to: '/login'` on a mock; the test files are
excluded above for that reason. Rules to add the same way: hardcoded color regex in `tsx`;
`href="/…"` / `to="/…"` JSX attributes.

**Recommendation: adopt, scoped to 3-5 rules.** Cost: one binary, sub-second scan, one
`make check-ast` target. Risk: 0.x syntax churn; keep rules few and tested.

---

## 3. Other candidates, by area

Verdict key: **adopt** / **maybe** / **skip**.

### 3.1 Go

| Tool                                                        | What it catches                                                                                                                                                                                                  | Fit / cost                                                                                                                                                                                                                                                                                                                                             | Verdict                                                  |
| ----------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------- |
| More golangci-lint linters (`default: standard` only today) | `gosec` (security), `errorlint` (error wrapping), `sqlclosecheck` / `rowserrcheck` (leaked DB rows, relevant to raw SQL beside ent), `bodyclose`, `noctx`, `contextcheck`, `depguard`, `forbidigo`, `usetesting` | Zero new tooling; same `make check-be`. Current golangci-lint is v2.14.0 (24 Sep 2026): https://github.com/golangci/golangci-lint/releases . Linter descriptions page could not be fetched in this session, so per-linter behavior is **unverified**; read each at https://golangci-lint.run/docs/linters/ before enabling. Expect a one-time cleanup. | **adopt** (enable in small batches, fix, never `nolint`) |
| `testifylint`, `paralleltest`, `thelper`, `tparallel`       | testify misuse; missing `t.Parallel`                                                                                                                                                                             | Tests run `-p 1` inside a shared txdb transaction harness; parallelism lints would fight the design. `testifylint` only if testify is used (not checked).                                                                                                                                                                                              | **skip** paralleltest/tparallel; **maybe** testifylint   |
| `go mod tidy -diff` / `go mod verify` in CI                 | untidy `go.mod`/`go.sum`                                                                                                                                                                                         | One line. (`-diff` flag is **unverified** from a primary page in this session.)                                                                                                                                                                                                                                                                        | **adopt** (trivial)                                      |
| `goleak` (uber-go)                                          | goroutine leaks via `goleak.VerifyTestMain(m)`; v1, semver-strict; supports only the two latest Go minors. https://github.com/uber-go/goleak                                                                     | Project runs river, watermill and a pubsub router in goroutines; leaks there are plausible. Needs a `TestMain` per package; interplay with the shared-process fixture loader needs care.                                                                                                                                                               | **maybe** (try in `internal/pubsub`, `internal/jobs`)    |
| Coverage reporting (`go test -coverprofile`) and gotestsum  | visibility only                                                                                                                                                                                                  | Cheap as an artifact/summary; a hard gate invites test-gaming. Hosted coverage services (Codecov etc.) were not evaluated from primary sources.                                                                                                                                                                                                        | **maybe**, report-only; **skip** gating                  |
| `gremlins` mutation testing                                 | weak assertions                                                                                                                                                                                                  | 0.x with config flags that "can change among minor releases"; README says it "doesn't work very well on very big Go modules" (runs can take hours); every mutant would need the Postgres-backed suite. https://github.com/go-gremlins/gremlins                                                                                                         | **skip** for now                                         |
| Native fuzzing (`go test -fuzz`)                            | panics/invariants in parsers                                                                                                                                                                                     | No new dependency; targeted use on MIME/DKIM header building, webhook signature verification, segment filter parsing. Not a CI gate (corpus replay only runs in `go test`).                                                                                                                                                                            | **maybe**, ad hoc                                        |

### 3.2 Database and migrations

- **`atlas migrate lint`: skip.** Atlas docs state that starting with v0.38 the command "is
  available only to Atlas Pro users" and requires `atlas login`; enforcing with `force` is Pro
  too. The repo uses `atlas-community` (`.mise.toml`).
  Source: https://atlasgo.io/versioned/lint
- **squawk (`sbdchd/squawk`): adopt.** Linter for Postgres migrations aimed at downtime
  hazards. Rules named on its page: `require-concurrent-index-creation`,
  `constraint-missing-not-valid`, `ban-drop-column`, `adding-field-with-default`,
  `renaming-column`, `prefer-bigint-over-int`, `prefer-identity`, `prefer-text-field`.
  Config in `.squawk.toml`; GitHub Action `sbdchd/squawk-action`; about 1.2k stars, 1,157
  commits, 42 open issues. The page does not state the current version (**unverified**).
  Source: https://github.com/sbdchd/squawk
  Fit: `migrations/*.sql` are plain Postgres SQL produced by `atlas migrate diff`; squawk
  would run on the _new_ files in a PR (or the whole dir with an allowlist for pre-release
  migrations already applied). Caveat: Atlas-generated migrations will trip rules like
  `require-concurrent-index-creation`; with the "no ignore comments" rule the fix must be a
  config-level rule selection in `.squawk.toml`, decided deliberately. Pre-launch the cost of
  getting this wrong is low; value rises sharply once real tenants exist.

### 3.3 API contracts

- **oasdiff: adopt (for `external` and `collect` only).** Compares two OpenAPI specs and
  detects breaking changes; supports OpenAPI 3.0 and 3.1; Apache-2.0; separate GitHub Action
  repo; `checks` command lists rules. Install example pins 1.11.7 (example only, current
  version **unverified**). https://github.com/oasdiff/oasdiff
  Fit: `openapi/*.openapi.json` are committed, so `oasdiff breaking origin/main:openapi/external.openapi.json openapi/external.openapi.json`
  needs no generation. `/api/*` is public to customers and `/collect/*` is hit by tracker
  snippets already deployed on customer sites, so breaking either is a real incident; `/site/*`
  ships with the SPA in the same binary and does not need it.
- **Spectral / vacuum / Redocly: skip.** vacuum is a fast Spectral-compatible linter for
  OpenAPI 3.0/3.1/3.2, MIT (https://github.com/daveshanley/vacuum), but the OpenAPI here is
  _generated_ from TypeSpec, so style defects are fixed at the TypeSpec source. TypeSpec has
  a linter framework (`defineLinter`, `createRule`, rulesets; the HTTP library has a rule
  `op-reference-container-route`), but how to enable it was not on the fetched page, so any
  claim beyond that is **unverified** (https://typespec.io/docs/extending-typespec/linters/).
  Revisit only if the API surface grows.

### 3.4 Security and supply chain (including hosted/GitHub-native)

| Tool                              | What it gives                                                                                                                                                                                                                                                   | Notes                                                                                                                                                                                                       | Verdict                                                          |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| **CodeQL** (GitHub code scanning) | SAST; supports Go and JavaScript/TypeScript (`javascript-typescript`); "default setup" auto-chooses languages, suite and triggers. https://docs.github.com/en/code-security/code-scanning/introduction-to-code-scanning/about-code-scanning-with-codeql         | Pricing was not on that page (**unverified** here; public repos are known to be eligible, confirm in repo settings). Runs on GitHub's runners, not in `make check`; enable default setup, no workflow file. | **adopt**                                                        |
| **dependency-review-action**      | Fails PRs that introduce vulnerable dependencies (`fail-on-severity`) or disallowed licenses (`allow-licenses`, SPDX list). Available for public repos (private needs GHAS). v5 requires runner >= 2.327.1. https://github.com/actions/dependency-review-action | One small workflow, PR-only, covers npm + Go in one place; `deny-licenses` is deprecated, use `allow-licenses`. Gives the license-policy check that open-core (AGPL core + closed `ee/`) needs.             | **adopt**                                                        |
| **OpenSSF Scorecard action**      | Repo-posture score and badge; "free for all public repositories", runs on `push`/`schedule` on the default branch, publishing needs `id-token: write`. https://github.com/ossf/scorecard-action                                                                 | Mostly measures practices (branch protection, pinned deps, etc.); this repo commits directly to `main`, so expect a lower "Code-Review" score regardless. Useful as a weekly signal, not a gate.            | **maybe** (cheap; add weekly schedule, do not gate)              |
| **OSV-Scanner**                   | Lockfile scanning against OSV.dev, 19+ lockfile types, Go and JS listed, optional call analysis, `--licenses` check. https://github.com/google/osv-scanner                                                                                                      | `govulncheck` already covers Go with reachability; pnpm-lock support was **unverified** on the page. Value is the npm side. Dependabot security alerts already cover npm advisories.                        | **maybe/skip**; revisit if dependency-review proves insufficient |
| **Trivy**                         | Scans images, filesystems, IaC, secrets, licenses, SBOM; Apache-2.0; `aquasecurity/trivy-action`. https://github.com/aquasecurity/trivy                                                                                                                         | Useful for the release image built from `Dockerfile.release`; any action must be SHA-pinned (pinact already in toolchain). Image scan costs a Docker build per run, so schedule weekly/on release.          | **maybe** (release image only)                                   |
| **Semgrep CE**                    | Pattern SAST, 30+ languages; LGPL-2.1; CE analyzes only within a single function or file, Pro rules/engine need login. https://github.com/semgrep/semgrep                                                                                                       | Overlaps CodeQL (SAST) and ast-grep (custom rules); adds login-gated pieces.                                                                                                                                | **skip**                                                         |
| Renovate (vs current Dependabot)  | Hosted Mend app or self-hosted; AGPL-3.0; schedules, monorepo detection, shared presets. https://docs.renovatebot.com/                                                                                                                                          | Comparison with Dependabot features was not on the fetched page (**unverified**). Dependabot with cooldown and groups already works.                                                                        | **skip**                                                         |
| SBOM                              | Trivy can emit SBOMs (above); GitHub's dependency graph provides one for free. goreleaser SBOM support: **unverified**.                                                                                                                                         | Only needed if a customer or the SaaS offering asks.                                                                                                                                                        | **skip** for now                                                 |

### 3.5 Frontend

- **`i18next-cli lint`: adopt (no new dependency).** The CLI the repo already uses has
  `lint`: detects hardcoded user-facing text in JSX not wrapped in `t()`/`<Trans>`,
  `{{placeholder}}` mismatches, string concatenation of translations; exits non-zero only on
  errors, and the docs list no `--ci` flag for it. `status --unused` reports unused keys.
  Source: https://github.com/i18next/i18next-cli
  This directly protects the instance-locale (ru/en/es) feature, which no current check does.
- **size-limit: adopt, but only for the tracker.** `@1mail/analytics` builds `t.js`, an IIFE
  embedded into the binary and loaded on customer sites. A byte budget on
  `packages/analytics/dist/t.js` via its `file` plugin (Brotli default) is exactly what the
  tool is for. It also ships a PR-comment action (`andresz1/size-limit-action`); Vite is not
  mentioned on its page, but the `file` plugin measures any built path.
  Source: https://github.com/ai/size-limit
  For the SPA, a budget is less useful (large deps: xyflow, recharts, workflow builder) and
  would need a stable build step in CI. Not now.
- **Vitest coverage: maybe, report-only.** Providers `v8` (default, works in Chromium) and
  `istanbul`; the thresholds page was not fetched (**unverified**).
  Source: https://vitest.dev/guide/coverage.html
  Only 11 `*.test.ts(x)` files exist; a threshold now would be noise.
- **dependency-cruiser: maybe.** Forbidden-rule import boundaries, TypeScript supported, MIT
  (https://github.com/sverweij/dependency-cruiser). The frontend is small and knip +
  oxlint `import` plugin cover unused/cycle basics; add only if layering (e.g. "routes may not
  import `generated` mutations directly") becomes a recurring review comment. The same rule
  can usually be a 5-line ast-grep rule.
- **Playwright + axe a11y: maybe.** `@axe-core/playwright` provides `analyze()` with tag
  filters (https://github.com/dequelabs/axe-core-npm/tree/develop/packages/playwright).
  oxlint's `jsx-a11y` already covers static issues. Runtime contrast/ARIA checks matter given
  light+dark theming, but the project's rule against disabling rules means the suite must not
  use `disableRules()`; expect to fix real findings in Mantine usage first. Costs a browser
  job; the Vitest browser job already has Chromium.
- **Storybook: skip.** No component-isolation need shown; Mantine components are not custom.

### 3.6 Repo hygiene

- **typos (`crate-ci/typos`): adopt.** Source-code spell checker designed for low false
  positives, "fast enough to run on monorepos", config in `_typos.toml`, GitHub Action and
  pre-commit docs. Docs-heavy repo (ADRs, research notes, i18n JSON in three languages: set
  `[files] extend-exclude` for locale files and `fixtures/`, which is config, not an ignore
  comment). https://github.com/crate-ci/typos
- **commitlint: adopt as a local `commit-msg` hook + CI range check.** Checks
  `type(scope?): subject`; `@commitlint/config-conventional` types: build, chore, ci, docs,
  feat, fix, perf, refactor, revert, style, test; requires Node >= 22.12.0 (repo is on Node
  25). The page documents local (husky) and CI guides but no first-party GitHub Action.
  https://github.com/conventional-changelog/commitlint
  Why it matters here: release-please derives versions and changelog from commits, and
  commits go straight to `main`, so a malformed message silently drops out of the changelog.
  Because there are no PRs, the hook (hk `commit-msg`; hk hook-type support **unverified**) is
  the effective gate; CI backstop on `HEAD~N..HEAD`.
- **lychee: maybe.** Async Rust link checker for Markdown/HTML, `--offline` mode for local
  file links, separate `lychee-action`. https://github.com/lycheeverse/lychee
  `--offline` is a cheap `make check` step for ADR/GLOSSARY cross-links; online mode belongs
  in a weekly scheduled workflow (external URLs flake and would break unrelated pushes).
- **hadolint: adopt.** Dockerfile linter (AST rules + ShellCheck on `RUN`); GPL-3.0 (used as a
  binary, no linkage concern); active (1.4k commits, 12.5k stars). Four Dockerfiles in the
  repo. The page does not describe an official Action (**unverified**); run the binary from
  mise. https://github.com/hadolint/hadolint
- **markdownlint / editorconfig:** oxfmt already formats Markdown; adding markdownlint would
  duplicate and fight it. **skip.** (Not researched from primary sources.)
- **ADR/doc consistency:** no mature tool found; an ast-grep-free option is a tiny `make`
  check that every `docs/adr/NNNN-*.md` is referenced from `GLOSSARY.md` or the ADR index.
  Own suggestion, not a tool; **skip** until drift is observed.

### 3.7 Cross-cutting (brief)

SonarQube/SonarCloud, CodeRabbit and Codecov-style services were **not** evaluated from
primary sources. In general they add review noise and a SaaS dependency to a repo whose
rules are deliberately encoded in-repo (AGENTS.md/CLAUDE.md). **skip** unless a team joins.

---

## 4. Prioritized shortlist

Ordered by value for this repo (codegen-centric, public, direct-to-`main`, pre-launch).

1. **Generated-code drift check** (no third-party tool). `make check-generated`: run
   `make generate-typespec generate-openapi generate-backend`, then
   `git diff --exit-code` (and untracked check). This is the real enforcement of "never
   hand-edit `gen/`, `ent/`, `openapi/`, `src/generated/`, `*_gen.go`", and it catches the
   other common failure: schema or TypeSpec changed but output not committed. Cost: a
   second CI job (needs Go, node, tsp, ogen via `go tool`) that installs the same mise
   toolchain; run only on push to `main`, not in the hk hook. Wire as its own CI job, not
   inside `make check`, to keep `make check` fast.
2. **Widen golangci-lint** (`.golangci.yml`): add `gosec`, `errorlint`, `sqlclosecheck`,
   `rowserrcheck`, `bodyclose`, `noctx`, `contextcheck`, `usetesting`, `depguard` (ban
   `github.com/samber/mo` and other policy imports, per the library-policy memory),
   `forbidigo`. No new tool, already inside `make check-be` and the hk `golangci_lint` step.
3. **ast-grep for 3-5 repo-specific rules** (section 2.3): path-literal navigation,
   hardcoded colors, `client.X.Create()` in tests (warning ratchet). Add `ast-grep` to
   `.mise.toml`, `sgconfig.yml` + `ast-grep/{rules,tests}/`, target
   `check-ast: ast-grep test && ast-grep scan`, append to `check` (next to
   `check-security`, host-run like gitleaks). hk step on `**/*.{ts,tsx,go}` is fine
   (milliseconds). CI: nothing extra, `make check` already runs it.
4. **GitHub-native bundle: CodeQL default setup + `dependency-review-action` (with
   `allow-licenses`) + Scorecard weekly.** CodeQL is repo-settings only; dependency-review
   is a 10-line PR workflow (SHA-pinned, per pinact/zizmor); Scorecard is a scheduled
   workflow that never gates. Not in `make check`, not in hk.
5. **oasdiff on `openapi/external.openapi.json` and `openapi/collect.openapi.json`.**
   `make check-api-compat`: diff against `origin/main` (CI needs `fetch-depth: 0`, already set
   for the lint job). Run on push and compare to the previous commit/tag; in a direct-to-main
   flow the more meaningful base is the **last release tag**, so a deliberate break forces a
   version bump decision. Not in hk.
6. **Cheap hygiene trio:** `i18next-cli lint` (extend `check-i18n`, runs in containers like
   the existing i18n steps), `typos` (add to `check-security`-style host tools via mise;
   hk-eligible, fast), `commitlint` (hk `commit-msg` hook + CI backstop on the pushed range).
   `go mod tidy -diff` rides along in `check-be`.
7. **squawk on migrations.** `make check-migrations` -> `squawk migrations/*.sql` (against
   new files only once the first real release ships); CI via mise. Decide the
   `.squawk.toml` rule set once, up front, because inline ignores are off the table.
8. **size-limit for `t.js`** (and `hadolint` for the four Dockerfiles, same slot: a
   `check-docker` target via mise). Run `make build-tracker` first, so put it in a CI job
   that already builds, not in the hk hook.

Not recommended now: Atlas Pro lint (paid), gremlins, Semgrep, Renovate, vacuum/Spectral,
Storybook, paralleltest. Keep as "maybe" and revisit: goleak, OSV-Scanner, Trivy (release
image), lychee (weekly online), dependency-cruiser, axe a11y, coverage reporting.

### Suggested rollout order

Week 1: items 1 and 2 (largest correctness gain, no new binaries). Week 2: items 3 and 6.
Then 4 and 5 (CI/settings only), then 7 and 8 when migrations and the tracker budget start to
matter. Every step keeps the repo rules intact: nothing here adds CSS, generated dirs are
excluded via `ignores:`/linter exclusions (not comments), and every tool is run through a
`make` target using the existing `RUN_FE`/`RUN_GO` runners or mise-installed host tools.

---

## 5. Open questions / unverified list

- ast-grep: install package/mise name, `scan` flags, inline-suppression comment syntax, and
  the year attached to the 0.45.x releases were not confirmed on a primary page.
- golangci-lint: per-linter semantics and the exact `standard` set were not confirmed on
  the (unfetchable) linters page; `go mod tidy -diff` unconfirmed.
- oasdiff and squawk current versions, squawk-action pinning details, hadolint action
  existence, OSV-Scanner pnpm-lock support, CodeQL pricing for public repos, Vitest
  `thresholds` options, hk `commit-msg` support, goreleaser SBOM, TypeSpec linter enablement:
  all unverified.
- Hosted services (Codecov, SonarQube, CodeRabbit) not researched.

---

## 6. Outcome (2026-10-08)

Adopted and wired (`mise run check` runs the local ones; CI jobs in `.github/workflows/`):

- golangci-lint widened: `bodyclose`, `contextcheck`, `depguard` (bans `samber/mo`, `html/template`), `errorlint`, `noctx`, `rowserrcheck`, `sqlclosecheck`, `usetesting`.
- Generated-code drift check (`generate:check`, CI job `generated`); `openapi-ts` pinned to `@next` for TypeScript 7.
- ast-grep (`sgconfig.yml`, `ast-grep/`): no path-literal navigation, no hardcoded colors.
- typos (`_typos.toml`), `go mod tidy -diff`, `i18next-cli lint`.
- commitlint (hk `commit-msg` hook and CI check on the pushed range), size-limit budget for `t.js` (2 kB gzip, currently 1.17 kB).
- CodeQL, OpenSSF Scorecard and dependency-review workflows.

Evaluated and **not** wired, with the reason:

- `gosec`: 17 findings, mostly false positives (test secrets, admin CLI, DTO int conversions); fixing them needs `nolint` or rule exclusions, both banned. CodeQL covers the security angle.
- ast-grep "no raw fetch": 3 real hits (`src/routes/unsubscribe.tsx`, `src/routes/confirm.tsx`, `src/components/UserMenu.tsx`) because `/e/*` and `/auth/logout` are not in the generated client. Wire the rule once those endpoints are in the TypeSpec contract.
- ast-grep "no `client.X.Create()` in tests": 35 hits on primary entities (Contact 16, Automation 8, Workspace 5, User 4, Broadcast 2); needs a fixture refactor first.
- squawk: 118 findings on Atlas-generated migrations, which are immutable (`atlas.sum`) and cannot be annotated.
- hadolint: DL3018 (pin `apk add` versions) cannot be satisfied without an ignore; revisit if apk pinning is acceptable.
- oasdiff: there is no release tag yet to diff against; add it with the first release.
- `committed` (Rust commit linter): no macOS arm64 binary.
