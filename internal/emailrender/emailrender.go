// Package emailrender renders broadcast/template email content. Bodies are
// authored as MJML; the pipeline is: Liquid merge tags → MJML compile (which
// emits email-safe, inlined HTML) → derived plain-text part. Each step is a
// maintained library (osteele/liquid, preslavrachev/gomjml, k3a/html2text) — we
// don't hand-roll templating, MJML, or HTML stripping.
//
// Tracking (open pixel, click rewriting, unsubscribe footer) is deliberately NOT
// done here: the caller layers it on AFTER this pipeline so re-parsing can never
// mangle the injected pixel or escape the click-URL query strings.
package emailrender

import (
	"fmt"

	"github.com/k3a/html2text"
	"github.com/osteele/liquid"
	"github.com/preslavrachev/gomjml/mjml"
)

// engine is safe for concurrent ParseAndRender calls.
var engine = liquid.NewEngine()

// Email is a fully-rendered message ready for tracking + delivery.
type Email struct {
	Subject string
	HTML    string
	Text    string
}

// RenderEmail renders subject + MJML body for one recipient: Liquid merge tags
// first (so a {% if %} can change MJML structure per recipient), then MJML
// compiles to email HTML, then the text part is derived from that HTML (before
// any tracking is layered on). A non-nil error means the MJML failed to compile
// — the caller should not send that message.
func RenderEmail(subject, body string, bindings map[string]any) (Email, error) {
	renderedSubject, _ := Render(subject, bindings)
	renderedBody, _ := Render(body, bindings)

	html, err := mjml.Render(renderedBody)
	if err != nil {
		return Email{}, fmt.Errorf("compile mjml: %w", err)
	}

	return Email{Subject: renderedSubject, HTML: html, Text: HTMLToText(html)}, nil
}

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
