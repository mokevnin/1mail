//go:build !embed_spa

package server

import "io/fs"

// Default builds (go build ./..., go test, golangci-lint, local dev) do NOT
// embed the SPA, so they compile without a prior `pnpm build`. In dev the
// frontend is served separately by Vite; the Go server's catch-all route is
// only meaningful in a release build (-tags embed_spa).
func spaFS() (fs.FS, bool) { return nil, false }
