package emailrender_test

import (
	"strings"
	"testing"

	"github.com/mokevnin/1mail/internal/emailrender"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A representative MJML template exercising the common components.
const sampleMJML = `<mjml>
  <mj-head><mj-style>.brand { color: #ff0000; }</mj-style></mj-head>
  <mj-body>
    <mj-section>
      <mj-column>
        <mj-text>Hello {{ first_name }}, welcome!</mj-text>
        <mj-button href="https://example.com/sale">Shop now</mj-button>
      </mj-column>
    </mj-section>
  </mj-body>
</mjml>`

func TestRenderEmailMJML(t *testing.T) {
	out, err := emailrender.RenderEmail(emailrender.FormatMJML, "Hi {{ first_name }}", sampleMJML, map[string]any{
		"first_name": "Alex",
	})
	require.NoError(t, err)

	assert.Equal(t, "Hi Alex", out.Subject)
	// Merge tag resolved and MJML compiled to table-based email HTML.
	assert.Contains(t, out.HTML, "Alex")
	assert.NotContains(t, out.HTML, "{{")
	assert.Contains(t, strings.ToLower(out.HTML), "<table")
	assert.Contains(t, out.HTML, "https://example.com/sale")
	assert.NotEmpty(t, out.Text)
}

func TestRenderEmailHTMLInlinesCSS(t *testing.T) {
	body := `<html><head><style>.greeting { color: red; }</style></head>` +
		`<body><p class="greeting">Hello {{ first_name }}</p></body></html>`
	out, err := emailrender.RenderEmail(emailrender.FormatHTML, "Subject", body, map[string]any{
		"first_name": "Sam",
	})
	require.NoError(t, err)

	assert.Contains(t, out.HTML, "Hello Sam")
	// premailer inlined the rule into a style attribute.
	assert.Contains(t, out.HTML, "style=")
	assert.Contains(t, strings.ToLower(out.HTML), "color")
	assert.Contains(t, out.Text, "Hello Sam")
}

func TestRenderEmailInvalidMJMLFails(t *testing.T) {
	_, err := emailrender.RenderEmail(emailrender.FormatMJML, "s", "<mjml><not-valid", nil)
	assert.Error(t, err)
}
