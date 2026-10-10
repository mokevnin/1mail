# Segments

A segment is a named, reusable rule that answers "which contacts match?". It is the one way to
choose who receives a broadcast.

## Build a rule

In **Segments**, combine conditions with all/any and nest groups where needed. A condition can
look at:

- **Attributes**: email, name, time zone and the other core fields.
- **Custom fields**: any field your tracking or import created, with the operators its type
  supports.
- **Tags**: whether the contact has a tag or not.
- **Events**: whether the contact did, or did not do, something, optionally within the last
  N days. Because delivery facts are events, you can target on `email.opened` or `email.clicked`
  the same way.

A live preview shows how many contacts match as you edit.

## Evaluated live

Membership is never stored. A segment is a query that runs when you preview it and when a
broadcast starts sending, so it reflects your data at that moment.

There is deliberately no list or "subscribed to X" group. Tags and custom fields describe a
contact; a segment decides who to target; and the opt-out rules described in
[Deliverability and consent](/guide/deliverability) always subtract from the result.

## Segments through the API

The definition is the rule query as JSON (the format used by react-querybuilder). An empty rule
group matches every contact.

```sh
curl https://sphericon.example.com/api/segments/preview \
  -H "Authorization: Bearer $SPHERICON_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"definition":"{\"combinator\":\"and\",\"rules\":[]}"}'
```
