//go:build e2e

package e2e

// MJML wraps text in the minimal MJML body (the product's one body format).
func MJML(text string) string {
	return "<mjml><mj-body><mj-section><mj-column><mj-text>" + text + "</mj-text></mj-column></mj-section></mj-body></mjml>"
}
