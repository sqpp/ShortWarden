package httpapi

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"shortwarden/internal/screenshoturl"
)

// ScreenshotPreview proxies to the internal screenshotd service after validating the target URL.
// Browsers load this path on the public app origin; screenshotd stays on the Docker network.
func (h *Handler) ScreenshotPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	if raw == "" {
		writeError(w, http.StatusBadRequest, "missing url")
		return
	}
	if _, err := screenshoturl.ValidateTargetURL(raw); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	base := strings.TrimSpace(h.cfg.ScreenshotServiceURL)
	if base == "" {
		writeError(w, http.StatusServiceUnavailable, "screenshot service not configured")
		return
	}
	svcURL := strings.TrimRight(base, "/") + "/render?url=" + url.QueryEscape(raw)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(svcURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, "screenshot service unreachable")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadGateway, "screenshot capture failed")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.Copy(w, resp.Body)
}
