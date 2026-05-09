setup: install db-create db-create-test db-migrate

install:
	pnpm install
	go mod download
	# mise use -g air

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

dev-frontend:
	npx vite

dev-backend:
	air

dev:
	overmind start

test: db-create-test
	go test ./...

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

generate-openapi-external:
	pnpm exec openapi-ts -f openapi-ts.config.ts

generate-openapi-site:
	pnpm exec openapi-ts -f openapi-ts.site.config.ts

generate-openapi: generate-openapi-external generate-openapi-site

generate-i18n-types:
	pnpm run i18n:types

generate-backend:
	cd ent && go run -mod=mod entc.go
	oapi-codegen --package=siteapi --generate=echo-server,strict-server,models,embedded-spec -o gen/site/site.gen.go openapi/site.openapi.json
	oapi-codegen --package=externalapi --generate=echo-server,strict-server,models,embedded-spec -o gen/external/external.gen.go openapi/external.openapi.json
	oapi-codegen --package=collectapi --generate=echo-server,strict-server,models,embedded-spec -o gen/collect/collect.gen.go openapi/collect.openapi.json

generate: generate-typespec generate-openapi generate-backend generate-i18n-types check-fix

check:
	npx tsgo --noEmit
	npx biome check .
	go vet ./...

check-fix:
	pnpx @biomejs/biome check --write
	npx tsp format typespec
	go fmt ./...

build:
	go build ./...

.PHONY: setup install db-create db-create-test db-drop db-drop-test db-migrate db-seed db-reset db-reset-test db-generate dev-frontend dev-backend dev test test-watch update update-npm update-go generate generate-backend generate-openapi generate-openapi-external generate-openapi-site generate-typespec generate-typespec-external generate-typespec-site generate-typespec-collect generate-i18n-types check check-fix build
