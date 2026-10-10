# Tracking visitors and events

Tracking turns anonymous visits into contacts you can segment on. A small script runs in the
browser, assigns each device a visitor id, and sends events to your sphericon instance.

## Add the tracker

Add the script to your pages. The collect key and a ready-made test request are in the tracking
section of the workspace **Settings**.

```html
<script
  async
  src="https://sphericon.example.com/t.js"
  data-collect-key="YOUR_COLLECT_KEY"
  data-collect-url="https://sphericon.example.com"
></script>
```

The tracker stores a first-party `om_vid` cookie as the visitor id. It batches events and
flushes them shortly after they are queued and when the page is hidden.

## Send events and identify people

Commands queue on `window._omq`, so you can call them before the script has loaded.

```js
window._omq = window._omq || []

// Record something a visitor did
_omq.push(['track', 'viewed_pricing', { plan: 'team' }])

// Tell sphericon who this visitor is
_omq.push(['identify', { email: 'ada@example.com', subjectId: 'user_42' }])
```

`identify` binds the visitor to a contact and asserts its alias keys: `subjectId`, `email` and
`phone`. Everything that visitor did earlier, while anonymous, is stitched onto the contact, so
pre-signup behavior becomes visible to segments.

The same contract is available as an HTTP API if you would rather send from your server, using
the collect key in an `x-collect-key` header:

| Endpoint                 | Purpose                                      |
| ------------------------ | -------------------------------------------- |
| `POST /collect/identify` | Bind a `visitorId` to alias keys and traits. |
| `POST /collect/events`   | Record a batch of events for visitors.       |

## Contacts and identity

A contact is anchored by an internal id. Email, phone and `subject_id` are alias keys: each is
unique within the workspace and any of them may be absent. Events attach to the resolved
contact at ingest, never by email, so changing someone's email does not orphan their history.

## Custom fields

Any trait or property you send under an unknown key is created as a typed **custom field** with
an inferred type. From that moment it is a first-class, renameable field the segment builder
offers, not an anonymous bag of keys.

## Server-side events

Events you record from your backend use the API with a bearer token and the `events:write`
scope:

```sh
curl https://sphericon.example.com/api/events/batch \
  -H "Authorization: Bearer $SPHERICON_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"events":[{"action":"subscription_started","subjectId":"user_42"}]}'
```

See [API](/guide/api) for tokens and the other endpoints.
