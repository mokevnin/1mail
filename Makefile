setup:
	pnpm install

db-generate:
	pnpm exec drizzle-kit generate

db-migrate:
	pnpm exec drizzle-kit migrate

dev-frontend:
	npx vite

dev-backend:
	npx fastify start -w -l info -P app.ts

dev:
	overmind start

test:
	pnpm exec vitest run

test-watch:
	pnpm exec vitest

update: update-npm-deps

update-npm-deps:
	npx ncu -u
	pnpm update

generate-typespec:
	npx tsp compile typespec

generate-typebox:
	npx openapi-box openapi/openapi.yaml -o generated/openapi-box.js

generate: generate-typespec generate-typebox lint-fix

check:
	npx tsgo --noEmit
	npx biome check .

check-fix:
	pnpx @biomejs/biome check --write
	npx tsp format typespec

build:
	pnpm run build

.PHONY: test test-watch db-generate db-migrate
