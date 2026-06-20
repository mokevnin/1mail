package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spaHandler serves the embedded single-page app. Real files (index.html, the
// hashed /assets/* bundles) are served directly; any other path falls back to
// index.html so client-side routing works on deep links / refreshes.
//
// It is mounted on the catch-all "/" pattern, so it only sees requests not
// matched by a more specific handler (/site, /api, /collect, /auth, /avatar,
// /t.js) — Go 1.22+ ServeMux picks the most specific pattern.
//
// When the SPA is not embedded (default build, no embed_spa tag) it returns a
// hint instead; the frontend is served by Vite in dev.
func spaHandler() http.Handler {
	sub, ok := spaFS()
	if !ok {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusNotImplemented)
			_, _ = w.Write([]byte("frontend not embedded in this build (build with -tags embed_spa); in dev it is served by Vite\n"))
		})
	}

	fileServer := http.FileServerFS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		// Serve the file when it exists; otherwise treat the path as a
		// client-side route and return the app shell.
		if f, err := sub.Open(name); err == nil {
			info, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		serveIndex(w, r, sub)
	})
}

func serveIndex(w http.ResponseWriter, _ *http.Request, sub fs.FS) {
	b, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}
