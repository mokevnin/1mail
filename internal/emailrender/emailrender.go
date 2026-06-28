// Package emailrender renders broadcast/template email content. The pipeline is:
// Liquid merge tags → (MJML compile when the body is MJML) → CSS inlining for
// plain HTML → derived plain-text part. Each step is a maintained library
// (osteele/liquid, preslavrachev/gomjml, vanng822/go-premailer, k3a/html2text)
// — we don't hand-roll templating, MJML, CSS inlining, or HTML stripping.
//
// Tracking (open pixel, click rewriting, unsubscribe footer) is deliberately NOT
// done here: the caller layers it on AFTER this pipeline so re-parsing/inlining
// can never mangle the injected pixel or escape the click-URL query strings.
package emailrender

import (
	"fmt"
	"strings"

	"github.com/k3a/html2text"
	"github.com/osteele/liquid"
	"github.com/preslavrachev/gomjml/mjml"
	"github.com/vanng822/go-premailer/premailer"
)

// Format is the authored body format of a broadcast/template.
const (
	FormatHTML = "html"
	FormatMJML = "mjml"
)

// engine is safe for concurrent ParseAndRender calls.
var engine = liquid.NewEngine()

// Email is a fully-rendered message ready for tracking + delivery.
type Email struct {
	Subject string
	HTML    string
	Text    string
}

// RenderEmail renders subject + body for one recipient. Liquid runs first (so a
// {% if %} can change MJML structure per recipient), then MJML compiles to
// email HTML, or plain HTML gets its <style> blocks inlined. The text part is
// derived from the compiled HTML (before any tracking is layered on).
//
// Liquid errors degrade gracefully (best-effort output). A non-nil error means
// the MJML body failed to compile — the caller should not send that message.
func RenderEmail(format, subject, body string, bindings map[string]any) (Email, error) {
	renderedSubject, _ := Render(subject, bindings)
	renderedBody, _ := Render(body, bindings)

	html := renderedBody
	if strings.EqualFold(format, FormatMJML) {
		out, err := mjml.Render(renderedBody)
		if err != nil {
			return Email{}, fmt.Errorf("compile mjml: %w", err)
		}
		html = out
	} else if inlined, err := inlineCSS(html); err == nil {
		// Best-effort: inlining <style> into attributes for plain-HTML emails.
		html = inlined
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

func inlineCSS(html string) (string, error) {
	p, err := premailer.NewPremailerFromString(html, premailer.NewOptions())
	if err != nil {
		return html, err
	}
	return p.Transform()
}
