// Package docs embeds the OpenAPI specification so the binary is self-contained
// and can serve the API docs without reading from disk.
package docs

import _ "embed"

//go:embed openapi.yaml
var OpenAPISpec []byte
