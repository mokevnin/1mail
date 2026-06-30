package tracking_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenRoundTrip(t *testing.T) {
	tr := tracking.New("secret", "https://app.test")
	tok, err := tr.Token(42)
	require.NoError(t, err)

	rid, err := tr.Decode(tok)
	require.NoError(t, err)
	assert.Equal(t, int64(42), rid)
}

func TestDecodeRejectsWrongSecret(t *testing.T) {
	tok, err := tracking.New("secret", "https://app.test").Token(1)
	require.NoError(t, err)

	_, err = tracking.New("other-secret", "https://app.test").Decode(tok)
	assert.Error(t, err)
}

// unsubToken↔DecodeUnsub round-trips every field, including a large int64 id that
// would lose precision if stored as a JSON number (float64) instead of a string.
func TestUnsubTokenRoundTrip(t *testing.T) {
	tr := tracking.New("secret", "https://app.test")
	target := tracking.UnsubTarget{
		Source:      "automation:7",
		Destination: "person@example.com",
		WorkspaceID: 1,
		ContactID:   9007199254740993, // 2^53 + 1: not representable as float64
		BroadcastID: 0,
	}
	urlStr, err := tr.UnsubscribeURL(target)
	require.NoError(t, err)
	assert.Contains(t, urlStr, "https://app.test/e/u/")

	token := strings.TrimPrefix(urlStr, "https://app.test/e/u/")
	got, err := tr.DecodeUnsub(token)
	require.NoError(t, err)
	assert.Equal(t, target, got)
}

func TestRewriteWrapsLinksAndAddsPixelAndFooter(t *testing.T) {
	tr := tracking.New("secret", "https://app.test")
	body := `<p>Hello <a href="https://dest.test/path?x=1">click</a> or <a href="mailto:a@b.test">mail</a></p>`

	out, err := tr.Rewrite(body, 7, tracking.UnsubTarget{
		Source: "broadcasts", Destination: "x@y.test", WorkspaceID: 1, ContactID: 2, BroadcastID: 3,
	})
	require.NoError(t, err)

	// http link routed through the click tracker, with the destination preserved.
	assert.Contains(t, out, "https://app.test/e/c/")
	assert.Contains(t, out, url.QueryEscape("https://dest.test/path?x=1"))
	// mailto links are left untouched.
	assert.Contains(t, out, `href="mailto:a@b.test"`)
	// open pixel + unsubscribe footer appended.
	assert.Contains(t, out, "https://app.test/e/o/")
	assert.Contains(t, out, "https://app.test/e/u/")
	assert.Contains(t, strings.ToLower(out), "unsubscribe")
}
