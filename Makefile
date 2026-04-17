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

generate-typespec-external:
	npx tsp compile typespec/external

generate-typespec-site:
	npx tsp compile typespec/site

generate-typespec: generate-typespec-external generate-typespec-site

generate: generate-typespec check-fix

check:
	npx tsgo --noEmit
	npx biome check .

check-fix:
	pnpx @biomejs/biome check --write
	npx tsp format typespec

build:
	pnpm run build

.PHONY: test test-watch db-generate db-migrate generate generate-typespec generate-typespec-external generate-typespec-site
