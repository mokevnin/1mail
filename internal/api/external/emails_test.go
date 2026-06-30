package external_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/go-faster/jx"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/ent/transactionalemail"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mjmlBody = `<mjml><mj-body><mj-section><mj-column><mj-text>Hi {{ name }}, your code is {{ code }}</mj-text></mj-column></mj-section></mj-body></mjml>`

func seedTemplate(t *testing.T, db *ent.Client, workspaceID int64) *ent.EmailTemplate {
	t.Helper()
	tmpl, err := db.EmailTemplate.Create().
		SetWorkspaceID(workspaceID).
		SetName("Receipt").
		SetSubject("Hello {{ name }}").
		SetBody(mjmlBody).
		Save(context.Background())
	require.NoError(t, err)
	return tmpl
}

func templateID(tmpl *ent.EmailTemplate) externalapi.EntityId {
	return externalapi.EntityId(strconv.FormatInt(tmpl.ID, 10))
}

// A transactional send renders the referenced template with per-call variables
// and is accepted by the (capturing) provider.
func TestExternalEmailsSend(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"emails:send"}))
	ctx := context.Background()
	tmpl := seedTemplate(t, env.DB, 1)

	res, err := c.EmailsSend(ctx, &externalapi.SendTransactionalEmailInput{
		TemplateId:  templateID(tmpl),
		Destination: "Customer@Example.com",
		Variables: externalapi.NewOptSendTransactionalEmailInputVariables(
			externalapi.SendTransactionalEmailInputVariables{
				"name": jx.Raw(`"Ada"`),
				"code": jx.Raw(`123456`),
			}),
	}, externalapi.EmailsSendParams{})
	require.NoError(t, err)
	ok, isOK := res.(*externalapi.SendTransactionalEmailResponse)
	require.Truef(t, isOK, "got %T", res)
	assert.Equal(t, externalapi.TransactionalSendStatusSent, ok.Status)
	assert.Equal(t, "customer@example.com", ok.Destination, "destination normalized")
	assert.NotEmpty(t, ok.ID, "the durable send record id is returned")

	// The capturing provider received the rendered message — template content is
	// referenced and rendered at send time, variables merged, no raw braces.
	msgs := env.CustomerMail.Messages()
	require.Len(t, msgs, 1)
	assert.Equal(t, "customer@example.com", msgs[0].To)
	assert.Equal(t, "Hello Ada", msgs[0].Subject)
	assert.Contains(t, msgs[0].HTML, "Ada")
	assert.Contains(t, msgs[0].HTML, "123456")
	assert.NotContains(t, msgs[0].HTML, "{{")
}

// A suppressed destination is skipped (Suppression is the hard floor on every
// surface) and reported as status "suppressed" — not an error.
func TestExternalEmailsSendSuppressed(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"emails:send"}))
	ctx := context.Background()
	tmpl := seedTemplate(t, env.DB, 1)

	_, err := env.DB.Suppression.Create().
		SetWorkspaceID(1).
		SetChannel(suppression.ChannelEmail).
		SetDestination("blocked@example.com").
		SetReason(suppression.ReasonBounce).Save(ctx)
	require.NoError(t, err)

	res, err := c.EmailsSend(ctx, &externalapi.SendTransactionalEmailInput{
		TemplateId:  templateID(tmpl),
		Destination: "blocked@example.com",
	}, externalapi.EmailsSendParams{})
	require.NoError(t, err)
	ok, isOK := res.(*externalapi.SendTransactionalEmailResponse)
	require.Truef(t, isOK, "got %T", res)
	assert.Equal(t, externalapi.TransactionalSendStatusSuppressed, ok.Status)
	assert.Empty(t, env.CustomerMail.Messages(), "suppressed destination is not sent to")
}

// Transactional skips Unsubscribe: an "everything" opt-out does NOT block a
// transactional send (you cannot unsubscribe from your own password reset).
func TestExternalEmailsSendIgnoresUnsubscribe(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"emails:send"}))
	ctx := context.Background()
	tmpl := seedTemplate(t, env.DB, 1)

	_, err := env.DB.Unsubscribe.Create().
		SetWorkspaceID(1).
		SetChannel("email").
		SetDestination("optout@example.com").
		SetSendingSource(eligibility.SourceEverything).Save(ctx)
	require.NoError(t, err)

	res, err := c.EmailsSend(ctx, &externalapi.SendTransactionalEmailInput{
		TemplateId:  templateID(tmpl),
		Destination: "optout@example.com",
	}, externalapi.EmailsSendParams{})
	require.NoError(t, err)
	ok := res.(*externalapi.SendTransactionalEmailResponse)
	assert.Equal(t, externalapi.TransactionalSendStatusSent, ok.Status)
	assert.Len(t, env.CustomerMail.Messages(), 1)
}

// Multi-tenant isolation: a token cannot send with another workspace's template.
func TestExternalEmailsSendCrossWorkspaceTemplate(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()

	ws2, err := env.DB.Workspace.Create().
		SetName("Globex").SetSlug("globex-emails").
		SetCollectKey("globex-ck").SetIngestKey("globex-ik").Save(ctx)
	require.NoError(t, err)
	otherTmpl := seedTemplate(t, env.DB, ws2.ID)

	// Token belongs to workspace 1; referencing ws2's template id must 404.
	c := client(t, env, seedToken(t, env.DB, []string{"emails:send"}))
	res, err := c.EmailsSend(ctx, &externalapi.SendTransactionalEmailInput{
		TemplateId:  templateID(otherTmpl),
		Destination: "x@example.com",
	}, externalapi.EmailsSendParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.EmailsSendNotFound{}, res)
	assert.Empty(t, env.CustomerMail.Messages())
}

func TestExternalEmailsSendMissingTemplate(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"emails:send"}))

	res, err := c.EmailsSend(context.Background(), &externalapi.SendTransactionalEmailInput{
		TemplateId:  "999999",
		Destination: "x@example.com",
	}, externalapi.EmailsSendParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.EmailsSendNotFound{}, res)
}

// emails:send is required.
func TestExternalEmailsSendScope(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"contacts:read"}))
	tmpl := seedTemplate(t, env.DB, 1)

	res, err := c.EmailsSend(context.Background(), &externalapi.SendTransactionalEmailInput{
		TemplateId:  templateID(tmpl),
		Destination: "x@example.com",
	}, externalapi.EmailsSendParams{})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.EmailsSendUnauthorized{}, res)
}

// A successful send writes a durable record (the trace), resolves the destination
// to a contact when one exists, and publishes email.sent onto the outbox so the
// send is segmentable like broadcast/automation sends.
func TestExternalEmailsSendRecordsTrace(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"emails:send"}))
	ctx := context.Background()
	tmpl := seedTemplate(t, env.DB, 1)

	contact, err := env.DB.Contact.Create().SetWorkspaceID(1).SetEmail("trace@example.com").Save(ctx)
	require.NoError(t, err)

	res, err := c.EmailsSend(ctx, &externalapi.SendTransactionalEmailInput{
		TemplateId:  templateID(tmpl),
		Destination: "trace@example.com",
	}, externalapi.EmailsSendParams{})
	require.NoError(t, err)
	require.IsType(t, &externalapi.SendTransactionalEmailResponse{}, res)

	rec, err := env.DB.TransactionalEmail.Query().
		Where(transactionalemail.WorkspaceID(1)).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, transactionalemail.StatusSent, rec.Status)
	assert.Equal(t, "trace@example.com", rec.Destination)
	assert.Equal(t, tmpl.ID, rec.TemplateID)
	require.NotNil(t, rec.ContactID)
	assert.Equal(t, contact.ID, *rec.ContactID, "destination resolved to the contact")

	var sent int
	require.NoError(t, env.SQLDB.QueryRow(
		`SELECT count(*) FROM watermill_domain_events
		   WHERE payload->>'name' = 'email.sent' AND payload->'data'->>'email' = $1`,
		"trace@example.com",
	).Scan(&sent))
	assert.Equal(t, 1, sent, "email.sent published once")
}

// A repeated Idempotency-Key replays the original outcome without a second send.
func TestExternalEmailsSendIdempotentReplay(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"emails:send"}))
	ctx := context.Background()
	tmpl := seedTemplate(t, env.DB, 1)

	send := func() *externalapi.SendTransactionalEmailResponse {
		res, err := c.EmailsSend(ctx, &externalapi.SendTransactionalEmailInput{
			TemplateId:  templateID(tmpl),
			Destination: "once@example.com",
		}, externalapi.EmailsSendParams{IdempotencyKey: externalapi.NewOptString("key-1")})
		require.NoError(t, err)
		ok, isOK := res.(*externalapi.SendTransactionalEmailResponse)
		require.Truef(t, isOK, "got %T", res)
		return ok
	}

	first := send()
	second := send()

	assert.Equal(t, first.ID, second.ID, "same key replays the same record")
	assert.Equal(t, externalapi.TransactionalSendStatusSent, second.Status)
	assert.Len(t, env.CustomerMail.Messages(), 1, "the email is sent exactly once")

	n, err := env.DB.TransactionalEmail.Query().Where(transactionalemail.WorkspaceID(1)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "one record for the key")
}

// A suppressed send still records a row and honors the idempotency key, but emits
// no email and no email.sent event.
func TestExternalEmailsSendSuppressedRecordsNoEvent(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"emails:send"}))
	ctx := context.Background()
	tmpl := seedTemplate(t, env.DB, 1)

	_, err := env.DB.Suppression.Create().
		SetWorkspaceID(1).SetChannel(suppression.ChannelEmail).
		SetDestination("blocked@example.com").SetReason(suppression.ReasonBounce).Save(ctx)
	require.NoError(t, err)

	res, err := c.EmailsSend(ctx, &externalapi.SendTransactionalEmailInput{
		TemplateId:  templateID(tmpl),
		Destination: "blocked@example.com",
	}, externalapi.EmailsSendParams{IdempotencyKey: externalapi.NewOptString("sup-1")})
	require.NoError(t, err)
	ok := res.(*externalapi.SendTransactionalEmailResponse)
	assert.Equal(t, externalapi.TransactionalSendStatusSuppressed, ok.Status)

	rec, err := env.DB.TransactionalEmail.Query().Where(transactionalemail.WorkspaceID(1)).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, transactionalemail.StatusSuppressed, rec.Status)

	var events int
	require.NoError(t, env.SQLDB.QueryRow(
		`SELECT count(*) FROM watermill_domain_events
		   WHERE payload->>'name' = 'email.sent' AND payload->'data'->>'email' = $1`,
		"blocked@example.com",
	).Scan(&events))
	assert.Zero(t, events, "no send happened, so no email.sent")
}

// An in-flight key (a pending record) returns 409 rather than sending again.
func TestExternalEmailsSendConflictOnPending(t *testing.T) {
	env := testhelper.Setup(t)
	c := client(t, env, seedToken(t, env.DB, []string{"emails:send"}))
	ctx := context.Background()
	tmpl := seedTemplate(t, env.DB, 1)

	// Simulate a concurrent request that already claimed the key and is still sending.
	_, err := env.DB.TransactionalEmail.Create().
		SetWorkspaceID(1).SetChannel(transactionalemail.ChannelEmail).
		SetDestination("busy@example.com").SetTemplateID(tmpl.ID).
		SetStatus(transactionalemail.StatusPending).SetIdempotencyKey("inflight").Save(ctx)
	require.NoError(t, err)

	res, err := c.EmailsSend(ctx, &externalapi.SendTransactionalEmailInput{
		TemplateId:  templateID(tmpl),
		Destination: "busy@example.com",
	}, externalapi.EmailsSendParams{IdempotencyKey: externalapi.NewOptString("inflight")})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.EmailsSendConflict{}, res)
	assert.Empty(t, env.CustomerMail.Messages())
}
