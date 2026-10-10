//go:build e2e

package e2e

import (
	"net/http"
	"regexp"
	"strings"
)

// angleURL picks the <...> entries of a List-Unsubscribe value (RFC 2369).
var angleURL = regexp.MustCompile(`<([^>]*)>`)

// Message is a delivered email with the fields the scenarios assert on.
type Message struct {
	ID      string
	From    Address
	To      []Address
	Subject string
	Text    string
	HTML    string
	// Headers are the raw header values keyed by canonical name, so a scenario can
	// read List-Unsubscribe and List-Unsubscribe-Post exactly as they went on the wire.
	Headers map[string][]string
}

// Header returns the first value of the named header ("" when absent).
func (m Message) Header(name string) string {
	return first(m.Headers[http.CanonicalHeaderKey(name)])
}

func first(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// UnsubscribeURL is the http(s) target in List-Unsubscribe (preferred over mailto:),
// or "" when the header is absent or carries none.
func (m Message) UnsubscribeURL() string {
	for _, e := range angleURL.FindAllStringSubmatch(m.Header("List-Unsubscribe"), -1) {
		if strings.HasPrefix(e[1], "https://") || strings.HasPrefix(e[1], "http://") {
			return e[1]
		}
	}
	return ""
}

// OneClickBody is the List-Unsubscribe-Post value (RFC 8058), the form body of the
// one-click request; "" when the header is absent.
func (m Message) OneClickBody() string {
	return m.Header("List-Unsubscribe-Post")
}
