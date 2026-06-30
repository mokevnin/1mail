package segments_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/predicate"
	"github.com/mokevnin/1mail/internal/segments"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wsID = int64(1) // fixture workspace "acme"

func rule(field, op, value string) segments.Rule {
	return segments.Rule{Field: field, Operator: op, Value: value}
}

// marshal encodes leaf rules (or nested groups) into the raw rules[] entries.
func marshal(entries ...any) []json.RawMessage {
	out := make([]json.RawMessage, len(entries))
	for i, e := range entries {
		b, err := json.Marshal(e)
		if err != nil {
			panic(err)
		}
		out[i] = b
	}
	return out
}

func TestCompileEvaluatesAgainstContacts(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	mk := func(email, first, status string, custom map[string]string) {
		c := env.DB.Contact.Create().SetWorkspaceID(wsID).SetEmail(email).SetFirstName(first)
		if status == "unsubscribed" {
			c.SetStatus(contact.StatusUnsubscribed)
		}
		if custom != nil {
			c.SetCustomFields(custom)
		}
		_, err := c.Save(ctx)
		require.NoError(t, err)
	}
	mk("ann@seg.test", "Ann", "active", map[string]string{"plan": "pro"})
	mk("bob@seg.test", "Bob", "active", map[string]string{"plan": "free"})
	mk("cara@seg.test", "Cara", "unsubscribed", map[string]string{"plan": "pro"})

	count := func(g segments.Group) int {
		p, err := segments.Compile(g, segments.ContactSchema())
		require.NoError(t, err)
		n, err := env.DB.Contact.Query().
			Where(contact.WorkspaceID(wsID), contact.EmailContainsFold("@seg.test"), predicate.Contact(p)).
			Count(ctx)
		require.NoError(t, err)
		return n
	}

	// Empty group matches everyone (within the @seg.test scope: all 3).
	assert.Equal(t, 3, count(segments.Group{}))

	// status = active
	assert.Equal(t, 2, count(segments.Group{Rules: marshal(rule("status", "=", "active"))}))

	// custom:plan = pro
	assert.Equal(t, 2, count(segments.Group{Rules: marshal(rule("custom:plan", "=", "pro"))}))

	// and: active AND plan=pro → just Ann
	assert.Equal(t, 1, count(segments.Group{
		Combinator: "and",
		Rules:      marshal(rule("status", "=", "active"), rule("custom:plan", "=", "pro")),
	}))

	// or: first_name=Ann OR first_name=Bob → 2
	assert.Equal(t, 2, count(segments.Group{
		Combinator: "or",
		Rules:      marshal(rule("first_name", "=", "Ann"), rule("first_name", "=", "Bob")),
	}))

	// custom:plan notNull → all 3 have it
	assert.Equal(t, 3, count(segments.Group{Rules: marshal(rule("custom:plan", "notNull", ""))}))

	// not: NOT(plan=pro) → only Bob (free)
	assert.Equal(t, 1, count(segments.Group{Not: true, Rules: marshal(rule("custom:plan", "=", "pro"))}))
}

func TestEventConditions(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	mk := func(email string) {
		_, err := env.DB.Contact.Create().SetWorkspaceID(wsID).SetEmail(email).Save(ctx)
		require.NoError(t, err)
	}
	mkEvent := func(email, action string, workspace int64, occurred time.Time) {
		_, err := env.DB.Event.Create().
			SetWorkspaceID(workspace).SetSubjectID(email).SetEmail(email).
			SetAction(action).SetOccurredAt(occurred).Save(ctx)
		require.NoError(t, err)
	}
	mk("ann@evseg.test")
	mk("bob@evseg.test")

	// A second real workspace (FK-valid) to prove the join is workspace-scoped.
	ws2, err := env.DB.Workspace.Create().
		SetName("Other").SetSlug("other-evseg").SetCollectKey("k-evseg").SetIngestKey("ik-evseg").SetUserID(1).Save(ctx)
	require.NoError(t, err)

	now := time.Now()
	mkEvent("ann@evseg.test", "page_view", wsID, now.AddDate(0, 0, -2))  // recent
	mkEvent("ann@evseg.test", "purchase", wsID, now.AddDate(0, 0, -40))  // old
	mkEvent("ann@evseg.test", "ws2_only", ws2.ID, now.AddDate(0, 0, -1)) // other workspace

	count := func(g segments.Group) int {
		p, err := segments.Compile(g, segments.ContactSchema())
		require.NoError(t, err)
		n, err := env.DB.Contact.Query().
			Where(contact.WorkspaceID(wsID), contact.EmailContainsFold("@evseg.test"), predicate.Contact(p)).
			Count(ctx)
		require.NoError(t, err)
		return n
	}

	// performed ever → only Ann did page_view.
	assert.Equal(t, 1, count(segments.Group{Rules: marshal(rule("event:page_view", "performed", ""))}))
	// performed within 7 days → Ann's page_view (2d ago) qualifies.
	assert.Equal(t, 1, count(segments.Group{Rules: marshal(rule("event:page_view", "performed", "7"))}))
	// purchase within 7 days → none (Ann's is 40d ago); ever → Ann.
	assert.Equal(t, 0, count(segments.Group{Rules: marshal(rule("event:purchase", "performed", "7"))}))
	assert.Equal(t, 1, count(segments.Group{Rules: marshal(rule("event:purchase", "performed", ""))}))
	// notPerformed page_view → Bob only.
	assert.Equal(t, 1, count(segments.Group{Rules: marshal(rule("event:page_view", "notPerformed", ""))}))
	// workspace isolation: ws2_only event must not match in ws1.
	assert.Equal(t, 0, count(segments.Group{Rules: marshal(rule("event:ws2_only", "performed", ""))}))

	// validation: bad operator + non-numeric window are rejected.
	assert.Error(t, segments.Validate(`{"rules":[{"field":"event:x","operator":"weird","value":""}]}`, segments.ContactSchema()))
	assert.Error(t, segments.Validate(`{"rules":[{"field":"event:x","operator":"performed","value":"soon"}]}`, segments.ContactSchema()))
}

func TestParseAndValidate(t *testing.T) {
	g, err := segments.Parse("")
	require.NoError(t, err)
	assert.Equal(t, "and", g.Combinator)

	require.NoError(t, segments.Validate(`{"combinator":"or","rules":[{"field":"email","operator":"contains","value":"x"}]}`, segments.ContactSchema()))

	// nested group
	require.NoError(t, segments.Validate(`{"combinator":"and","rules":[{"combinator":"or","rules":[{"field":"status","operator":"=","value":"active"}]}]}`, segments.ContactSchema()))

	// unknown field
	assert.Error(t, segments.Validate(`{"rules":[{"field":"nope","operator":"=","value":"x"}]}`, segments.ContactSchema()))
	// unsupported operator
	assert.Error(t, segments.Validate(`{"rules":[{"field":"email","operator":"weird","value":"x"}]}`, segments.ContactSchema()))
	// malformed JSON
	assert.Error(t, segments.Validate(`{not json`, segments.ContactSchema()))
}
