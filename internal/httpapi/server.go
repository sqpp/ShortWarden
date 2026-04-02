package httpapi

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	genapi "shortwarden/api/gen"
	"shortwarden/internal/config"
)

func NewServer(cfg config.Config, db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// Screenshot preview proxy can take ~25s (headless Chromium); keep headroom.
	r.Use(middleware.Timeout(55 * time.Second))

	h := NewHandler(cfg, db)
	r.Use(authMiddleware(authDeps{db: db, q: h.q, jwt: h.jwt}))

	// Public image proxy for redirect interstitials (browser must not use internal Docker hostnames).
	r.Get("/preview/screenshot", h.ScreenshotPreview)

	openapiYAML, err := os.ReadFile("api/openapi.yaml")
	if err != nil {
		log.Printf("warning: failed to read api/openapi.yaml: %v", err)
		openapiYAML = []byte("openapi: 3.0.3\ninfo:\n  title: ShortWarden API\n  version: 0.0.0\n")
	}
	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(openapiYAML)
	})
	r.Get("/docs", swaggerUIHandler())
	r.Get("/docs/", swaggerUIHandler())

	// Generated OpenAPI router.
	handler := genapi.HandlerFromMux(h, r)
	_ = handler

	// Non-OpenAPI internal system endpoints (authenticated + CSRF protected in handlers).
	r.Post("/v1/system/update", h.TriggerUpdate)
	r.Get("/v1/system/update", h.GetUpdateStatus)
	r.Get("/v1/system/version", h.GetSystemVersion)
	r.Get("/v1/system/latest-version", h.GetLatestVersion)
	r.Get("/v1/me/redirect-customization", h.GetRedirectCustomization)
	r.Patch("/v1/me/redirect-customization", h.UpdateRedirectCustomization)

	return r
}

func swaggerUIHandler() http.HandlerFunc {
	const html = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>ShortWarden API Docs</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css" />
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: "/openapi.yaml",
        dom_id: "#swagger-ui",
        deepLinking: true,
        persistAuthorization: true
      });
    </script>
  </body>
</html>`
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}
}

