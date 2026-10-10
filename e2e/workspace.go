//go:build e2e

package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/membership"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/apitokens"
	"github.com/mokevnin/1mail/internal/events"
)

// EmailTimeout bounds every wait for an email; asynchronous steps are polled, never slept.
const EmailTimeout = 60 * time.Second

// tokenScopes are everything the scenario steps need; a test's token carries them all.
var tokenScopes = []string{
	"integrations:read", "integrations:write",
	"sending_domains:read", "sending_domains:write",
	"contacts:read", "contacts:write",
	"broadcasts:read", "broadcasts:write", "broadcasts:send",
	"automations:read", "automations:write", "automations:activate",
	"events:write",
}

// Workspace is one test's own tenant: a fresh User, Workspace and API token made with
// the product's own domain functions, a unique Sending domain and a client for the
// external API. Nothing is cleaned up; everything is Workspace-scoped, so tests are
// independent and may run in parallel.
type Workspace struct {
	t   testing.TB
	env *Env
	api *externalapi.Client

	// Domain is this Workspace's unique Sending domain (the dev DKIM lookup resolves
	// by name alone, so no two tests may share one).
	Domain string
	// FromEmail is the address mail is sent from, on Domain.
	FromEmail string
	// FromName is the sender display name configured on the Integration.
	FromName string

	msgIDs []string

	scoped *ent.Scoped
}

// uniq is a short random token for unique domains and recipients.
func uniq() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type bearer string

func (b bearer) BearerAuth(context.Context, externalapi.OperationName) (externalapi.BearerAuth, error) {
	return externalapi.BearerAuth{Token: string(b)}, nil
}

// NewWorkspace is the arrange step: a User, a Workspace and its first API token, made
// with internal/accounts and internal/apitokens (the bootstrap token is not used, it
// only addresses the oldest Workspace). It registers cleanup of the test's own mail.
func (e *Env) NewWorkspace(t testing.TB) *Workspace {
	t.Helper()
	ctx := t.Context()
	id := uniq()

	user, err := e.accounts.CreateUser(ctx, "E2E "+id, "owner-"+id+"@e2e.test", "!")
	require.NoError(t, err)
	ws, err := e.accounts.CreateWorkspace(ctx, user.ID, "e2e "+id)
	require.NoError(t, err)
	scoped, _, err := e.accounts.Scope(ctx, user.ID, ws.Slug)
	require.NoError(t, err)
	minted, err := apitokens.Mint(ctx, scoped, apitokens.Input{Name: "e2e", Scopes: tokenScopes})
	require.NoError(t, err)

	api, err := externalapi.NewClient(e.BaseURL+"/api", bearer(minted.Value))
	require.NoError(t, err)

	w := &Workspace{
		t: t, env: e, api: api,
		Domain:    id + ".e2e.test",
		FromEmail: "news@" + id + ".e2e.test",
		FromName:  "E2E News",
		scoped:    scoped,
	}
	t.Cleanup(func() {
		// Delete only this test's messages: other tests share the inbox.
		_ = e.Mailpit.Delete(context.WithoutCancel(ctx), w.msgIDs...)
	})
	return w
}

// Ready is the common setup: an Integration pointing at Mailpit and a verified Sending
// domain, so a Broadcast can leave. Scenarios that need less call the steps themselves.
func (w *Workspace) Ready() *Workspace {
	w.t.Helper()
	w.CreateMailpitIntegration()
	w.AddVerifiedSendingDomain()
	return w
}

// NewRecipient returns an address unique to this call, so mail can be matched to the test.
func (w *Workspace) NewRecipient() string {
	return "to-" + uniq() + "@recipients.e2e.test"
}

// ok unwraps an API result of the expected type or fails the test with whatever
// the API answered instead (an RFC 7807 problem, typically).
func ok[T any](t testing.TB, step string, res any, err error) *T {
	t.Helper()
	require.NoError(t, err, step)
	v, isT := res.(*T)
	require.True(t, isT, "%s: unexpected response %T: %+v", step, res, res)
	return v
}

// CreateMailpitIntegration creates the default SMTP Integration pointing at the
// suite's Mailpit, sending as FromName <FromEmail>.
func (w *Workspace) CreateMailpitIntegration() {
	w.t.Helper()
	cfg := externalapi.SmtpConfigInput{
		Kind:     externalapi.SmtpConfigInputKindSMTP,
		Host:     w.env.Mailpit.SMTPHost,
		Port:     int32(w.env.Mailpit.SMTPPort),
		From:     externalapi.EmailAddress(w.FromEmail),
		FromName: externalapi.NewOptNilString(w.FromName),
	}
	res, err := w.api.IntegrationsCreate(w.t.Context(), &externalapi.CreateIntegrationInput{
		Name:      "mailpit",
		IsDefault: externalapi.NewOptBool(true),
		Config:    externalapi.IntegrationConfigInput{OneOf: externalapi.NewSmtpConfigInputIntegrationConfigInputSum(cfg)},
	})
	ok[externalapi.IntegrationResource](w.t, "create integration", res, err)
}

// AddVerifiedSendingDomain creates the unique Sending domain, triggers verification
// (an asynchronous job) and polls until it reads as verified.
func (w *Workspace) AddVerifiedSendingDomain() {
	w.t.Helper()
	ctx := w.t.Context()
	res, err := w.api.SendingDomainsCreate(ctx, &externalapi.CreateSendingDomainInput{Domain: w.Domain})
	sd := ok[externalapi.SendingDomainResource](w.t, "create sending domain", res, err)

	_, err = w.api.SendingDomainsVerify(ctx, externalapi.SendingDomainsVerifyParams{ID: sd.ID})
	require.NoError(w.t, err, "verify sending domain") // 202: the check is a job; the poll below is the assertion

	require.Eventually(w.t, func() bool {
		got, err := w.api.SendingDomainsGet(ctx, externalapi.SendingDomainsGetParams{ID: sd.ID})
		if err != nil {
			return false
		}
		d, isD := got.(*externalapi.SendingDomainResource)
		return isD && d.Verified
	}, EmailTimeout, 100*time.Millisecond, "Sending domain %s never became verified", w.Domain)
}

// ImportContacts upserts a Contact per email through the batch endpoint.
func (w *Workspace) ImportContacts(emails ...string) {
	w.t.Helper()
	items := make([]externalapi.UpsertContactInput, len(emails))
	for i, e := range emails {
		items[i] = externalapi.UpsertContactInput{Email: externalapi.NewOptNilEmailAddress(externalapi.EmailAddress(e))}
	}
	res, err := w.api.ContactsBatchUpsert(w.t.Context(), &externalapi.UpsertContactsInput{Contacts: items})
	out := ok[externalapi.UpsertContactsResult](w.t, "import contacts", res, err)
	for _, r := range out.Results {
		require.NotEqual(w.t, externalapi.ContactBatchStatusFailed, r.Status, "import contact #%d: %s", r.Index, r.Error.Value)
	}
}

// Broadcast is what a scenario chooses about a Broadcast; the rest defaults.
type Broadcast struct {
	Name    string
	Subject string
	// Body is MJML (the product's one body format).
	Body string
}

// SendBroadcast creates a Broadcast from the Workspace's FromEmail/FromName, sets its
// audience to all active Contacts and schedules it for now. The send itself is an
// asynchronous job: observe it with WaitForEmail.
func (w *Workspace) SendBroadcast(b Broadcast) {
	w.t.Helper()
	ctx := w.t.Context()
	if b.Name == "" {
		b.Name = "e2e broadcast " + uniq()
	}
	res, err := w.api.BroadcastsCreate(ctx, &externalapi.CreateBroadcastInput{
		Name:      b.Name,
		Subject:   externalapi.NewOptString(b.Subject),
		Body:      externalapi.NewOptString(b.Body),
		FromName:  externalapi.NewOptString(w.FromName),
		FromEmail: externalapi.NewOptEmailAddress(externalapi.EmailAddress(w.FromEmail)),
	})
	created := ok[externalapi.BroadcastResource](w.t, "create broadcast", res, err)

	aud, err := w.api.BroadcastsSetAudience(ctx, &externalapi.SetBroadcastAudienceInput{SegmentId: externalapi.NilEntityId{Null: true}},
		externalapi.BroadcastsSetAudienceParams{ID: created.ID})
	ok[externalapi.BroadcastResource](w.t, "set audience", aud, err)

	sched, err := w.api.BroadcastsSchedule(ctx, &externalapi.ScheduleBroadcastInput{ScheduledAt: externalapi.Timestamp(time.Now())},
		externalapi.BroadcastsScheduleParams{ID: created.ID})
	ok[externalapi.BroadcastResource](w.t, "schedule broadcast", sched, err)
}

// WaitForEmail waits (bounded by EmailTimeout) for a message addressed to recipient and
// returns it with its headers. The failure message lists what the inbox held.
func (w *Workspace) WaitForEmail(recipient string) Message {
	w.t.Helper()
	return w.waitForEmail(recipient, "")
}

// waitForEmail is the shared wait behind WaitForEmail and WaitForEmailWithSubject
// (subject "" matches any); it remembers the message for cleanup.
func (w *Workspace) waitForEmail(recipient, subject string) Message {
	w.t.Helper()
	msg, err := w.env.Mailpit.WaitForMessage(w.t.Context(), recipient, subject, EmailTimeout)
	require.NoError(w.t, err)
	w.msgIDs = append(w.msgIDs, msg.ID)
	return msg
}

// String names the Workspace in failures.
func (w *Workspace) String() string { return fmt.Sprintf("e2e workspace %s", w.Domain) }

// AddOwner makes a second owner of the Workspace and returns the Membership ids of all
// its owners (the founder first).
func (w *Workspace) AddOwner() []int64 {
	w.t.Helper()
	ctx := w.t.Context()
	id := uniq()
	user, err := w.env.accounts.CreateUser(ctx, "E2E co-owner "+id, "co-owner-"+id+"@e2e.test", "!")
	require.NoError(w.t, err)
	_, err = w.scoped.Membership().Create().SetUserID(user.ID).SetRole(membership.RoleOwner).Save(ctx)
	require.NoError(w.t, err)
	return w.OwnerMembershipIDs()
}

// OwnerMembershipIDs lists the Membership ids of the Workspace's owners, ascending.
func (w *Workspace) OwnerMembershipIDs() []int64 {
	w.t.Helper()
	ids, err := w.scoped.Membership().Query().
		Where(membership.RoleEQ(membership.RoleOwner)).
		Order(ent.Asc(membership.FieldID)).
		IDs(w.t.Context())
	require.NoError(w.t, err)
	return ids
}

// DemoteConcurrently demotes each Membership to member, all at once, as an owner would,
// and returns the error of each attempt.
func (w *Workspace) DemoteConcurrently(ids ...int64) []error {
	w.t.Helper()
	errs := make([]error, len(ids))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Go(func() {
			<-start
			_, errs[i] = w.env.accounts.ChangeMembershipRole(w.t.Context(), w.scoped, events.Actor{Kind: events.ActorSystem},
				membership.RoleOwner, id, membership.RoleMember)
		})
	}
	close(start)
	wg.Wait()
	return errs
}
