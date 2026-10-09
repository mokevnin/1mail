package external_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/ent/emailtemplate"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// Isolated: the failed insert/update aborts this test's transaction at most once
// per statement, so the blank-name cases run last.
func TestExternalTemplatesRefuseABlankName(t *testing.T) {
	env := testhelper.Setup(t)
	ctx := context.Background()
	c := env.ExternalScoped(t, "templates:read", "templates:write")
	tmpl := seedTemplate(t, env.DB, fixtures.AcmeID)

	upd, err := c.TemplatesUpdate(ctx, &externalapi.UpdateTemplateInput{Name: externalapi.NewOptString("")},
		externalapi.TemplatesUpdateParams{ID: templateID(tmpl)})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesUpdateUnprocessableEntity{}, upd)
	assert.Equal(t, "Receipt", env.DB.EmailTemplate.GetX(ctx, tmpl.ID).Name, "the name is unchanged")

	created, err := c.TemplatesCreate(ctx, &externalapi.CreateTemplateInput{Name: ""})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesCreateUnprocessableEntity{}, created)
	n, err := env.DB.EmailTemplate.Query().Where(emailtemplate.WorkspaceID(fixtures.AcmeID), emailtemplate.Name("")).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, n)
}
