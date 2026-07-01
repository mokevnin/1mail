package server

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestBuildIndexShellInjectsLocale(t *testing.T) {
	fs := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(
			`<html lang="{{APP_LOCALE}}"><script>window.__APP_LOCALE__ = '{{APP_LOCALE}}'</script></html>`,
		)},
	}

	shell := buildIndexShell(fs, "ru")
	got := string(shell)

	if strings.Contains(got, localeSentinel) {
		t.Fatalf("sentinel %q was not substituted: %s", localeSentinel, got)
	}
	// The value slots become "ru", but the window.__APP_LOCALE__ identifier must
	// survive intact (the sentinel is chosen not to collide with it).
	if !strings.Contains(got, `lang="ru"`) || !strings.Contains(got, `window.__APP_LOCALE__ = 'ru'`) {
		t.Fatalf("locale not injected into both slots: %s", got)
	}
}

func TestBuildIndexShellMissingFile(t *testing.T) {
	if shell := buildIndexShell(fstest.MapFS{}, "en"); shell != nil {
		t.Fatalf("expected nil shell when index.html is absent, got %q", shell)
	}
}
