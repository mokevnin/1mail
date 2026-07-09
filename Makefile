# All tooling runs inside the dev containers (docker-compose.yml) so a host with
# only Docker installed can build, lint, test, and generate. The runner vars below
# are overridable: CI (which installs native toolchains) runs everything natively
# with `make check RUN_FE= RUN_GO= RUN_GO_DB=`.
#
#   RUN_FE    — node tooling (frontend image): tsgo, biome, tsp, openapi-ts, pnpm
#   RUN_GO    — go tooling, no DB needed: golangci-lint, go fmt, entc, ogen, go mod
#   RUN_GO_DB — go tooling that needs Postgres: cmd/db, atlas, go test (starts `db`)
# Run containers as the host user so files written into the bind mount (generated
# code, caches) are owned by you, not root (matters on native-Linux hosts).
export COMPOSE_UID ?= $(shell id -u)
export COMPOSE_GID ?= $(shell id -g)

DC        ?= docker compose
RUN_FE    ?= $(DC) run --rm --no-deps frontend
RUN_GO    ?= $(DC) run --rm --no-deps backend
RUN_GO_DB ?= $(DC) run --rm backend

# Connection strings default to the compose `db` service. Override the host part
# for a host-native run, e.g. `make test RUN_GO_DB= TEST_DB_URL=postgres://...@localhost:5432/1mail_test?sslmode=disable`.
TEST_DB_URL  ?= postgres://postgres:postgres@db:5432/1mail_test?sslmode=disable
ATLAS_DB_URL ?= postgres://postgres:postgres@db:5432/atlas_dev?sslmode=disable

setup: install db-create db-create-test db-create-atlas db-migrate db-seed

# Frontend deps install into the bind-mounted node_modules (Linux-native, run in
# compose). Go modules download into the host-bind-mounted cache (./.cache/go-mod)
# so host gopls resolves imports. Neither needs a host toolchain.
install:
	$(RUN_FE) pnpm install
	$(RUN_GO) go mod download

db-create:
	$(RUN_GO_DB) go run ./cmd/db create

db-create-test:
	$(RUN_GO_DB) sh -c 'APP_ENV=test DATABASE_URL=$(TEST_DB_URL) go run ./cmd/db create'

# Scratch DB atlas uses to compute migration diffs (see atlas.hcl `dev`).
db-create-atlas:
	$(RUN_GO_DB) sh -c 'DATABASE_URL=$(ATLAS_DB_URL) go run ./cmd/db create'

db-drop:
	$(RUN_GO_DB) go run ./cmd/db drop

db-drop-test:
	$(RUN_GO_DB) sh -c 'APP_ENV=test DATABASE_URL=$(TEST_DB_URL) go run ./cmd/db drop'

db-migrate: db-migrate-atlas db-migrate-river

db-migrate-atlas:
	$(RUN_GO_DB) atlas migrate apply --env local --allow-dirty

# river owns its schema (river_job, …), applied out of band from Atlas.
db-migrate-river:
	$(RUN_GO_DB) go run ./cmd/db river-up

db-seed:
	$(RUN_GO_DB) go run ./cmd/seed

db-reset: db-drop db-create db-migrate

db-reset-test: db-drop-test db-create-test

db-generate:
	$(RUN_GO_DB) atlas migrate diff --env local $(name)

# Mint a fresh ENCRYPTION_KEY (base64 Tink keyset) to paste into your .env.
gen-encryption-key:
	$(RUN_GO) go run ./cmd/genkey

dev:
	$(DC) up

dev-down:
	$(DC) down

test: db-create-test
	$(RUN_GO_DB) sh -c 'APP_ENV=test DATABASE_URL=$(TEST_DB_URL) go test -p 1 ./...'

test-watch:
	$(RUN_FE) pnpm exec vitest

# One-shot frontend run (Vitest Browser Mode, headless Chromium) for CI.
test-frontend:
	$(RUN_FE) pnpm exec vitest run

update: update-npm update-go update-skills

update-npm:
	$(RUN_FE) sh -c 'pnpm exec ncu -u && pnpm update'

update-go:
	$(RUN_GO) sh -c 'go get -u ./... && go mod tidy'

update-skills:
	$(RUN_FE) npx skills@latest update -y

generate-typespec-external:
	$(RUN_FE) pnpm exec tsp compile typespec/external

generate-typespec-site:
	$(RUN_FE) pnpm exec tsp compile typespec/site

generate-typespec-collect:
	$(RUN_FE) pnpm exec tsp compile typespec/collect

generate-typespec: generate-typespec-external generate-typespec-site generate-typespec-collect

# Generates both the site client and the collect types (single config, two jobs).
generate-openapi-site:
	$(RUN_FE) pnpm exec openapi-ts -f openapi-ts.config.ts

generate-openapi: generate-openapi-site

generate-i18n-types:
	$(RUN_FE) pnpm run i18n:types

generate-backend:
	$(RUN_GO) sh -c 'cd ent && go run -mod=mod entc.go'
	$(RUN_GO) sh -c 'go tool ogen --target gen/site     --package siteapi     --clean openapi/site.openapi.json && \
		go tool ogen --target gen/external --package externalapi --clean openapi/external.openapi.json && \
		go tool ogen --target gen/collect  --package collectapi  --clean openapi/collect.openapi.json'
	$(RUN_GO) go tool goverter gen ./internal/api/site/resources ./internal/api/external/resources

generate: generate-typespec generate-openapi generate-backend check-fix

check: check-fe check-i18n check-be

check-fe:
	$(RUN_FE) pnpm exec tsgo --noEmit
	$(RUN_FE) pnpm exec biome check .

# Read-only: --dry-run never writes, --ci exits non-zero on drift.
check-i18n:
	$(RUN_FE) pnpm exec i18next-cli extract --ci --dry-run
	$(RUN_FE) pnpm exec i18next-cli types --ci

check-be:
	$(RUN_GO) golangci-lint run ./...

# i18n runs first so biome formats the freshly extracted JSON.
check-fix: check-fix-i18n check-fix-fe check-fix-be

check-fix-i18n:
	$(RUN_FE) pnpm exec i18next-cli extract --with-types

check-fix-fe:
	$(RUN_FE) pnpm exec biome check --write
	$(RUN_FE) pnpm exec tsp format typespec

check-fix-be:
	$(RUN_GO) go fmt ./...

# Builds the standalone tracker (IIFE) and copies it into the Go embed tree.
build-tracker:
	$(RUN_FE) pnpm build:tracker
	cp packages/analytics/dist/t.js internal/server/assets/t.js

# Builds the SPA (Vite) and copies dist/ into the Go embed tree so the binary
# can serve the frontend itself. Gitignored; populated only for release builds.
build-spa:
	$(RUN_FE) pnpm build
	rm -rf internal/server/assets/spa
	mkdir -p internal/server/assets/spa
	cp -R dist/. internal/server/assets/spa/

VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Produces the self-contained release binary: tracker + SPA embedded.
build: build-tracker build-spa
	$(RUN_GO) go build -tags embed_spa -ldflags "$(LDFLAGS)" -o bin/1mail ./cmd/server

# Lines-of-code report (scc), excluding generated code. The set of generated files
# is the source of truth in .gitattributes (linguist-generated=true), so we feed scc
# only the tracked files git marks as hand-written — no fragile exclude patterns.
# Runs on the host; needs scc + git (https://github.com/boyter/scc).
loc:
	git ls-files \
	  | git check-attr --stdin linguist-generated \
	  | awk -F': ' '$$3!="true"{print $$1}' \
	  | xargs scc

.PHONY: setup install db-create db-create-test db-create-atlas db-drop db-drop-test db-migrate db-migrate-atlas db-migrate-river db-seed db-reset db-reset-test db-generate dev dev-down test test-watch test-frontend update update-npm update-go update-skills generate generate-backend generate-openapi generate-openapi-site generate-typespec generate-typespec-external generate-typespec-site generate-typespec-collect generate-i18n-types check check-fe check-i18n check-be check-fix check-fix-i18n check-fix-fe check-fix-be build-tracker build-spa build loc
