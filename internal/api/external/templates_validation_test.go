package external_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// Mapping only: the blank-name rule itself is tested at the templates module.
func TestExternalTemplatesMapABlankNameTo422(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.ExternalScoped(t, "templates:read", "templates:write")

	created, err := c.TemplatesCreate(context.Background(), &externalapi.CreateTemplateInput{Name: ""})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesCreateUnprocessableEntity{}, created)
}

func TestExternalTemplatesUpdateMapsABlankNameTo422(t *testing.T) {
	env := testhelper.Setup(t)
	c := env.ExternalScoped(t, "templates:read", "templates:write")
	tmpl := seedTemplate(t, env.DB, fixtures.AcmeID)

	upd, err := c.TemplatesUpdate(context.Background(), &externalapi.UpdateTemplateInput{Name: externalapi.NewOptString("")},
		externalapi.TemplatesUpdateParams{ID: templateID(tmpl)})
	require.NoError(t, err)
	assert.IsType(t, &externalapi.TemplatesUpdateUnprocessableEntity{}, upd)
}
