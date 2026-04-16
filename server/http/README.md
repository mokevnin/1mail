## Public HTTP API

This directory contains external HTTP endpoints served by Fastify.

- `server/http/*` is the public API surface described by TypeSpec/OpenAPI.
- `server/trpc/*` is the internal API used by the frontend.
- `src/routes/*` contains frontend page routes for TanStack Router.

Keep route modules here transport-focused: they should expose HTTP handlers and schemas, while shared business logic stays outside the transport layer.
