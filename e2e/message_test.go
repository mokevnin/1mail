//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func messageWith(headers map[string][]string) Message {
	return Message{Headers: headers}
}

func TestMessageUnsubscribeURLPrefersHTTPOverMailto(t *testing.T) {
	t.Parallel()
	m := messageWith(map[string][]string{
		"List-Unsubscribe": {"<mailto:unsub@example.test?subject=u>, <https://app.example.test/u/abc>"},
	})
	assert.Equal(t, "https://app.example.test/u/abc", m.UnsubscribeURL())
}

func TestMessageUnsubscribeURLAcceptsPlainHTTP(t *testing.T) {
	t.Parallel()
	m := messageWith(map[string][]string{"List-Unsubscribe": {"<http://localhost/u/1>"}})
	assert.Equal(t, "http://localhost/u/1", m.UnsubscribeURL())
}

func TestMessageUnsubscribeURLIsEmptyWithoutHTTPTarget(t *testing.T) {
	t.Parallel()
	assert.Empty(t, messageWith(nil).UnsubscribeURL())
	assert.Empty(t, messageWith(map[string][]string{"List-Unsubscribe": {"<mailto:unsub@example.test>"}}).UnsubscribeURL())
}

func TestMessageOneClickBody(t *testing.T) {
	t.Parallel()
	m := messageWith(map[string][]string{"List-Unsubscribe-Post": {"List-Unsubscribe=One-Click"}})
	assert.Equal(t, "List-Unsubscribe=One-Click", m.OneClickBody())
	assert.Empty(t, messageWith(nil).OneClickBody())
}
