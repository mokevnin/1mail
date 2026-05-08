setup: install db-create db-create-test db-migrate-dev

install:
	pnpm install
	cd backend && go mod download

db-create:
	psql "postgres://postgres:postgres@localhost:5432/postgres" -c 'CREATE DATABASE "1mail"' 2>/dev/null || true

db-create-test:
	psql "postgres://postgres:postgres@localhost:5432/postgres" -c 'CREATE DATABASE "1mail_test"' 2>/dev/null || true

db-generate:
	cd backend && atlas migrate diff --env local --name $(name)

db-migrate:
	cd backend && atlas migrate apply --env local --url $$DATABASE_URL --allow-dirty
	cd backend && river migrate-up --database-url $$DATABASE_URL

db-migrate-dev:
	. .env && cd backend && atlas migrate apply --env local --url $$DATABASE_URL --allow-dirty && river migrate-up --database-url $$DATABASE_URL

dev-frontend:
	npx vite

dev-backend:
	cd backend && air

dev:
	overmind start

test: db-create-test
	cd backend && go test ./...

test-watch:
	pnpm exec vitest

update: update-npm update-go

update-npm:
	npx ncu -u
	pnpm update

update-go:
	cd backend && go get -u ./... && go mod tidy

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

generate: generate-typespec generate-openapi check-fix

check:
	npx tsgo --noEmit
	npx biome check .
	cd backend && go vet ./...

check-fix:
	pnpx @biomejs/biome check --write
	npx tsp format typespec
	cd backend && go fmt ./...

build:
	cd backend && go build ./...

.PHONY: setup install db-create db-create-test db-generate db-migrate db-migrate-dev dev-frontend dev-backend dev test test-watch update update-npm update-go generate generate-openapi generate-openapi-external generate-openapi-site generate-typespec generate-typespec-external generate-typespec-site generate-typespec-collect check check-fix build
