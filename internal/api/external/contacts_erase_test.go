package external_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/automationrun"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/confirmation"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/ent/outboundmessage"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/ent/tag"
	"github.com/mokevnin/1mail/ent/unsubscribe"
	"github.com/mokevnin/1mail/ent/visitor"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// initechClient is an /api client for the Initech Workspace, the tenant that holds the
// fixture Contact carrying every dependent row (ContactErasable).
func initechClient(t *testing.T, env *testhelper.TestEnv, scopes ...string) *externalapi.Client {
	t.Helper()
	return env.ExternalWithToken(t, env.ScopedBearerFor(t, fixtures.InitechID, scopes...))
}

func eraseErasable(t *testing.T, env *testhelper.TestEnv) {
	t.Helper()
	res, err := initechClient(t, env, "contacts:erase").ContactsDelete(context.Background(),
		externalapi.ContactsDeleteParams{ID: entityIDString(fixtures.ContactErasableID)})
	require.NoError(t, err)
	require.IsTypef(t, &externalapi.ContactsDeleteNoContent{}, res, "got %T", res)
}

func eventByID(t *testing.T, env *testhelper.TestEnv, id int64) *ent.Event {
	t.Helper()
	e, err := env.DB.Event.Get(context.Background(), id)
	require.NoError(t, err)
	return e
}

func eventExists(t *testing.T, env *testhelper.TestEnv, id int64) bool {
	t.Helper()
	ok, err := env.DB.Event.Query().Where(event.ID(id)).Exist(context.Background())
	require.NoError(t, err)
	return ok
}

func TestExternalContactsEraseNeedsTheEraseScope(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	id := entityIDString(fixtures.ContactErasableID)

	denied, err := initechClient(t, env, "contacts:read", "contacts:write").ContactsDelete(ctx, externalapi.ContactsDeleteParams{ID: id})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsDeleteUnauthorized{}, denied, "contacts:write does not erase")
	still, err := env.DB.Contact.Query().Where(contact.ID(fixtures.ContactErasableID)).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, still)

	eraseErasable(t, env)
	gone, err := env.DB.Contact.Query().Where(contact.ID(fixtures.ContactErasableID)).Exist(ctx)
	require.NoError(t, err)
	assert.False(t, gone)
}

func TestExternalContactsEraseRemovesThePersonsData(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	eraseErasable(t, env)

	// Removed outright: the Contact, its Visitors, Tag links, Confirmation, runs.
	n, err := env.DB.Visitor.Query().Where(visitor.ID(300)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "visitors")
	linked, err := env.DB.Tag.Query().Where(tag.ID(fixtures.TagInitechID)).QueryContacts().IDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, []int64{fixtures.ContactBystanderID}, linked, "the tag stays, only the bystander keeps it")
	n, err = env.DB.Confirmation.Query().Where(confirmation.ID(fixtures.ConfirmationErasableID)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "confirmation")
	n, err = env.DB.AutomationRun.Query().Where(automationrun.ID(fixtures.AutomationRunErasableID)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "automation runs")

	// Events: customer-tracked and marketing.confirmed are deleted ...
	for _, id := range []int64{
		fixtures.EventErasableCustomID, fixtures.EventErasableAnonymousID, fixtures.EventErasableConfirmedID,
	} {
		assert.Falsef(t, eventExists(t, env, id), "event %d is deleted", id)
	}

	// ... every reserved system Event stays as an anonymous row.
	for _, id := range []int64{
		fixtures.EventErasableSentID, fixtures.EventErasableOpenedID, fixtures.EventErasableClickedID,
		fixtures.EventErasableBouncedID, fixtures.EventErasableComplainedID, fixtures.EventErasableUnsubscribedID,
		fixtures.EventErasableCreatedID,
	} {
		e := eventByID(t, env, id)
		assert.Nilf(t, e.ContactID, "event %d contact_id", id)
		assert.Nilf(t, e.VisitorID, "event %d visitor_id", id)
		assert.Nilf(t, e.Email, "event %d email", id)
		assert.Nilf(t, e.Phone, "event %d phone", id)
		assert.Emptyf(t, e.SubjectID, "event %d subject_id", id)
		assert.NotNilf(t, e.OccurredAt, "event %d keeps its time", id)
	}
	// Only the non-personal delivery facts the rates are computed from survive.
	assert.Equal(t, map[string]any{"sendingDomain": "mail.initech.test"}, eventByID(t, env, fixtures.EventErasableSentID).Properties)
	assert.Equal(t, map[string]any{"sendingDomain": "mail.initech.test", "bounceKind": "permanent"}, eventByID(t, env, fixtures.EventErasableBouncedID).Properties)
	assert.Empty(t, eventByID(t, env, fixtures.EventErasableClickedID).Properties, "the clicked url is gone")
	assert.Empty(t, eventByID(t, env, fixtures.EventErasableCreatedID).Properties)

	// The refusal survives by Destination, detached from the Contact.
	unsub, err := env.DB.Unsubscribe.Get(ctx, 300)
	require.NoError(t, err)
	assert.Equal(t, fixtures.ContactErasableEmail, unsub.Destination)
	assert.Nil(t, unsub.ContactID)
	supp, err := env.DB.Suppression.Get(ctx, 300)
	require.NoError(t, err)
	assert.Equal(t, fixtures.ContactErasableEmail, supp.Destination)
	assert.Nil(t, supp.ContactID)

	// Delivery records stay as rows that no longer name anyone.
	for _, id := range []int64{fixtures.OutboundMessageErasableBroadcastID, fixtures.OutboundMessageErasableTransactionalID} {
		m, err := env.DB.OutboundMessage.Get(ctx, id)
		require.NoError(t, err)
		assert.Emptyf(t, m.Destination, "message %d destination", id)
		assert.Nilf(t, m.ContactID, "message %d contact_id", id)
		assert.Equal(t, outboundmessage.StatusSent, m.Status)
	}
	r, err := env.DB.BroadcastRecipient.Get(ctx, fixtures.BroadcastRecipientErasableID)
	require.NoError(t, err)
	assert.Zero(t, r.ContactID)
	assert.Equal(t, broadcastrecipient.StatusSent, r.Status)
}

func TestExternalContactsEraseLeavesOtherContactsAlone(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	eraseErasable(t, env)

	b, err := env.DB.Contact.Get(ctx, fixtures.ContactBystanderID)
	require.NoError(t, err)
	assert.Equal(t, "ben@initech.test", *b.Email)
	for _, id := range []int64{fixtures.EventBystanderCustomID, fixtures.EventBystanderSentID} {
		e := eventByID(t, env, id)
		assert.EqualValues(t, fixtures.ContactBystanderID, *e.ContactID)
		assert.Equal(t, "ben@initech.test", *e.Email)
	}
	m, err := env.DB.OutboundMessage.Get(ctx, fixtures.OutboundMessageBystanderID)
	require.NoError(t, err)
	assert.Equal(t, "ben@initech.test", m.Destination)
	r, err := env.DB.BroadcastRecipient.Get(ctx, fixtures.BroadcastRecipientBystanderID)
	require.NoError(t, err)
	assert.EqualValues(t, fixtures.ContactBystanderID, r.ContactID)
}

func TestExternalContactsEraseDoesNotMoveRatesOrMetering(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	c := initechClient(t, env, "contacts:erase", "sending_domains:read")

	ratesOf := func() any {
		res, err := c.SendingDomainRatesList(ctx, externalapi.SendingDomainRatesListParams{})
		require.NoError(t, err)
		ok, isOK := res.(*externalapi.SendingDomainRatesListOK)
		require.Truef(t, isOK, "got %T", res)
		return ok.Items
	}
	// The billable/reported volume: system Event rows per action in the Workspace.
	countsOf := func() map[string]int {
		var rows []struct {
			Action string `json:"action"`
			Count  int    `json:"count"`
		}
		require.NoError(t, env.DB.Event.Query().
			Where(event.WorkspaceID(fixtures.InitechID)).
			GroupBy(event.FieldAction).
			Aggregate(ent.As(ent.Count(), "count")).
			Scan(ctx, &rows))
		out := map[string]int{}
		for _, r := range rows {
			out[r.Action] = r.Count
		}
		return out
	}

	ratesBefore, countsBefore := ratesOf(), countsOf()
	assert.Equal(t, 2, countsBefore["email.sent"])

	res, err := c.ContactsDelete(ctx, externalapi.ContactsDeleteParams{ID: entityIDString(fixtures.ContactErasableID)})
	require.NoError(t, err)
	require.IsType(t, &externalapi.ContactsDeleteNoContent{}, res)

	assert.Equal(t, ratesBefore, ratesOf(), "complaint and bounce rates")
	countsAfter := countsOf()
	for _, action := range []string{"email.sent", "email.opened", "email.clicked", "email.bounced", "email.complained", "email.unsubscribed", "contact.created"} {
		assert.Equalf(t, countsBefore[action], countsAfter[action], "%s count", action)
	}
}

// A failure anywhere in the transaction leaves nothing partially erased.
func TestExternalContactsEraseIsAtomic(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	// Fail the very last step, deleting the Contact, after everything else was written.
	env.Bus.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if m.Type() == ent.TypeContact && m.Op().Is(ent.OpDelete|ent.OpDeleteOne) {
				return nil, errors.New("injected failure")
			}
			return next.Mutate(ctx, m)
		})
	})

	_, err := initechClient(t, env, "contacts:erase").ContactsDelete(ctx,
		externalapi.ContactsDeleteParams{ID: entityIDString(fixtures.ContactErasableID)})
	require.Error(t, err)

	sent := eventByID(t, env, fixtures.EventErasableSentID)
	require.NotNil(t, sent.ContactID)
	assert.Equal(t, "erin@initech.test", *sent.Email)
	assert.True(t, eventExists(t, env, fixtures.EventErasableCustomID))
	assert.True(t, eventExists(t, env, fixtures.EventErasableConfirmedID))
	m, err := env.DB.OutboundMessage.Get(ctx, fixtures.OutboundMessageErasableBroadcastID)
	require.NoError(t, err)
	assert.Equal(t, "erin@initech.test", m.Destination)
	unsub, err := env.DB.Unsubscribe.Get(ctx, 300)
	require.NoError(t, err)
	assert.NotNil(t, unsub.ContactID)
	n, err := env.DB.Visitor.Query().Where(visitor.ID(300)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	n, err = env.DB.Confirmation.Query().Where(confirmation.ID(fixtures.ConfirmationErasableID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	n, err = env.DB.Suppression.Query().Where(suppression.ContactID(fixtures.ContactErasableID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	n, err = env.DB.Unsubscribe.Query().Where(unsubscribe.ContactID(fixtures.ContactErasableID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestExternalContactsEraseIsWorkspaceScoped(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	// An Acme token cannot erase an Initech Contact: it does not exist for Acme.
	res, err := env.ExternalScoped(t, "contacts:erase").ContactsDelete(ctx,
		externalapi.ContactsDeleteParams{ID: entityIDString(fixtures.ContactErasableID)})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.ContactsDeleteNotFound{}, res)

	still, err := env.DB.Contact.Query().Where(contact.ID(fixtures.ContactErasableID)).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, still)
	assert.Equal(t, "erin@initech.test", *eventByID(t, env, fixtures.EventErasableSentID).Email)
}

// An erased person who reappears is a new Contact, and the surviving opt-out still binds
// their address.
func TestExternalContactsEraseDoesNotBlockReappearance(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	eraseErasable(t, env)

	res, err := initechClient(t, env, "contacts:write").ContactsCreate(ctx, &externalapi.CreateContactInput{
		Email: externalapi.NewOptNilEmailAddress(fixtures.ContactErasableEmail),
	})
	require.NoError(t, err)
	created, ok := res.(*externalapi.ContactResource)
	require.Truef(t, ok, "got %T", res)
	assert.NotEqual(t, entityIDString(fixtures.ContactErasableID), created.ID)

	n, err := env.DB.Unsubscribe.Query().Where(unsubscribe.DestinationEQ(fixtures.ContactErasableEmail), unsubscribe.WorkspaceID(fixtures.InitechID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "the opt-out outlives the gap")
}
