# Everything runs natively with the toolchain pinned in .mise.toml (`mise install`).
# The dev stack (Postgres, Mailpit, backend, Vite, Caddy) is a set of mise daemons:
# `make dev` starts it, `make dev-down` stops it.

# Dev DB URLs derive from the `db` daemon (mise exports PG*/DATABASE_URL); the test and
# atlas scratch databases live on the same server. Override for an external Postgres.
PGHOSTPORT   ?= $(or $(PGHOST),127.0.0.1):$(or $(PGPORT),5432)
PGAUTH       ?= $(if $(PGUSER),$(PGUSER)@,)
TEST_DB_URL  ?= postgres://$(PGAUTH)$(PGHOSTPORT)/1mail_test?sslmode=disable
ATLAS_DB_URL ?= postgres://$(PGAUTH)$(PGHOSTPORT)/atlas_dev?sslmode=disable

setup: install db-up db-create db-create-test db-create-atlas db-migrate db-seed

install:
	mise install
	pnpm install
	go mod download

# Starts the Postgres daemon (no-op when it is already running).
db-up:
	mise daemons start db

db-create:
	go run ./cmd/db create

db-create-test:
	APP_ENV=test DATABASE_URL=$(TEST_DB_URL) go run ./cmd/db create

# Scratch DB atlas uses to compute migration diffs (see atlas.hcl `dev`).
db-create-atlas:
	DATABASE_URL=$(ATLAS_DB_URL) go run ./cmd/db create

db-drop:
	go run ./cmd/db drop

db-drop-test:
	APP_ENV=test DATABASE_URL=$(TEST_DB_URL) go run ./cmd/db drop

db-migrate: db-migrate-atlas db-migrate-river

db-migrate-atlas:
	atlas migrate apply --env local --allow-dirty

# river owns its schema (river_job, …), applied out of band from Atlas.
db-migrate-river:
	go run ./cmd/db river-up

db-seed:
	go run ./cmd/seed

db-reset: db-drop db-create db-migrate

db-reset-test: db-drop-test db-create-test

db-generate:
	atlas migrate diff --env local $(name)

# Mint a fresh ENCRYPTION_KEY (base64 Tink keyset) to paste into your .env.
gen-encryption-key:
	go run ./cmd/genkey

dev:
	mise daemons start caddy

dev-down:
	mise daemons stop

test: db-create-test
	APP_ENV=test DATABASE_URL=$(TEST_DB_URL) go test -p 1 ./...

test-watch:
	pnpm exec vitest

# One-shot frontend run (Vitest Browser Mode, headless Chromium) for CI.
test-frontend:
	pnpm exec vitest run

update: update-npm update-go update-skills

update-npm:
	sh -c 'pnpm exec ncu -u && pnpm update'

update-go:
	sh -c 'go get -u ./... && go mod tidy'

update-skills:
	npx skills@latest update -y

generate-typespec-external:
	pnpm exec tsp compile typespec/external

generate-typespec-site:
	pnpm exec tsp compile typespec/site

generate-typespec-collect:
	pnpm exec tsp compile typespec/collect

generate-typespec: generate-typespec-external generate-typespec-site generate-typespec-collect

# Generates both the site client and the collect types (single config, two jobs).
generate-openapi-site:
	pnpm exec openapi-ts -f openapi-ts.config.ts

generate-openapi: generate-openapi-site

generate-i18n-types:
	pnpm run i18n:types

generate-backend:
	sh -c 'cd ent && go run -mod=mod entc.go'
	sh -c 'go tool ogen --target gen/site     --package siteapi     --clean openapi/site.openapi.json && \
		go tool ogen --target gen/external --package externalapi --clean openapi/external.openapi.json && \
		go tool ogen --target gen/collect  --package collectapi  --clean openapi/collect.openapi.json'
	go tool goverter gen ./internal/api/site/resources ./internal/api/external/resources

generate: generate-typespec generate-openapi generate-backend check-fix

# Fails when committed generated output (openapi/, gen/, ent/, src/generated/, *_gen.go)
# differs from what `make generate` produces: a hand-edit, or a schema/TypeSpec change
# whose output was not committed. Needs a clean working tree, so it is a CI gate rather
# than part of `make check`.
check-generated: generate
	@test -z "$$(git status --porcelain)" || { echo 'generated code is out of date; run `make generate` and commit:'; git status --short; git diff --stat; exit 1; }

check: check-fe check-i18n check-be check-deps check-security

check-fe: check-css
	pnpm exec tsc --noEmit
	pnpm exec oxlint
	pnpm exec oxfmt --check

# The project has no custom CSS: everything is styled through Mantine. Library
# stylesheets are imported from node_modules; no stylesheet may be tracked here.
check-css:
	@if git ls-files | grep -Ei '\.(css|scss|sass|less|pcss|styl)$$'; then \
		echo 'error: custom CSS is not allowed, style through Mantine'; exit 1; fi

# Unused files, exports and dependencies (knip.json).
check-deps:
	pnpm exec knip

# Read-only: --dry-run never writes, --ci exits non-zero on drift.
check-i18n:
	pnpm exec i18next-cli extract --ci --dry-run
	pnpm exec i18next-cli types --ci

check-be:
	golangci-lint run ./...
	go tool govulncheck ./...

# Secrets and workflow linters, installed by mise (.mise.toml).
check-security:
	gitleaks git --no-banner --redact
	jactionlint
	zizmor --no-progress .github

# i18n runs first so oxfmt formats the freshly extracted JSON.
check-fix: check-fix-i18n check-fix-fe check-fix-be

check-fix-i18n:
	pnpm exec i18next-cli extract --with-types

check-fix-fe:
	pnpm exec oxlint --fix
	# tsp format rewrites tspconfig.yaml quotes; oxfmt must run last to normalize them.
	pnpm exec tsp format typespec
	pnpm exec oxfmt

check-fix-be:
	go fmt ./...

# Builds the standalone tracker (IIFE) and copies it into the Go embed tree.
build-tracker:
	pnpm build:tracker
	cp packages/analytics/dist/t.js internal/server/assets/t.js

# Builds the SPA (Vite) and copies dist/ into the Go embed tree so the binary
# can serve the frontend itself. Gitignored; populated only for release builds.
build-spa:
	pnpm build
	rm -rf internal/server/assets/spa
	mkdir -p internal/server/assets/spa
	cp -R dist/. internal/server/assets/spa/

VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Produces the self-contained release binary: tracker + SPA embedded.
build: build-tracker build-spa
	go build -tags embed_spa -ldflags "$(LDFLAGS)" -o bin/1mail ./cmd/server

# Lines-of-code report (scc), excluding generated code. The set of generated files
# is the source of truth in .gitattributes (linguist-generated=true), so we feed scc
# only the tracked files git marks as hand-written — no fragile exclude patterns.
# Runs on the host; needs scc + git (https://github.com/boyter/scc).
loc:
	git ls-files \
	  | git check-attr --stdin linguist-generated \
	  | awk -F': ' '$$3!="true"{print $$1}' \
	  | xargs scc

.PHONY: setup install db-up db-create db-create-test db-create-atlas db-drop db-drop-test db-migrate db-migrate-atlas db-migrate-river db-seed db-reset db-reset-test db-generate dev dev-down test test-watch test-frontend update update-npm update-go update-skills generate check-generated generate-backend generate-openapi generate-openapi-site generate-typespec generate-typespec-external generate-typespec-site generate-typespec-collect generate-i18n-types check check-fe check-css check-deps check-security check-i18n check-be check-fix check-fix-i18n check-fix-fe check-fix-be build-tracker build-spa build loc
