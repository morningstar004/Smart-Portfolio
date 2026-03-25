// Package docs provides an embedded Swagger UI handler that serves the OpenAPI
// specification bundled with the application binary. No external dependencies
// or CDN loads are required at runtime — everything is self-contained.
//
// Usage:
//
//	r.Get("/docs", docs.SwaggerRedirect)
//	r.Get("/docs/*", docs.SwaggerHandler("/docs"))
package docs

import (
	_ "embed"
	"net/http"
	"path"
	"strings"
)

//go:embed openapi.yaml
var openapiSpec []byte

// swaggerUIVersion is the Swagger UI release served from the jsDelivr CDN.
// Using a CDN for the UI assets keeps the binary small while the OpenAPI spec
// itself is embedded and served locally.
const swaggerUIVersion = "5.17.14"

// SwaggerRedirect sends a 301 redirect from /docs to /docs/ so the relative
// asset paths inside the HTML page resolve correctly.
func SwaggerRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
}

// SwaggerHandler returns an http.HandlerFunc that serves:
//   - GET {basePath}/           → Swagger UI HTML page
//   - GET {basePath}/openapi.yaml → the embedded OpenAPI spec
//
// The basePath parameter should match the route prefix where the handler is
// mounted (e.g. "/docs").
func SwaggerHandler(basePath string) http.HandlerFunc {
	// Pre-render the HTML once at startup.
	html := buildSwaggerHTML(basePath)

	return func(w http.ResponseWriter, r *http.Request) {
		// Strip the base path to get the relative path.
		rel := strings.TrimPrefix(r.URL.Path, basePath)
		rel = strings.TrimPrefix(rel, "/")

		switch {
		case rel == "" || rel == "index.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))

		case rel == "openapi.yaml":
			w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=300")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(openapiSpec)

		default:
			http.NotFound(w, r)
		}
	}
}

// buildSwaggerHTML returns a self-contained HTML page that loads Swagger UI
// from a CDN and points it at the locally-served OpenAPI spec.
func buildSwaggerHTML(basePath string) string {
	specURL := path.Join(basePath, "openapi.yaml")

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Smart Portfolio API — Docs</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@` + swaggerUIVersion + `/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
    .topbar { display: none !important; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@` + swaggerUIVersion + `/swagger-ui-bundle.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@` + swaggerUIVersion + `/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = () => {
      SwaggerUIBundle({
        url: "` + specURL + `",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset,
        ],
        plugins: [
          SwaggerUIBundle.plugins.DownloadUrl,
        ],
        layout: "StandaloneLayout",
        defaultModelsExpandDepth: 1,
        defaultModelExpandDepth: 2,
        docExpansion: "list",
        filter: true,
        showExtensions: true,
        showCommonExtensions: true,
        tryItOutEnabled: true,
      });
    };
  </script>
</body>
</html>`
}
