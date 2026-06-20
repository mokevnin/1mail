# 1mail

Marketing Automation.

## Стек

- **Бэкенд** — Go + [echo](https://echo.labstack.com/), ORM [ent](https://entgo.io/),
  очереди [river](https://riverqueue.com/), pub/sub [watermill](https://watermill.io/).
- **Фронтенд** — React + Vite, [Mantine](https://mantine.dev/),
  [TanStack Router/Query](https://tanstack.com/), [oRPC](https://orpc.unnoq.com/).
- **API-контракты** — [TypeSpec](https://typespec.io/) как источник правды.

## Генерация кода

Пайплайн односторонний: TypeSpec → OpenAPI → код.

```
typespec/{external,site,collect}
  └─ tsp compile ─▶ openapi/*.openapi.json
       ├─ oapi-codegen ───────▶ gen/{external,site,collect}/*.gen.go  (Go echo-сервер)
       └─ @hey-api/openapi-ts ▶ src/generated/site                     (TS-клиент фронтенда)
```

Запуск:

```sh
make generate          # полный цикл: typespec → openapi → backend + frontend → i18n → форматирование
make generate-typespec # только typespec → openapi
make generate-backend  # только openapi → Go (ent + oapi-codegen)
make generate-openapi  # только openapi → TS-клиент
```

## Разработка

```sh
make setup   # установка зависимостей и создание БД
make dev     # overmind: фронтенд + бэкенд
make test    # go test ./...
make check   # tsgo --noEmit, biome, golangci-lint
```

`make check` requires [golangci-lint](https://golangci-lint.run/) (v2). The version is
pinned in `.mise.toml`, so the easiest way is [mise](https://mise.jdx.dev/):

```sh
mise install
```

Any other install method works too — it only needs to be on your `PATH`, e.g.
`brew install golangci-lint`.
