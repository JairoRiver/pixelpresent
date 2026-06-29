package api

import (
	"net/http"

	"github.com/JairoRiver/pixelpresent/docs"
)

// docsHTML renders the embedded OpenAPI spec with Scalar. The renderer script is
// loaded from a CDN, which is acceptable because docs are a development-only
// affordance (see EnableDocs).
const docsHTML = `<!doctype html>
<html>
  <head>
    <title>Pixel Present API</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <script id="api-reference" data-url="/docs/openapi.yaml"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`

// EnableDocs mounts the API documentation routes on the next call to Routes. It
// is a development-only toggle: serve calls it only when environment is not
// production, so the docs add no attack surface in production. It is kept out of
// NewServer because it is an optional wiring-time switch, not a dependency, and
// adding it there would churn every NewServer call site.
func (s *Server) EnableDocs() {
	s.docsEnabled = true
}

func (s *Server) handleDocsUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(docsHTML))
}

func (s *Server) handleOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(docs.OpenAPISpec)
}
