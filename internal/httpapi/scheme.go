package httpapi

import (
	"net/http"
	"net/url"
	"strings"
)

func requestScheme(r *http.Request) string {
	// Prefer reverse-proxy header
	if v := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); v != "" {
		// Can be "https,http" in some setups.
		if i := strings.IndexByte(v, ','); i >= 0 {
			v = v[:i]
		}
		v = strings.TrimSpace(v)
		if v == "http" || v == "https" {
			return v
		}
	}
	if r.URL != nil && r.URL.Scheme != "" {
		return r.URL.Scheme
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func schemeFromBaseURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "http"
	}
	if u.Scheme == "https" {
		return "https"
	}
	return "http"
}

