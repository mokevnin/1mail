//go:build e2e

package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/apitokens"
)

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
// external API. Everything is Workspace-scoped, so tests are
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

	// Inbox is the only way a scenario observes mail.
	Inbox *Inbox
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
// only addresses the oldest Workspace). Its Inbox deletes the test's own mail on cleanup.
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
	}
	w.Inbox = newInbox(t, e.mailpit)
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
		Host:     w.env.mailpit.SMTPHost,
		Port:     int32(w.env.mailpit.SMTPPort),
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
// asynchronous job: observe it with Inbox.Wait.
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

// String names the Workspace in failures.
func (w *Workspace) String() string { return fmt.Sprintf("e2e workspace %s", w.Domain) }

// mjml wraps text in the minimal MJML body (the product's one body format).
func mjml(text string) string {
	return "<mjml><mj-body><mj-section><mj-column><mj-text>" + text + "</mj-text></mj-column></mj-section></mj-body></mjml>"
}
