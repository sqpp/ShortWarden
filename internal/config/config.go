package config

import (
	"errors"
	"os"
)

type Config struct {
	HTTPAddr string
	PostgresDSN string
	JWTSecret string

	// PublicBaseURL is the canonical base used for displaying short links
	// when a user has no primary domain set (e.g. https://sw.example.com).
	PublicBaseURL string

	// CookieDomain can be empty to default to host-only cookies in dev.
	CookieDomain string
	CookieSecure bool

	// CORSAllowedOrigins is a comma-separated list of origins (scheme+host+port)
	// allowed to send credentialed browser requests (for the Vue dev server).
	CORSAllowedOrigins string

	// ScreenshotPreviewURLTemplate is optional. When set, it overrides the default preview URL.
	// Must contain "{url}" (URL-encoded destination). For the bundled screenshotd stack, leave empty
	// and set ScreenshotServiceURL instead.
	ScreenshotPreviewURLTemplate string

	// ScreenshotServiceURL is the internal base URL of screenshotd (e.g. http://screenshot:8088).
	// When set, the public preview path is PublicBaseURL + "/preview/screenshot?url=…" (proxied by the API).
	ScreenshotServiceURL string
}

func FromEnv() (Config, error) {
	cfg := Config{
		HTTPAddr:                     envOr("SHORTWARDEN_HTTP_ADDR", ":8080"),
		PostgresDSN:                  envOr("SHORTWARDEN_POSTGRES_DSN", ""),
		JWTSecret:                    envOr("SHORTWARDEN_JWT_SECRET", ""),
		PublicBaseURL:                envOr("SHORTWARDEN_PUBLIC_BASE_URL", "http://localhost:8080"),
		CookieDomain:                 os.Getenv("SHORTWARDEN_COOKIE_DOMAIN"),
		CookieSecure:                 envOr("SHORTWARDEN_COOKIE_SECURE", "0") == "1",
		CORSAllowedOrigins:           os.Getenv("SHORTWARDEN_CORS_ALLOWED_ORIGINS"),
		ScreenshotPreviewURLTemplate: os.Getenv("SHORTWARDEN_SCREENSHOT_PREVIEW_URL"),
		ScreenshotServiceURL:         os.Getenv("SHORTWARDEN_SCREENSHOT_SERVICE_URL"),
	}
	if cfg.PostgresDSN == "" {
		return Config{}, errors.New("SHORTWARDEN_POSTGRES_DSN must not be empty")
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("SHORTWARDEN_JWT_SECRET must not be empty")
	}
	if cfg.PublicBaseURL == "" {
		return Config{}, errors.New("SHORTWARDEN_PUBLIC_BASE_URL must not be empty")
	}
	return cfg, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

