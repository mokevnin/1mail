package external_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/apitoken"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// tokenActor is the actor of the one fresh token `ExternalScoped` just minted.
func tokenActor(t *testing.T, env *testhelper.TestEnv) events.Actor {
	t.Helper()
	tok, err := env.DB.ApiToken.Query().Where(apitoken.Name("actor-token"), apitoken.WorkspaceID(fixtures.AcmeID)).Only(context.Background())
	require.NoError(t, err)
	return events.Actor{Kind: events.ActorAPIToken, ID: strconv.FormatInt(tok.ID, 10), Name: tok.Name}
}

// Integration writes through /api are audited under the token, and the credentials never
// reach the entry.
func TestExternalIntegrationWritesAreAuditedUnderTheToken(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	c := env.ExternalScoped(t, "integrations:read", "integrations:write")

	created := createSMTP(t, c, "Audited mailer", false)
	row, err := env.DB.Scoped(fixtures.AcmeID).Integration().Get(ctx, mustID(t, created.ID))
	require.NoError(t, err)

	_, err = c.IntegrationsUpdate(ctx, &externalapi.UpdateIntegrationInput{Config: externalapi.NewOptNilIntegrationConfigInput(smtpInput("smtp2.example.com"))},
		externalapi.IntegrationsUpdateParams{ID: created.ID})
	require.NoError(t, err)
	updated, err := env.DB.Scoped(fixtures.AcmeID).Integration().Get(ctx, mustID(t, created.ID))
	require.NoError(t, err)
	require.NotEqual(t, row.ConfigEncrypted, updated.ConfigEncrypted, "the update sealed a new config")
	del, err := c.IntegrationsDelete(ctx, externalapi.IntegrationsDeleteParams{ID: created.ID})
	require.NoError(t, err)
	require.IsType(t, &externalapi.IntegrationsDeleteNoContent{}, del)

	got := env.OutboxEvents(t, events.NameAuditEntry)
	require.Len(t, got, 3)
	var actions []string
	for _, ev := range got {
		e := ev.(*events.AuditEntry)
		actions = append(actions, e.Action)
		assert.Equal(t, tokenActor(t, env), e.Actor)
		assert.Equal(t, strconv.FormatInt(mustID(t, created.ID), 10), e.TargetID)
		raw, err := json.Marshal(e)
		require.NoError(t, err)
		assert.NotContains(t, string(raw), smtpSecret)
		assert.NotContains(t, string(raw), row.ConfigEncrypted)
		assert.NotContains(t, string(raw), updated.ConfigEncrypted)
		if v, ok := e.Diff["config_encrypted"]; ok {
			assert.Equal(t, "changed", v, "a sensitive field is redacted")
		}
	}
	assert.Equal(t, []string{"integration.create", "integration.update", "integration.delete"}, actions)
	assert.Equal(t, "Audited mailer", got[0].(*events.AuditEntry).TargetName)
}

// Sending domain writes through /api are audited under the token, and the DKIM key never
// reaches the entry.
func TestExternalSendingDomainWritesAreAuditedUnderTheToken(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	c := env.ExternalScoped(t, "sending_domains:read", "sending_domains:write")

	created := createDomain(t, c, externalapi.CreateSendingDomainInput{Domain: "audited.example.com"})
	row, err := env.DB.Scoped(fixtures.AcmeID).SendingDomain().Get(ctx, mustID(t, created.ID))
	require.NoError(t, err)

	_, err = c.SendingDomainsUpdate(ctx, &externalapi.UpdateSendingDomainInput{DkimSelector: externalapi.NewOptString("s2")},
		externalapi.SendingDomainsUpdateParams{ID: created.ID})
	require.NoError(t, err)
	updated, err := env.DB.Scoped(fixtures.AcmeID).SendingDomain().Get(ctx, mustID(t, created.ID))
	require.NoError(t, err)
	del, err := c.SendingDomainsDelete(ctx, externalapi.SendingDomainsDeleteParams{ID: created.ID})
	require.NoError(t, err)
	require.IsType(t, &externalapi.SendingDomainsDeleteNoContent{}, del)

	got := env.OutboxEvents(t, events.NameAuditEntry)
	require.Len(t, got, 3)
	var actions []string
	for _, ev := range got {
		e := ev.(*events.AuditEntry)
		actions = append(actions, e.Action)
		assert.Equal(t, tokenActor(t, env), e.Actor)
		assert.Equal(t, strconv.FormatInt(mustID(t, created.ID), 10), e.TargetID)
		assert.Equal(t, "audited.example.com", e.TargetName)
		raw, err := json.Marshal(e)
		require.NoError(t, err)
		assert.NotContains(t, string(raw), "PRIVATE KEY")
		assert.NotContains(t, string(raw), row.DkimPrivateKeyEncrypted)
		assert.NotContains(t, string(raw), updated.DkimPrivateKeyEncrypted)
		if v, ok := e.Diff["dkim_private_key_encrypted"]; ok {
			assert.Equal(t, "changed", v, "a sensitive field is redacted")
		}
	}
	assert.Equal(t, []string{"sending_domain.create", "sending_domain.update", "sending_domain.delete"}, actions)
}
