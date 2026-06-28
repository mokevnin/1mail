// Package emailrender renders broadcast email content: Liquid templating for
// merge tags (e.g. {{ first_name }}, conditionals) and HTML→plaintext for the
// text part. Both are thin wrappers over maintained libraries (osteele/liquid,
// k3a/html2text) so we don't hand-roll templating or HTML stripping.
package emailrender

import (
	"github.com/k3a/html2text"
	"github.com/osteele/liquid"
)

// engine is safe for concurrent ParseAndRender calls.
var engine = liquid.NewEngine()

// Render renders a Liquid template with the given bindings. On a template error
// it returns the original template unchanged together with the error, so a
// malformed template degrades gracefully (the caller logs it) instead of
// blocking the whole send.
func Render(tmpl string, bindings map[string]any) (string, error) {
	out, err := engine.ParseAndRenderString(tmpl, bindings)
	if err != nil {
		return tmpl, err
	}
	return out, nil
}

// HTMLToText derives a plain-text representation of an HTML body for the
// text/plain MIME part.
func HTMLToText(html string) string {
	return html2text.HTML2Text(html)
}
