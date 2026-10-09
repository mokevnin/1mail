# Go test harness

`internal/testhelper.Setup(t)` is the standard fixture. It migrates schema + loads
committed YAML fixtures from `fixtures/` **once per process**, then each test runs inside
a **go-txdb** transaction that rolls back on cleanup (full isolation, DB stays at fixture
state). Tests drive the real server in-memory via the **typed ogen client** + an injecting
transport (`env.Transport(headers)`) — no sockets. See `internal/api/site/contacts_test.go`.

**Build test scenarios on the committed fixtures — don't fabricate the primary
entities inline.** Reference fixture rows by their known IDs (e.g. draft broadcast 100,
sent broadcast 200 with recipient rows 1000+, anchor contacts 1–3, anchor segments 1–2);
query counts that vary with the dataset dynamically instead of hardcoding them. If a
scenario isn't covered, **add a fixture row** rather than a `client.X.Create()` in the
test. Anchor rows flagged "DO NOT change" in the YAML are referenced across the suite —
leave them. txdb rolls back each test, so mutating a fixture row inside a test is fine.
Exception: incidental one-off records (e.g. a suppression to trip a specific edge) may be
created inline when no fixture expresses them.
