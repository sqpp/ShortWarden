package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type RedirectCustomizationButton struct {
	Label string `json:"label"`
	Url   string `json:"url"`
}

type RedirectCustomization struct {
	DelaySeconds   int                           `json:"delay_seconds"`
	Mode           string                        `json:"mode"`
	ShowScreenshot bool                          `json:"show_screenshot"`
	CustomButtons  []RedirectCustomizationButton `json:"custom_buttons"`
}

func normalizeRedirectCustomization(in RedirectCustomization) RedirectCustomization {
	out := in
	if out.DelaySeconds < 0 {
		out.DelaySeconds = 0
	}
	if out.DelaySeconds > 30 {
		out.DelaySeconds = 30
	}
	mode := strings.ToLower(strings.TrimSpace(out.Mode))
	if mode != "click" {
		mode = "auto"
	}
	out.Mode = mode
	clean := make([]RedirectCustomizationButton, 0, len(out.CustomButtons))
	for _, b := range out.CustomButtons {
		label := strings.TrimSpace(b.Label)
		url := strings.TrimSpace(b.Url)
		if label == "" || url == "" {
			continue
		}
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			continue
		}
		if len(label) > 80 {
			label = label[:80]
		}
		if len(url) > 1024 {
			url = url[:1024]
		}
		clean = append(clean, RedirectCustomizationButton{Label: label, Url: url})
		if len(clean) >= 8 {
			break
		}
	}
	out.CustomButtons = clean
	return out
}

func (h *Handler) readRedirectCustomization(r *http.Request) (RedirectCustomization, error) {
	uid, ok := requireUserID(r)
	if !ok {
		return RedirectCustomization{}, pgx.ErrNoRows
	}
	row := h.db.QueryRow(r.Context(), `
SELECT redirect_delay_seconds, redirect_mode, redirect_show_screenshot, redirect_custom_buttons
FROM users
WHERE id = $1
`, uid)
	var out RedirectCustomization
	var raw []byte
	if err := row.Scan(&out.DelaySeconds, &out.Mode, &out.ShowScreenshot, &raw); err != nil {
		return RedirectCustomization{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out.CustomButtons)
	}
	return normalizeRedirectCustomization(out), nil
}

func (h *Handler) GetRedirectCustomization(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.readRedirectCustomization(r)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *Handler) UpdateRedirectCustomization(w http.ResponseWriter, r *http.Request) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req RedirectCustomization
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req = normalizeRedirectCustomization(req)
	raw, _ := json.Marshal(req.CustomButtons)
	if _, err := h.db.Exec(r.Context(), `
UPDATE users
SET redirect_delay_seconds = $2,
    redirect_mode = $3,
    redirect_show_screenshot = $4,
    redirect_custom_buttons = $5::jsonb
WHERE id = $1
`, uid, req.DelaySeconds, req.Mode, req.ShowScreenshot, string(raw)); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, req)
}

