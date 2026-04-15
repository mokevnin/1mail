setup:
	pnpm install

dev:
	pnpm run dev

test:
	pnpm run test

update: update-npm-deps

update-npm-deps:
	npx ncu -u
	pnpm update

generate:
	pnpm run typespec:compile
	pnpm run openapi-ts

format:
	pnpm run format

lint:
	pnpm run lint

lint-fix:
	pnpm run lint:fix
	pnpm run typespec:format

check:
	pnpm run check:apply

build:
	pnpm run build

.PHONY: test
