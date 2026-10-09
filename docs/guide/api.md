# API

1mail exposes three HTTP surfaces. All are described by OpenAPI documents generated from
TypeSpec, which is the single source of truth; you will find them in the repository under
`openapi/`.

| Surface      | Base path  | Used by                             | Auth                            |
| ------------ | ---------- | ----------------------------------- | ------------------------------- |
| **External** | `/api`     | Your backend and automation         | `Authorization: Bearer <token>` |
| **Collect**  | `/collect` | The browser tracker and your server | `x-collect-key` header          |
| **Site**     | `/site`    | The 1mail web app itself            | Session, per workspace          |

The site API is the web app's own backend and may change with the UI. Build on the external and
collect APIs.

## Tokens and scopes

Create an API token under **Settings → API keys**. Each token carries scopes, so you can hand an
integration only what it needs, for example:

- `contacts:read`, `contacts:write`
- `events:write`
- `segments:read`, `segments:write`
- `broadcasts:write`, and the separate `broadcasts:send` to actually send
- `automations:write`, and `automations:activate` to turn one on
- `emails:send` for transactional mail
- `webhooks:read`, `webhooks:write`

Sending and activating have their own scopes, so a token that can edit drafts cannot send mail.

## What the external API covers

| Area                | Endpoints                                                            |
| ------------------- | -------------------------------------------------------------------- |
| Contacts            | `/contacts`, `/contacts/batch`, `/contacts/{id}/tags`                |
| Custom fields, tags | `/custom-fields`, `/tags`                                            |
| Events              | `/events`, `/events/batch`, `/events/actions`                        |
| Segments            | `/segments`, `/segments/preview`                                     |
| Templates           | `/templates`                                                         |
| Broadcasts          | `/broadcasts`, `…/audience`, `…/schedule`, `…/test-send`, `…/report` |
| Automations         | `/automations`, `…/activate`, `…/deactivate`                         |
| Transactional       | `/emails`                                                            |
| Consent             | `/unsubscribes`, `/suppressions`                                     |
| Sending domains     | `/sending-domains`, `/sending-domains/rates`                         |
| Webhooks            | `/webhooks`                                                          |

### Example: upsert contacts

```sh
curl https://1mail.example.com/api/contacts/batch \
  -H "Authorization: Bearer $ONEMAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"contacts":[{"subjectId":"user_42","email":"ada@example.com","firstName":"Ada"}]}'
```

## Operating 1mail with an agent

The same binary serves an MCP endpoint at `/mcp`. Its tools are projected from the external API
contract, so token scopes, workspace scoping and errors behave exactly as they do over `/api`.
See [the design decision](/adr/0016-mcp-surface-projection-of-external-api).

## Webhooks

Register an endpoint with `POST /api/webhooks` to be notified when things happen. Leave the event
list empty to receive everything, or name the events you want. Endpoints can be disabled without
deleting them.
