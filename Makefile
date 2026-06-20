setup: install db-create db-create-test db-migrate

# Frontend deps are installed via the container into the bind-mounted tree
# (Linux-native binaries on the host), so they run in compose while host LSPs
# still resolve them. Go deps stay on the host for gopls.
install:
	docker compose run --rm frontend pnpm install
	go mod download

db-create:
	go run ./cmd/db create

db-create-test:
	APP_ENV=test go run ./cmd/db create

db-drop:
	go run ./cmd/db drop

db-drop-test:
	APP_ENV=test go run ./cmd/db drop

db-migrate:
	. .env && atlas migrate apply --env local --url $$DATABASE_URL --allow-dirty

db-seed:
	go run ./cmd/seed

db-reset: db-drop db-create db-migrate

db-reset-test: db-drop-test db-create-test

db-generate:
	atlas migrate diff --env local $(name)

dev:
	docker compose up

dev-down:
	docker compose down

test: db-create-test
	go test -p 1 ./...

test-watch:
	pnpm exec vitest

update: update-npm update-go

update-npm:
	npx ncu -u
	pnpm update

update-go:
	go get -u ./... && go mod tidy

generate-typespec-external:
	npx tsp compile typespec/external

generate-typespec-site:
	npx tsp compile typespec/site

generate-typespec-collect:
	npx tsp compile typespec/collect

generate-typespec: generate-typespec-external generate-typespec-site generate-typespec-collect

# Generates both the site client and the collect types (single config, two jobs).
generate-openapi-site:
	pnpm exec openapi-ts -f openapi-ts.config.ts

generate-openapi: generate-openapi-site

generate-i18n-types:
	pnpm run i18n:types

generate-backend:
	cd ent && go run -mod=mod entc.go
	go tool ogen --target gen/site     --package siteapi     --clean openapi/site.openapi.json
	go tool ogen --target gen/external --package externalapi --clean openapi/external.openapi.json
	go tool ogen --target gen/collect  --package collectapi  --clean openapi/collect.openapi.json

generate: generate-typespec generate-openapi generate-backend generate-i18n-types check-fix

check:
	npx tsgo --noEmit
	npx biome check .
	golangci-lint run ./...

check-fix:
	pnpx @biomejs/biome check --write
	npx tsp format typespec
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

.PHONY: setup install db-create db-create-test db-drop db-drop-test db-migrate db-seed db-reset db-reset-test db-generate dev dev-down test test-watch update update-npm update-go generate generate-backend generate-openapi generate-openapi-site generate-typespec generate-typespec-external generate-typespec-site generate-typespec-collect generate-i18n-types check check-fix build-tracker build-spa build
