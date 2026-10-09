// Package onemail holds repository-root assets that Go code embeds.
package onemail

import _ "embed"

// ExternalOpenAPI is the generated external (/api) OpenAPI document. The MCP
// surface (internal/mcpserver) projects its tools from it, so the tool list can
// never drift from the contract (ADR 0016).
//
//go:embed openapi/external.openapi.json
var ExternalOpenAPI []byte
